// Package page holds the wiki's pages, their revisions and the links between them.
package page

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"strings"
	"time"

	"github.com/davidepedone/hausbuch/internal/markdown"
	"github.com/davidepedone/hausbuch/internal/search"
)

// Type is the kind of content a page holds.
type Type string

const (
	Blank     Type = "blank"
	Recipe    Type = "recipe"
	HowTo     Type = "howto"
	WhereIsIt Type = "where"
)

// Types lists every page type, in the order they are offered.
var Types = []Type{Blank, Recipe, HowTo, WhereIsIt}

// Label is the type's name as shown to users.
func (t Type) Label() string {
	switch t {
	case Recipe:
		return "Recipe"
	case HowTo:
		return "How-to"
	case WhereIsIt:
		return "Where is it"
	}
	return "Blank"
}

//go:embed pagetypes
var skeletons embed.FS

// Skeleton returns the text a new page of type t in language lang starts with: empty for Blank.
func Skeleton(t Type, lang string) string {
	b, err := skeletons.ReadFile("pagetypes/" + string(t) + "." + lang + ".md")
	if err != nil {
		return ""
	}
	return string(b)
}

// Languages lists the languages a page can be written in, as codes.
var Languages = []string{"en", "it"}

// LanguageName returns the name of a language code as shown to users.
func LanguageName(code string) string {
	if code == "it" {
		return "Italian"
	}
	return "English"
}

// DeletedLifetime is how long a deleted page can be restored before it is removed for good.
const DeletedLifetime = 30 * 24 * time.Hour

var (
	ErrNotFound   = errors.New("no such page")
	ErrEmptyTitle = errors.New("a page needs a title")
)

// TitleTakenError means another page already has the title, as its title or as a former one.
type TitleTakenError struct {
	PageID  int64
	Title   string // the other page's current title
	Deleted bool
}

func (e *TitleTakenError) Error() string { return "the title is taken by page " + e.Title }

// Content is what a user writes in the editor.
type Content struct {
	Title    string
	Type     Type
	Language string
	Body     string
}

// Page is a page with its current content.
type Page struct {
	ID       int64
	Revision int
	Content
	CreatedBy string    // display name of the author of the first revision
	ChangedAt time.Time // time of the current revision
	ChangedBy string    // display name of its author
	DeletedAt time.Time // zero unless deleted
	DeletedBy string
}

// RemovedAt is when a deleted page will be removed for good.
func (p Page) RemovedAt() time.Time { return p.DeletedAt.Add(DeletedLifetime) }

// Store reads and writes pages. Now is the clock, replaceable in tests.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a Store on db using the real clock.
func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, Now: time.Now}
}

func titleKey(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

// Create adds a page with c as its first revision, authored by the user, and returns its id.
func (s *Store) Create(ctx context.Context, userID int64, c Content) (int64, error) {
	c.Title = strings.TrimSpace(c.Title)
	if c.Title == "" {
		return 0, ErrEmptyTitle
	}
	if err := s.RemoveExpired(ctx); err != nil {
		return 0, err
	}
	var id int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := titleFree(ctx, tx, c.Title, 0); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO page (title, title_key, type, language, revision) VALUES (?, ?, ?, ?, 0)",
			c.Title, titleKey(c.Title), c.Type, c.Language)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		return s.appendRevision(ctx, tx, id, userID, c)
	})
	return id, err
}

// ConflictError means the page changed since the revision a save was based on.
type ConflictError struct {
	Revision int      // the current revision
	By       []string // display names of who saved since, in order
}

func (e *ConflictError) Error() string { return "the page changed meanwhile" }

// Update saves c as the page's new revision, authored by the user, unless it changes nothing. base is the revision the
// user started editing from: if the page has changed since, nothing is saved and the error is a *ConflictError. A
// rename keeps the former title as an alias, so links using it keep leading to the page.
func (s *Store) Update(ctx context.Context, id, userID int64, base int, c Content) error {
	c.Title = strings.TrimSpace(c.Title)
	if c.Title == "" {
		return ErrEmptyTitle
	}
	if err := s.RemoveExpired(ctx); err != nil {
		return err
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var oldKey string
		var current int
		var old Content
		err := tx.QueryRowContext(ctx, `
			SELECT p.title_key, p.revision, r.title, r.type, r.language, r.body
			FROM page p JOIN page_revision r ON r.page_id = p.id AND r.number = p.revision
			WHERE p.id = ? AND p.deleted_at IS NULL`, id).
			Scan(&oldKey, &current, &old.Title, &old.Type, &old.Language, &old.Body)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if current != base {
			return conflict(ctx, tx, id, base, current)
		}
		if c == old {
			return nil
		}
		if newKey := titleKey(c.Title); newKey != oldKey {
			if err := titleFree(ctx, tx, c.Title, id); err != nil {
				return err
			}
			// Renaming back to a former title makes it the title again rather than an alias.
			if _, err := tx.ExecContext(ctx, "DELETE FROM page_alias WHERE title_key = ?", newKey); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO page_alias (title_key, page_id) VALUES (?, ?)",
				oldKey, id); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE page SET title = ?, title_key = ?, type = ?, language = ? WHERE id = ?",
			c.Title, titleKey(c.Title), c.Type, c.Language, id); err != nil {
			return err
		}
		return s.appendRevision(ctx, tx, id, userID, c)
	})
}

func conflict(ctx context.Context, tx *sql.Tx, id int64, base, current int) error {
	e := &ConflictError{Revision: current}
	rows, err := tx.QueryContext(ctx, `
		SELECT u.display_name FROM page_revision r JOIN app_user u ON u.id = r.author_id
		WHERE r.page_id = ? AND r.number > ? GROUP BY u.id ORDER BY MIN(r.number)`, id, base)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		e.By = append(e.By, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return e
}

// titleFree returns a *TitleTakenError if a page other than self has the title, as its title or a former one.
func titleFree(ctx context.Context, tx *sql.Tx, title string, self int64) error {
	e := &TitleTakenError{}
	var deletedAt sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT p.id, p.title, p.deleted_at FROM page p
		WHERE p.id != ? AND (p.title_key = ? OR p.id IN (SELECT page_id FROM page_alias WHERE title_key = ?))`,
		self, titleKey(title), titleKey(title)).Scan(&e.PageID, &e.Title, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	e.Deleted = deletedAt.Valid
	return e
}

// appendRevision stores c as the page's next revision and rewrites the page's outgoing links.
func (s *Store) appendRevision(ctx context.Context, tx *sql.Tx, id, userID int64, c Content) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO page_revision (page_id, number, title, type, language, body, author_id, created_at)
		SELECT id, revision + 1, ?, ?, ?, ?, ?, ? FROM page WHERE id = ?`,
		c.Title, c.Type, c.Language, c.Body, userID, s.Now().Unix(), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE page SET revision = revision + 1 WHERE id = ?", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM page_link WHERE page_id = ?", id); err != nil {
		return err
	}
	for _, title := range markdown.Links(c.Body) {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO page_link (page_id, title_key) VALUES (?, ?)",
			id, titleKey(title)); err != nil {
			return err
		}
	}
	return index(ctx, tx, id)
}

// index brings the page's entry in the full-text index up to date: its current content, or none if it is deleted.
func index(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM page_fts WHERE rowid = ?", id); err != nil {
		return err
	}
	var title, body, lang string
	err := tx.QueryRowContext(ctx, `
		SELECT p.title, r.body, p.language
		FROM page p JOIN page_revision r ON r.page_id = p.id AND r.number = p.revision
		WHERE p.id = ? AND p.deleted_at IS NULL`, id).Scan(&title, &body, &lang)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO page_fts (rowid, title_stem, body_stem) VALUES (?, ?, ?)",
		id, search.Stems(title, lang), search.Stems(body, lang))
	return err
}

// Reindex rebuilds the full-text index from scratch, so that it holds pages saved before it existed and stems from
// the stemmers of this version.
func (s *Store) Reindex(ctx context.Context) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM page_fts"); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, "SELECT id FROM page WHERE deleted_at IS NULL")
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, id := range ids {
			if err := index(ctx, tx, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// The current content of pages, with who created and last changed them and who deleted them.
const selectPages = `
	SELECT p.id, p.revision, p.title, p.type, p.language, r.body, cu.display_name, r.created_at, ru.display_name,
		p.deleted_at, COALESCE(du.display_name, '')
	FROM page p
	JOIN page_revision r ON r.page_id = p.id AND r.number = p.revision
	JOIN app_user ru ON ru.id = r.author_id
	JOIN page_revision c ON c.page_id = p.id AND c.number = 1
	JOIN app_user cu ON cu.id = c.author_id
	LEFT JOIN app_user du ON du.id = p.deleted_by`

func scanPage(row interface{ Scan(...any) error }) (Page, error) {
	var p Page
	var changedAt int64
	var deletedAt sql.NullInt64
	if err := row.Scan(&p.ID, &p.Revision, &p.Title, &p.Type, &p.Language, &p.Body, &p.CreatedBy, &changedAt,
		&p.ChangedBy, &deletedAt, &p.DeletedBy); err != nil {
		return Page{}, err
	}
	p.ChangedAt = time.Unix(changedAt, 0)
	if deletedAt.Valid {
		p.DeletedAt = time.Unix(deletedAt.Int64, 0)
	}
	return p, nil
}

func (s *Store) queryPages(ctx context.Context, query string, args ...any) ([]Page, error) {
	rows, err := s.DB.QueryContext(ctx, selectPages+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// Get returns the page with that id, unless it is deleted.
func (s *Store) Get(ctx context.Context, id int64) (Page, error) {
	p, err := scanPage(s.DB.QueryRowContext(ctx, selectPages+" WHERE p.id = ? AND p.deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrNotFound
	}
	return p, err
}

// List returns the pages that aren't deleted, most recently changed first, only those of type t unless it is empty.
func (s *Store) List(ctx context.Context, t Type) ([]Page, error) {
	return s.queryPages(ctx, " WHERE p.deleted_at IS NULL AND (? = '' OR p.type = ?) ORDER BY r.created_at DESC, r.rowid DESC",
		t, t)
}

// Resolve returns the id of the page that isn't deleted and has the title, as its title or a former one.
func (s *Store) Resolve(ctx context.Context, title string) (id int64, found bool, err error) {
	err = s.DB.QueryRowContext(ctx, `
		SELECT id FROM page
		WHERE deleted_at IS NULL AND (title_key = ? OR id IN (SELECT page_id FROM page_alias WHERE title_key = ?))`,
		titleKey(title), titleKey(title)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

// Backlinks returns the pages that aren't deleted and link to the page, by its title or a former one, by title.
func (s *Store) Backlinks(ctx context.Context, id int64) ([]Page, error) {
	return s.queryPages(ctx, ` WHERE p.deleted_at IS NULL AND p.id != ? AND p.id IN (
			SELECT l.page_id FROM page_link l
			WHERE l.title_key IN (SELECT title_key FROM page WHERE id = ? UNION SELECT title_key FROM page_alias WHERE page_id = ?))
		ORDER BY p.title_key`, id, id, id)
}

// Revision is a saved version of a page.
type Revision struct {
	Number int
	Content
	Author    string // display name
	CreatedAt time.Time
}

const selectRevisions = `
	SELECT r.number, r.title, r.type, r.language, r.body, u.display_name, r.created_at
	FROM page_revision r JOIN app_user u ON u.id = r.author_id WHERE r.page_id = ?`

func scanRevision(row interface{ Scan(...any) error }) (Revision, error) {
	var r Revision
	var createdAt int64
	err := row.Scan(&r.Number, &r.Title, &r.Type, &r.Language, &r.Body, &r.Author, &createdAt)
	r.CreatedAt = time.Unix(createdAt, 0)
	return r, err
}

// Revisions returns every revision of the page, newest first.
func (s *Store) Revisions(ctx context.Context, id int64) ([]Revision, error) {
	rows, err := s.DB.QueryContext(ctx, selectRevisions+" ORDER BY r.number DESC", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var revs []Revision
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, r)
	}
	return revs, rows.Err()
}

// Revision returns revision number n of the page.
func (s *Store) Revision(ctx context.Context, id int64, n int) (Revision, error) {
	r, err := scanRevision(s.DB.QueryRowContext(ctx, selectRevisions+" AND r.number = ?", id, n))
	if errors.Is(err, sql.ErrNoRows) {
		return Revision{}, ErrNotFound
	}
	return r, err
}

// Delete moves the page among the deleted pages, recording who deleted it.
func (s *Store) Delete(ctx context.Context, id, userID int64) error {
	return s.change1(ctx, id, "UPDATE page SET deleted_at = ?, deleted_by = ? WHERE id = ? AND deleted_at IS NULL",
		s.Now().Unix(), userID, id)
}

// Deleted returns the deleted pages that can still be restored, most recently deleted first.
func (s *Store) Deleted(ctx context.Context) ([]Page, error) {
	return s.queryPages(ctx, " WHERE p.deleted_at > ? ORDER BY p.deleted_at DESC, p.id DESC", s.expiredBefore())
}

// Restore brings a deleted page back, with its history, unless it has been deleted for too long.
func (s *Store) Restore(ctx context.Context, id int64) error {
	return s.change1(ctx, id, "UPDATE page SET deleted_at = NULL, deleted_by = NULL WHERE id = ? AND deleted_at > ?",
		id, s.expiredBefore())
}

// RemoveExpired removes for good, with their history, the pages deleted for longer than DeletedLifetime.
func (s *Store) RemoveExpired(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM page WHERE deleted_at <= ?", s.expiredBefore())
	return err
}

// expiredBefore is the deletion time, in Unix seconds, at or before which a deleted page is gone.
func (s *Store) expiredBefore() int64 {
	return s.Now().Add(-DeletedLifetime).Unix()
}

// change1 runs a statement meant to change page id and updates its index entry, or returns ErrNotFound.
func (s *Store) change1(ctx context.Context, id int64, query string, args ...any) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return ErrNotFound
		}
		return index(ctx, tx, id)
	})
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

// Result is a page found by a search, with an excerpt of its body.
type Result struct {
	Page
	Excerpt search.Excerpt
}

// Search returns the pages that aren't deleted and contain every searched word, in their title or body, in any form:
// first those with the words in their title, then by relevance.
func (s *Store) Search(ctx context.Context, text string) ([]Result, error) {
	query := search.Query(text)
	if query == "" {
		return nil, nil
	}
	pages, err := s.queryPages(ctx, ` JOIN page_fts ON page_fts.rowid = p.id
		WHERE page_fts MATCH ? AND p.deleted_at IS NULL
		ORDER BY p.id IN (SELECT rowid FROM page_fts WHERE page_fts MATCH ?) DESC, bm25(page_fts)`,
		query, "title_stem : ("+query+")")
	if err != nil {
		return nil, err
	}
	results := make([]Result, len(pages))
	for i, p := range pages {
		results[i] = Result{Page: p, Excerpt: search.MakeExcerpt(p.Body, p.Language, text)}
	}
	return results, nil
}
