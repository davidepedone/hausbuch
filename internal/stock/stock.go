// Package stock keeps track of what the household has at home, in sections such as medicines and the pantry, and of
// what is about to expire.
package stock

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Section is a list of items kept apart from the others.
type Section string

const (
	Medicines Section = "medicines"
	Pantry    Section = "pantry"
	Supplies  Section = "supplies"
)

// Sections lists every section, in the order they are shown.
var Sections = []Section{Medicines, Pantry, Supplies}

// Label is the section's name as shown to users.
func (s Section) Label() string {
	switch s {
	case Medicines:
		return "Medicines"
	case Pantry:
		return "Pantry"
	case Supplies:
		return "Supplies"
	}
	return string(s)
}

var (
	ErrNotFound   = errors.New("no such item")
	ErrEmptyName  = errors.New("an item needs a name")
	ErrQuantity   = errors.New("the quantity must be above 0, with at most one decimal")
	ErrOpenedDays = errors.New("the days an item lasts once opened must be a whole number of at least 1")
)

// Quantity is an amount in tenths, so that it can have one decimal: 15 is 1.5.
type Quantity int

// One is a quantity of one.
const One Quantity = 10

// ParseQuantity reads a quantity above 0 with at most one decimal, after either a dot or a comma.
func ParseQuantity(s string) (Quantity, error) {
	whole, tenth, _ := strings.Cut(strings.Replace(strings.TrimSpace(s), ",", ".", 1), ".")
	if len(tenth) > 1 {
		return 0, ErrQuantity
	}
	n, err := strconv.Atoi(whole + tenth + strings.Repeat("0", 1-len(tenth)))
	if err != nil || n <= 0 {
		return 0, ErrQuantity
	}
	return Quantity(n), nil
}

// String writes the quantity with a dot before its decimal, and without one when it is whole.
func (q Quantity) String() string {
	if q%One == 0 {
		return strconv.Itoa(int(q / One))
	}
	return strconv.Itoa(int(q/One)) + "." + strconv.Itoa(int(q%One))
}

// Values is what a user enters for a row. Dates are calendar dates at midnight UTC, zero when not given; OpenedDays is
// 0 when not given.
type Values struct {
	Name       string
	Quantity   Quantity
	Unit       string
	Category   string
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
	Category   string
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
func DefaultLead() map[Section]int { return map[Section]int{Medicines: 30, Pantry: 7, Supplies: 7} }

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
	v.Category = strings.TrimSpace(v.Category)
	v.Location = strings.TrimSpace(v.Location)
	switch {
	case v.Name == "":
		return ErrEmptyName
	case v.Quantity <= 0:
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
				UPDATE stock_item SET quantity_tenths = quantity_tenths + ?, changed_by = ?, changed_at = ?
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
			INSERT INTO stock_item (section, name, name_key, quantity_tenths, unit, category, location, expires_on,
				opened_on, opened_days, created_by, created_at, changed_by, changed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			section, v.Name, nameKey(v.Name), v.Quantity, v.Unit, v.Category, v.Location, date(v.ExpiresOn), date(v.OpenedOn),
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
			UPDATE stock_item SET name = ?, name_key = ?, quantity_tenths = ?, unit = ?, category = ?, location = ?,
				expires_on = ?, opened_on = ?, opened_days = ?, changed_by = ?, changed_at = ?
			WHERE id = ? AND section = ?`,
			v.Name, nameKey(v.Name), v.Quantity, v.Unit, v.Category, v.Location, date(v.ExpiresOn), date(v.OpenedOn),
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
		INSERT INTO stock_product (section, name_key, name, category, location, opened_days) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (section, name_key) DO UPDATE
		SET name = excluded.name, category = excluded.category, location = excluded.location,
			opened_days = excluded.opened_days`,
		section, nameKey(v.Name), v.Name, v.Category, v.Location, days(v.OpenedDays))
	return err
}

// Increase adds one to a row's quantity.
func (s *Store) Increase(ctx context.Context, section Section, id, userID int64) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE stock_item SET quantity_tenths = quantity_tenths + ?, changed_by = ?, changed_at = ?
		WHERE id = ? AND section = ?`, One, userID, s.Now().Unix(), id, section)
	if err != nil {
		return err
	}
	return oneRow(res)
}

// Decrease takes one from a row's quantity. A row with a quantity of 1 or less is removed, and returned so that it
// can be restored.
// Both statements are relative to the stored quantity, so decreases made at the same time all count.
func (s *Store) Decrease(ctx context.Context, section Section, id, userID int64) (removed *Item, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE stock_item SET quantity_tenths = quantity_tenths - ?, changed_by = ?, changed_at = ?
			WHERE id = ? AND section = ? AND quantity_tenths > ?`, One, userID, s.Now().Unix(), id, section, One)
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

// Open marks a row as opened today. Of a row with a quantity above 1, one is split off into a row of its own, opened.
func (s *Store) Open(ctx context.Context, section Section, id, userID int64) error {
	today, now := date(s.today()), s.Now().Unix()
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var quantity Quantity
		err := tx.QueryRowContext(ctx, "SELECT quantity_tenths FROM stock_item WHERE id = ? AND section = ?", id, section).
			Scan(&quantity)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if quantity <= One {
			_, err = tx.ExecContext(ctx,
				"UPDATE stock_item SET opened_on = ?, changed_by = ?, changed_at = ? WHERE id = ?",
				today, userID, now, id)
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE stock_item SET quantity_tenths = quantity_tenths - ?, changed_by = ?, changed_at = ? WHERE id = ?",
			One, userID, now, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO stock_item (section, name, name_key, quantity_tenths, unit, category, location, expires_on,
				opened_on, opened_days, created_by, created_at, changed_by, changed_at)
			SELECT section, name, name_key, ?, unit, category, location, expires_on, ?, opened_days, ?, ?, ?, ?
			FROM stock_item WHERE id = ?`, One, today, userID, now, userID, now, id)
		return err
	})
}

const selectItems = `
	SELECT i.id, i.section, i.name, i.quantity_tenths, i.unit, i.category, i.location, i.expires_on, i.opened_on,
		i.opened_days, i.created_by, cu.display_name, i.created_at, chu.display_name, i.changed_at
	FROM stock_item i JOIN app_user cu ON cu.id = i.created_by JOIN app_user chu ON chu.id = i.changed_by`

func (s *Store) scan(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	var expires, opened sql.NullString
	var openedDays sql.NullInt64
	var createdAt, changedAt int64
	if err := row.Scan(&it.ID, &it.Section, &it.Name, &it.Quantity, &it.Unit, &it.Category, &it.Location, &expires, &opened,
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

// Group is the rows of a listed section with one category; Key is the category lower-cased, "" for rows without one,
// and Label its spelling in the group's first row.
type Group struct {
	Key, Label string
	Items      []Item
}

// GroupItems splits a section's rows by category, in alphabetical order with the rows without one last, keeping their
// order within a group. Rows without any category make one group with an empty key.
func GroupItems(items []Item) []Group {
	var groups []Group
	for _, it := range items {
		key := nameKey(it.Category)
		i := slices.IndexFunc(groups, func(g Group) bool { return g.Key == key })
		if i < 0 {
			groups = append(groups, Group{Key: key, Label: it.Category})
			i = len(groups) - 1
		}
		groups[i].Items = append(groups[i].Items, it)
	}
	slices.SortFunc(groups, func(a, b Group) int {
		if (a.Key == "") != (b.Key == "") {
			return strings.Compare(b.Key, a.Key) // the empty key last
		}
		return strings.Compare(a.Key, b.Key)
	})
	return groups
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

// Categories returns the categories the section's products last had, without repetitions ignoring case.
func (s *Store) Categories(ctx context.Context, section Section) ([]string, error) {
	list, err := s.strings(ctx, "SELECT DISTINCT category FROM stock_product WHERE section = ? AND category != ''", section)
	slices.SortStableFunc(list, func(a, b string) int { return strings.Compare(nameKey(a), nameKey(b)) })
	return slices.CompactFunc(list, func(a, b string) bool { return nameKey(a) == nameKey(b) }), err
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
		"SELECT name, category, location, opened_days FROM stock_product WHERE section = ? AND name_key = ?",
		section, nameKey(name)).Scan(&p.Name, &p.Category, &p.Location, &openedDays)
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
