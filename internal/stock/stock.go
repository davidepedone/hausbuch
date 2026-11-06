// Package stock keeps track of what the household has at home, in sections such as medicines and the pantry, and of
// what is about to expire.
package stock

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"
)

// Section is a list of items kept apart from the others.
type Section string

const (
	Medicines Section = "medicines"
	Pantry    Section = "pantry"
)

// Sections lists every section, in the order they are shown.
var Sections = []Section{Medicines, Pantry}

// Label is the section's name as shown to users.
func (s Section) Label() string {
	if s == Medicines {
		return "Medicines"
	}
	return "Pantry"
}

var (
	ErrNotFound   = errors.New("no such item")
	ErrEmptyName  = errors.New("an item needs a name")
	ErrQuantity   = errors.New("the quantity must be a whole number of at least 1")
	ErrOpenedDays = errors.New("the days an item lasts once opened must be a whole number of at least 1")
)

// Values is what a user enters for a row. Dates are calendar dates at midnight UTC, zero when not given; OpenedDays is
// 0 when not given.
type Values struct {
	Name       string
	Quantity   int
	Unit       string
	Location   string
	ExpiresOn  time.Time
	OpenedOn   time.Time
	OpenedDays int
}

// Expiry is the date the item expires on, taking its opening into account, or zero when it has none.
func (v Values) Expiry() time.Time {
	if v.OpenedOn.IsZero() || v.OpenedDays == 0 {
		return v.ExpiresOn
	}
	opened := v.OpenedOn.AddDate(0, 0, v.OpenedDays)
	if v.ExpiresOn.IsZero() || opened.Before(v.ExpiresOn) {
		return opened
	}
	return v.ExpiresOn
}

// State tells whether an item has expired or is about to.
type State int

const (
	Fine State = iota
	Expiring
	Expired
)

// Item is a row of a section.
type Item struct {
	ID      int64
	Section Section
	Values
	CreatedByID int64
	CreatedBy   string // display name
	CreatedAt   time.Time
	ChangedBy   string // display name
	ChangedAt   time.Time
	State       State
	DaysLeft    int // days from today to the expiry date, when there is one
}

// Product is what a section remembers of a name once used.
type Product struct {
	Name       string
	Location   string
	OpenedDays int
}

// Store reads and writes items. Now is the clock, replaceable in tests; Lead is each section's lead time in days.
type Store struct {
	DB   *sql.DB
	Now  func() time.Time
	Lead map[Section]int
}

// DefaultLead returns each section's lead time in days when none is configured.
func DefaultLead() map[Section]int { return map[Section]int{Medicines: 30, Pantry: 7} }

// NewStore returns a Store on db using the real clock and the default lead times.
func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, Now: time.Now, Lead: DefaultLead()}
}

func nameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Date returns the calendar date of t in t's time zone, as stored.
func Date(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// today is the current date in the local time zone, which TZ sets.
func (s *Store) today() time.Time { return Date(s.Now().In(time.Local)) }

func check(v *Values) error {
	v.Name = strings.TrimSpace(v.Name)
	v.Unit = strings.TrimSpace(v.Unit)
	v.Location = strings.TrimSpace(v.Location)
	switch {
	case v.Name == "":
		return ErrEmptyName
	case v.Quantity < 1:
		return ErrQuantity
	case v.OpenedDays < 0:
		return ErrOpenedDays
	}
	return nil
}

// Add adds v to the section, by the user, and returns the id of its row: an unopened row with the same name and
// expiry date gets the quantity added, otherwise a new row is made.
func (s *Store) Add(ctx context.Context, section Section, userID int64, v Values) (int64, error) {
	return s.add(ctx, section, userID, v, userID, s.Now())
}

// Restore puts back a row that was removed, keeping who added it and when; it is added as Add does.
func (s *Store) Restore(ctx context.Context, section Section, userID int64, it Item) (int64, error) {
	return s.add(ctx, section, userID, it.Values, it.CreatedByID, it.CreatedAt)
}

func (s *Store) add(ctx context.Context, section Section, userID int64, v Values, createdBy int64, createdAt time.Time) (int64, error) {
	if err := check(&v); err != nil {
		return 0, err
	}
	now := s.Now().Unix()
	var id int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if v.OpenedOn.IsZero() {
			err := tx.QueryRowContext(ctx, `
				UPDATE stock_item SET quantity = quantity + ?, changed_by = ?, changed_at = ?
				WHERE id = (SELECT id FROM stock_item
					WHERE section = ? AND name_key = ? AND expires_on IS ? AND opened_on IS NULL ORDER BY id LIMIT 1)
				RETURNING id`,
				v.Quantity, userID, now, section, nameKey(v.Name), date(v.ExpiresOn)).Scan(&id)
			if err == nil {
				return remember(ctx, tx, section, v)
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO stock_item (section, name, name_key, quantity, unit, location, expires_on, opened_on,
				opened_days, created_by, created_at, changed_by, changed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			section, v.Name, nameKey(v.Name), v.Quantity, v.Unit, v.Location, date(v.ExpiresOn), date(v.OpenedOn),
			days(v.OpenedDays), createdBy, createdAt.Unix(), userID, now)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		return remember(ctx, tx, section, v)
	})
	return id, err
}

// Update replaces the values of a row of the section, changed by the user.
func (s *Store) Update(ctx context.Context, section Section, id, userID int64, v Values) error {
	if err := check(&v); err != nil {
		return err
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE stock_item SET name = ?, name_key = ?, quantity = ?, unit = ?, location = ?, expires_on = ?,
				opened_on = ?, opened_days = ?, changed_by = ?, changed_at = ?
			WHERE id = ? AND section = ?`,
			v.Name, nameKey(v.Name), v.Quantity, v.Unit, v.Location, date(v.ExpiresOn), date(v.OpenedOn),
			days(v.OpenedDays), userID, s.Now().Unix(), id, section)
		if err != nil {
			return err
		}
		if err := oneRow(res); err != nil {
			return err
		}
		return remember(ctx, tx, section, v)
	})
}

// remember records v's product as the section's latest use of its name.
func remember(ctx context.Context, tx *sql.Tx, section Section, v Values) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO stock_product (section, name_key, name, location, opened_days) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (section, name_key) DO UPDATE
		SET name = excluded.name, location = excluded.location, opened_days = excluded.opened_days`,
		section, nameKey(v.Name), v.Name, v.Location, days(v.OpenedDays))
	return err
}

// Increase adds one to a row's quantity.
func (s *Store) Increase(ctx context.Context, section Section, id, userID int64) error {
	res, err := s.DB.ExecContext(ctx,
		"UPDATE stock_item SET quantity = quantity + 1, changed_by = ?, changed_at = ? WHERE id = ? AND section = ?",
		userID, s.Now().Unix(), id, section)
	if err != nil {
		return err
	}
	return oneRow(res)
}

// Decrease takes one from a row's quantity. A row with quantity 1 is removed, and returned so that it can be restored.
// Both statements are relative to the stored quantity, so decreases made at the same time all count.
func (s *Store) Decrease(ctx context.Context, section Section, id, userID int64) (removed *Item, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE stock_item SET quantity = quantity - 1, changed_by = ?, changed_at = ?
			WHERE id = ? AND section = ? AND quantity > 1`, userID, s.Now().Unix(), id, section)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 1 {
			return err
		}
		it, err := s.remove(ctx, tx, section, id)
		removed = &it
		return err
	})
	return removed, err
}

// Remove deletes a row whatever its quantity, and returns it so that it can be restored.
func (s *Store) Remove(ctx context.Context, section Section, id int64) (Item, error) {
	var it Item
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var err error
		it, err = s.remove(ctx, tx, section, id)
		return err
	})
	return it, err
}

func (s *Store) remove(ctx context.Context, tx *sql.Tx, section Section, id int64) (Item, error) {
	it, err := s.scan(tx.QueryRowContext(ctx, selectItems+" WHERE i.id = ? AND i.section = ?", id, section))
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM stock_item WHERE id = ?", id)
	return it, err
}

// Open marks a row as opened today. Of a row with more than one unit, one is split off into a row of its own, opened.
func (s *Store) Open(ctx context.Context, section Section, id, userID int64) error {
	today, now := date(s.today()), s.Now().Unix()
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var quantity int
		err := tx.QueryRowContext(ctx, "SELECT quantity FROM stock_item WHERE id = ? AND section = ?", id, section).
			Scan(&quantity)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if quantity == 1 {
			_, err = tx.ExecContext(ctx,
				"UPDATE stock_item SET opened_on = ?, changed_by = ?, changed_at = ? WHERE id = ?",
				today, userID, now, id)
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE stock_item SET quantity = quantity - 1, changed_by = ?, changed_at = ? WHERE id = ?",
			userID, now, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO stock_item (section, name, name_key, quantity, unit, location, expires_on, opened_on,
				opened_days, created_by, created_at, changed_by, changed_at)
			SELECT section, name, name_key, 1, unit, location, expires_on, ?, opened_days, ?, ?, ?, ?
			FROM stock_item WHERE id = ?`, today, userID, now, userID, now, id)
		return err
	})
}

const selectItems = `
	SELECT i.id, i.section, i.name, i.quantity, i.unit, i.location, i.expires_on, i.opened_on, i.opened_days,
		i.created_by, cu.display_name, i.created_at, chu.display_name, i.changed_at
	FROM stock_item i JOIN app_user cu ON cu.id = i.created_by JOIN app_user chu ON chu.id = i.changed_by`

func (s *Store) scan(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	var expires, opened sql.NullString
	var openedDays sql.NullInt64
	var createdAt, changedAt int64
	if err := row.Scan(&it.ID, &it.Section, &it.Name, &it.Quantity, &it.Unit, &it.Location, &expires, &opened,
		&openedDays, &it.CreatedByID, &it.CreatedBy, &createdAt, &it.ChangedBy, &changedAt); err != nil {
		return Item{}, err
	}
	it.ExpiresOn, it.OpenedOn = parseDate(expires), parseDate(opened)
	it.OpenedDays = int(openedDays.Int64)
	it.CreatedAt, it.ChangedAt = time.Unix(createdAt, 0), time.Unix(changedAt, 0)
	it.State, it.DaysLeft = s.state(it)
	return it, nil
}

// state tells whether the item has expired, on the day after its expiry date, or is expiring, within its section's
// lead time of it.
func (s *Store) state(it Item) (State, int) {
	expiry := it.Expiry()
	if expiry.IsZero() {
		return Fine, 0
	}
	left := int(expiry.Sub(s.today()).Hours() / 24)
	switch {
	case left < 0:
		return Expired, left
	case left <= s.Lead[it.Section]:
		return Expiring, left
	}
	return Fine, left
}

func (s *Store) query(ctx context.Context, query string, args ...any) ([]Item, error) {
	rows, err := s.DB.QueryContext(ctx, selectItems+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		it, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Get returns a row of the section.
func (s *Store) Get(ctx context.Context, section Section, id int64) (Item, error) {
	it, err := s.scan(s.DB.QueryRowContext(ctx, selectItems+" WHERE i.id = ? AND i.section = ?", id, section))
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	return it, err
}

// List returns the rows of the section by name, rows with the same name by expiry date, those without one last.
func (s *Store) List(ctx context.Context, section Section) ([]Item, error) {
	items, err := s.query(ctx, " WHERE i.section = ? ORDER BY i.id", section)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(items, func(a, b Item) int {
		return cmp.Or(strings.Compare(nameKey(a.Name), nameKey(b.Name)), compareExpiry(a, b))
	})
	return items, nil
}

// Warnings returns the expired and expiring rows of every section: expired first, then by expiry date.
func (s *Store) Warnings(ctx context.Context) ([]Item, error) {
	items, err := s.query(ctx, " ORDER BY i.id")
	if err != nil {
		return nil, err
	}
	items = slices.DeleteFunc(items, func(it Item) bool { return it.State == Fine })
	slices.SortStableFunc(items, func(a, b Item) int {
		return cmp.Or(cmp.Compare(b.State, a.State), compareExpiry(a, b))
	})
	return items, nil
}

// compareExpiry orders by expiry date, items without one last.
func compareExpiry(a, b Item) int {
	ea, eb := a.Expiry(), b.Expiry()
	switch {
	case ea.IsZero() && eb.IsZero():
		return 0
	case ea.IsZero():
		return 1
	case eb.IsZero():
		return -1
	}
	return ea.Compare(eb)
}

// Names returns the names the section has used, by name.
func (s *Store) Names(ctx context.Context, section Section) ([]string, error) {
	return s.strings(ctx, "SELECT name FROM stock_product WHERE section = ? ORDER BY name_key", section)
}

// Locations returns the locations the section's products were last kept in, without repetitions.
func (s *Store) Locations(ctx context.Context, section Section) ([]string, error) {
	return s.strings(ctx,
		"SELECT DISTINCT location FROM stock_product WHERE section = ? AND location != '' ORDER BY lower(location)",
		section)
}

func (s *Store) strings(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// Product returns what the section remembers of a name, ignoring case.
func (s *Store) Product(ctx context.Context, section Section, name string) (Product, bool, error) {
	var p Product
	var openedDays sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		"SELECT name, location, opened_days FROM stock_product WHERE section = ? AND name_key = ?",
		section, nameKey(name)).Scan(&p.Name, &p.Location, &openedDays)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, false, nil
	}
	p.OpenedDays = int(openedDays.Int64)
	return p, err == nil, err
}

func oneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) inTx(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	if err := f(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// date is a calendar date as stored, or NULL for the zero time.
func date(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.DateOnly)
}

func parseDate(s sql.NullString) time.Time {
	t, _ := time.Parse(time.DateOnly, s.String)
	return t
}

// days is a number of days as stored, or NULL for 0.
func days(n int) any {
	if n == 0 {
		return nil
	}
	return n
}
