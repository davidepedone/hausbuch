package stock

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
)

// Entry is a line of the household's shopping list.
type Entry struct {
	ID       int64
	Name     string
	Section  Section  // where it is put away once bought, "" for none
	Quantity Quantity // 0 when not given
	Unit     string
	Shop     string // "" for none
	Bought   bool
}

// EntryGroup is the entries to buy that go into one section, or into none when Section is "".
type EntryGroup struct {
	Section Section
	Entries []Entry
}

// Label is the group's heading.
func (g EntryGroup) Label() string {
	if g.Section == "" {
		return "Other"
	}
	return g.Section.Label()
}

// Chip is a shop the list's entries are bought at: Key is its name lower-cased, Label its first spelling.
type Chip struct{ Key, Label string }

// ShopKey is the key of a shop's chip.
func ShopKey(shop string) string { return nameKey(shop) }

// AddEntry adds e to the list, to buy, by the user. Without a shop it takes the one its name was last listed with. An
// entry to buy with the same name and section gets e merged into it instead.
func (s *Store) AddEntry(ctx context.Context, userID int64, e Entry) error {
	e.Name, e.Unit, e.Shop = strings.TrimSpace(e.Name), strings.TrimSpace(e.Unit), strings.TrimSpace(e.Shop)
	switch {
	case e.Name == "":
		return ErrEmptyName
	case e.Quantity < 0:
		return ErrQuantity
	case e.Section != "" && !slices.Contains(Sections, e.Section):
		return ErrSection
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if e.Shop == "" {
			err := tx.QueryRowContext(ctx, "SELECT shop FROM shopping_product WHERE name_key = ?", nameKey(e.Name)).
				Scan(&e.Shop)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		} else if _, err := tx.ExecContext(ctx, `
			INSERT INTO shopping_product (name_key, name, shop) VALUES (?, ?, ?)
			ON CONFLICT (name_key) DO UPDATE SET name = excluded.name, shop = excluded.shop`,
			nameKey(e.Name), e.Name, e.Shop); err != nil {
			return err
		}
		merged, err := mergeEntry(ctx, tx, e, false)
		if err != nil || merged {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO shopping_entry (name, name_key, section, quantity_tenths, unit, shop, created_by, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			e.Name, nameKey(e.Name), e.Section, quantity(e.Quantity), e.Unit, e.Shop, userID, s.Now().Unix())
		return err
	})
}

// mergeEntry merges e into the other entry with its name and section that is bought, or not, as given, when there is
// one: quantities given on both are added, otherwise the one given is kept, with its unit; e's shop wins when it has
// one. It reports whether it did.
func mergeEntry(ctx context.Context, tx *sql.Tx, e Entry, bought bool) (bool, error) {
	var into Entry
	var q sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT id, quantity_tenths, unit, shop FROM shopping_entry
		WHERE name_key = ? AND section = ? AND bought = ? AND id != ? ORDER BY id LIMIT 1`,
		nameKey(e.Name), e.Section, bought, e.ID).Scan(&into.ID, &q, &into.Unit, &into.Shop)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	into.Quantity = Quantity(q.Int64)
	switch {
	case into.Quantity > 0 && e.Quantity > 0:
		into.Quantity += e.Quantity
	case into.Quantity == 0:
		into.Quantity, into.Unit = e.Quantity, e.Unit
	}
	into.Shop = cmp.Or(e.Shop, into.Shop)
	_, err = tx.ExecContext(ctx, "UPDATE shopping_entry SET quantity_tenths = ?, unit = ?, shop = ? WHERE id = ?",
		quantity(into.Quantity), into.Unit, into.Shop, into.ID)
	return true, err
}

// Entries returns the list: the entries to buy, then the bought ones, each by section in the sections' order, those
// without one last, then by name.
func (s *Store) Entries(ctx context.Context) ([]Entry, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT id, name, section, quantity_tenths, unit, shop, bought FROM shopping_entry ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Entry
	for rows.Next() {
		var e Entry
		var q sql.NullInt64
		if err := rows.Scan(&e.ID, &e.Name, &e.Section, &q, &e.Unit, &e.Shop, &e.Bought); err != nil {
			return nil, err
		}
		e.Quantity = Quantity(q.Int64)
		list = append(list, e)
	}
	slices.SortStableFunc(list, func(a, b Entry) int {
		return cmp.Or(compareBool(a.Bought, b.Bought), cmp.Compare(sectionOrder(a.Section), sectionOrder(b.Section)),
			strings.Compare(nameKey(a.Name), nameKey(b.Name)))
	})
	return list, rows.Err()
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

// sectionOrder places a section in the sections' order, no section last.
func sectionOrder(section Section) int {
	if i := slices.Index(Sections, section); i >= 0 {
		return i
	}
	return len(Sections)
}

// GroupEntries splits the list as Entries orders it into the groups of entries to buy and the bought entries.
func GroupEntries(entries []Entry) (toBuy []EntryGroup, bought []Entry) {
	for _, e := range entries {
		switch {
		case e.Bought:
			bought = append(bought, e)
		case len(toBuy) > 0 && toBuy[len(toBuy)-1].Section == e.Section:
			toBuy[len(toBuy)-1].Entries = append(toBuy[len(toBuy)-1].Entries, e)
		default:
			toBuy = append(toBuy, EntryGroup{Section: e.Section, Entries: []Entry{e}})
		}
	}
	return toBuy, bought
}

// ShopChips returns the shops of the entries, one per name ignoring case, in alphabetical order.
func ShopChips(entries []Entry) []Chip {
	var chips []Chip
	for _, e := range entries {
		key := ShopKey(e.Shop)
		if key != "" && !slices.ContainsFunc(chips, func(c Chip) bool { return c.Key == key }) {
			chips = append(chips, Chip{Key: key, Label: e.Shop})
		}
	}
	slices.SortFunc(chips, func(a, b Chip) int { return strings.Compare(a.Key, b.Key) })
	return chips
}

// Tick marks an entry as bought, or as still to buy, merging it into the entry of that kind with its name and section
// when there is one.
func (s *Store) Tick(ctx context.Context, id int64, bought bool) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		e := Entry{ID: id}
		var q sql.NullInt64
		err := tx.QueryRowContext(ctx,
			"SELECT name, section, quantity_tenths, unit, shop, bought FROM shopping_entry WHERE id = ?", id).
			Scan(&e.Name, &e.Section, &q, &e.Unit, &e.Shop, &e.Bought)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil || e.Bought == bought {
			return err
		}
		e.Quantity = Quantity(q.Int64)
		merged, err := mergeEntry(ctx, tx, e, bought)
		if err != nil {
			return err
		}
		if merged {
			_, err = tx.ExecContext(ctx, "DELETE FROM shopping_entry WHERE id = ?", id)
		} else {
			_, err = tx.ExecContext(ctx, "UPDATE shopping_entry SET bought = ? WHERE id = ?", bought, id)
		}
		return err
	})
}

// RemoveEntry takes an entry off the list.
func (s *Store) RemoveEntry(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, "DELETE FROM shopping_entry WHERE id = ?", id)
	if err != nil {
		return err
	}
	return oneRow(res)
}

// ClearBought takes off the list the bought entries without a section.
func (s *Store) ClearBought(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM shopping_entry WHERE bought = 1 AND section = ''")
	return err
}

// Entry returns an entry of the list.
func (s *Store) Entry(ctx context.Context, id int64) (Entry, error) {
	e := Entry{ID: id}
	var q sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		"SELECT name, section, quantity_tenths, unit, shop, bought FROM shopping_entry WHERE id = ?", id).
		Scan(&e.Name, &e.Section, &q, &e.Unit, &e.Shop, &e.Bought)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	e.Quantity = Quantity(q.Int64)
	return e, err
}

// PutAway adds v, by the user, to the section of a bought entry, as Add does, and takes the entry off the list, both or
// neither. It returns the id of the row that got v; an entry not bought or without a section is ErrNotFound.
func (s *Store) PutAway(ctx context.Context, id, userID int64, v Values) (int64, error) {
	if err := check(&v); err != nil {
		return 0, err
	}
	var row int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var section Section
		err := tx.QueryRowContext(ctx,
			"SELECT section FROM shopping_entry WHERE id = ? AND bought = 1 AND section != ''", id).Scan(&section)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if row, err = s.addIn(ctx, tx, section, userID, v, userID, s.Now()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM shopping_entry WHERE id = ?", id)
		return err
	})
	return row, err
}

// AddRowToList adds a row's product to the list, by the user, with the row's name, unit and section and no quantity.
func (s *Store) AddRowToList(ctx context.Context, section Section, id, userID int64) (Item, error) {
	it, err := s.Get(ctx, section, id)
	if err != nil {
		return Item{}, err
	}
	return it, s.AddEntry(ctx, userID, Entry{Name: it.Name, Section: section, Unit: it.Unit})
}

// ListNames returns the names used in every section and on the list, one per name ignoring case, by name.
func (s *Store) ListNames(ctx context.Context) ([]string, error) {
	return s.distinct(ctx, "SELECT name FROM stock_product UNION ALL SELECT name FROM shopping_product")
}

// Shops returns the shops used on the list, one per name ignoring case, by name.
func (s *Store) Shops(ctx context.Context) ([]string, error) {
	return s.distinct(ctx, "SELECT shop FROM shopping_product UNION ALL SELECT shop FROM shopping_entry WHERE shop != ''")
}

// distinct returns the strings the query selects, one per value ignoring case, sorted that way.
func (s *Store) distinct(ctx context.Context, query string) ([]string, error) {
	list, err := s.strings(ctx, query)
	slices.SortStableFunc(list, func(a, b string) int { return strings.Compare(nameKey(a), nameKey(b)) })
	return slices.CompactFunc(list, func(a, b string) bool { return nameKey(a) == nameKey(b) }), err
}

// ListProduct returns what an entry named name can be filled in with: the section the name is used in, when it is used
// in exactly one, and the shop it was last listed with.
func (s *Store) ListProduct(ctx context.Context, name string) (Section, string, error) {
	sections, err := s.strings(ctx, "SELECT section FROM stock_product WHERE name_key = ?", nameKey(name))
	if err != nil {
		return "", "", err
	}
	var section Section
	if len(sections) == 1 {
		section = Section(sections[0])
	}
	var shop string
	err = s.DB.QueryRowContext(ctx, "SELECT shop FROM shopping_product WHERE name_key = ?", nameKey(name)).Scan(&shop)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return section, shop, err
}

// quantity is a quantity as stored on the list, or NULL for 0.
func quantity(q Quantity) any {
	if q == 0 {
		return nil
	}
	return q
}
