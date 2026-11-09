package page

import (
	"slices"
	"testing"

	"github.com/davidepedone/hausbuch/internal/search"
)

// indexed returns the ids of the pages whose index entry holds the English stem of word.
func (f *fixture) indexed(word string) []int64 {
	f.t.Helper()
	rows, err := f.pages.DB.QueryContext(ctx, "SELECT rowid FROM page_fts WHERE page_fts MATCH ? ORDER BY rowid",
		`"`+search.Stem(word, "en")+`"`)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			f.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		f.t.Fatal(err)
	}
	return ids
}

func (f *fixture) wantIndexed(word string, want ...int64) {
	f.t.Helper()
	if got := f.indexed(word); !slices.Equal(got, want) {
		f.t.Errorf("pages indexed under %q: %v, want %v", word, got, want)
	}
}

func TestIndexFollowsSaveDeleteAndRestore(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Coffee machine", "Descaling every three months.")
	f.wantIndexed("descale", id)
	f.wantIndexed("coffee", id)

	p := f.get(id)
	p.Body = "Clean the filter."
	if err := f.pages.Update(ctx, id, f.anna.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	f.wantIndexed("descale")
	f.wantIndexed("filter", id)

	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	f.wantIndexed("filter")
	f.wantIndexed("coffee")

	if err := f.pages.Restore(ctx, id); err != nil {
		t.Fatal(err)
	}
	f.wantIndexed("filter", id)
}

func TestIndexHoldsBothLanguages(t *testing.T) {
	f := newFixture(t)
	id, err := f.pages.Create(ctx, f.anna.ID,
		Content{Title: "Giardino", Type: Blank, Body: "L'irrigazione parte ad aprile."})
	if err != nil {
		t.Fatal(err)
	}
	var stems string
	if err := f.pages.DB.QueryRowContext(ctx, "SELECT body_stem FROM page_fts WHERE rowid = ?", id).Scan(&stems); err != nil {
		t.Fatal(err)
	}
	if want := search.Stems("L'irrigazione parte ad aprile."); stems != want {
		t.Errorf("body stems %q, want %q", stems, want)
	}
}

func TestReindex(t *testing.T) {
	f := newFixture(t)
	live := f.create(f.anna, "Coffee machine", "Descaling.")
	deleted := f.create(f.anna, "Old coffee machine", "Descaling.")
	if err := f.pages.Delete(ctx, deleted, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	// As in a database from before the index, or indexed by another version's stemmers.
	if _, err := f.pages.DB.ExecContext(ctx,
		"DELETE FROM page_fts; INSERT INTO page_fts (rowid, title_stem, body_stem) VALUES (?, 'stale', 'stale')",
		deleted); err != nil {
		t.Fatal(err)
	}

	if err := f.pages.Reindex(ctx); err != nil {
		t.Fatal(err)
	}
	f.wantIndexed("descale", live)
	f.wantIndexed("stale")
}
