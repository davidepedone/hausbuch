package page

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func (f *fixture) revisions(id int64) []Revision {
	f.t.Helper()
	revs, err := f.pages.Revisions(ctx, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return revs
}

func TestEditCreatesARevision(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "Boil water.")
	f.now = f.now.Add(time.Hour)
	p := f.get(id)
	p.Body = "Boil salted water."
	if err := f.pages.Update(ctx, id, f.marco.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	revs := f.revisions(id)
	if len(revs) != 2 {
		t.Fatalf("%d revisions", len(revs))
	}
	if r := revs[0]; r.Number != 2 || r.Body != "Boil salted water." || r.Author != "Marco" || !r.CreatedAt.Equal(f.now) {
		t.Errorf("newest = %+v", r)
	}
	if r := revs[1]; r.Number != 1 || r.Body != "Boil water." || r.Author != "Anna" {
		t.Errorf("first = %+v", r)
	}
}

func TestSaveWithoutChanges(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "Boil water.")
	p := f.get(id)
	if err := f.pages.Update(ctx, id, f.marco.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	if revs := f.revisions(id); len(revs) != 1 {
		t.Errorf("%d revisions after a save without changes", len(revs))
	}
	// Each of title, type, language and body is a change.
	for _, change := range []func(*Content){
		func(c *Content) { c.Title = "pasta" },
		func(c *Content) { c.Type = Recipe },
		func(c *Content) { c.Language = "it" },
		func(c *Content) { c.Body += "\n" },
	} {
		p := f.get(id)
		change(&p.Content)
		if err := f.pages.Update(ctx, id, f.anna.ID, p.Revision, p.Content); err != nil {
			t.Fatal(err)
		}
	}
	if revs := f.revisions(id); len(revs) != 5 {
		t.Errorf("%d revisions after four changes, want 5", len(revs))
	}
}

func TestRevision(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "v1")
	p := f.get(id)
	p.Body = "v2"
	if err := f.pages.Update(ctx, id, f.anna.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	r, err := f.pages.Revision(ctx, id, 1)
	if err != nil || r.Body != "v1" || r.Title != "Pasta" {
		t.Errorf("revision 1 = %+v, %v", r, err)
	}
	if _, err := f.pages.Revision(ctx, id, 3); !errors.Is(err, ErrNotFound) {
		t.Errorf("revision 3: %v", err)
	}
}

func TestConflictingSave(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "v1")
	annaStart, marcoStart := f.get(id), f.get(id)

	annaStart.Body = "Anna's"
	if err := f.pages.Update(ctx, id, f.anna.ID, annaStart.Revision, annaStart.Content); err != nil {
		t.Fatal(err)
	}
	marcoStart.Body = "Marco's"
	err := f.pages.Update(ctx, id, f.marco.ID, marcoStart.Revision, marcoStart.Content)
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Revision != 2 || !slices.Equal(conflict.By, []string{"Anna"}) {
		t.Fatalf("err = %v (%+v), want a conflict with Anna's revision 2", err, conflict)
	}
	if p := f.get(id); p.Body != "Anna's" || p.Revision != 2 {
		t.Errorf("conflicting save was applied: %+v", p)
	}
	// Saving again from the current revision goes through.
	if err := f.pages.Update(ctx, id, f.marco.ID, conflict.Revision, marcoStart.Content); err != nil {
		t.Errorf("save after review: %v", err)
	}
}

func TestConflictNamesEveryoneWhoSaved(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "v1")
	paolo := f.addUser("paolo", "Paolo")
	for i, u := range []int64{f.anna.ID, paolo.ID, f.anna.ID} {
		p := f.get(id)
		p.Body = string(rune('a' + i))
		if err := f.pages.Update(ctx, id, u, p.Revision, p.Content); err != nil {
			t.Fatal(err)
		}
	}
	err := f.pages.Update(ctx, id, f.marco.ID, 1, Content{Title: "Pasta", Type: Blank, Language: "en", Body: "x"})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || !slices.Equal(conflict.By, []string{"Anna", "Paolo"}) {
		t.Errorf("err = %v (%+v)", err, conflict)
	}
}

func TestParallelSavesFromTheSameRevision(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "v1")
	start := f.get(id)

	const n = 8
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			c := start.Content
			c.Body = string(rune('a' + i))
			errs[i] = f.pages.Update(ctx, id, f.marco.ID, start.Revision, c)
		})
	}
	wg.Wait()

	applied := 0
	for _, err := range errs {
		var conflict *ConflictError
		switch {
		case err == nil:
			applied++
		case !errors.As(err, &conflict):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if applied != 1 {
		t.Errorf("%d of %d parallel saves applied, want 1", applied, n)
	}
	if revs := f.revisions(id); len(revs) != 2 {
		t.Errorf("%d revisions, want 2", len(revs))
	}
}

func TestDiff(t *testing.T) {
	got := Diff("## Steps\nboil\nsalt\nserve\n", "## Steps\nboil\npepper\nserve\nenjoy\n")
	want := []DiffLine{
		{Text: "## Steps"}, {Text: "boil"},
		{Text: "salt", Removed: true}, {Text: "pepper", Added: true},
		{Text: "serve"}, {Text: "enjoy", Added: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Diff =\n%+v\nwant\n%+v", got, want)
	}
}
