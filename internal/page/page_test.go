package page

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/user"
)

var ctx = context.Background()

const day = 24 * time.Hour

type fixture struct {
	t     *testing.T
	pages *Store
	users *user.Store
	now   time.Time
	anna  user.User
	marco user.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "hausbuch.db"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	f := &fixture{t: t, users: user.NewStore(database), now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	f.pages = &Store{DB: database, Now: func() time.Time { return f.now }}
	f.anna = f.addUser("anna", "Anna")
	f.marco = f.addUser("marco", "Marco")
	return f
}

func (f *fixture) addUser(username, name string) user.User {
	f.t.Helper()
	u, err := f.users.Add(ctx, username, name, "pw")
	if err != nil {
		f.t.Fatal(err)
	}
	return u
}

func (f *fixture) create(u user.User, title, body string) int64 {
	f.t.Helper()
	id, err := f.pages.Create(ctx, u.ID, Content{Title: title, Type: Blank, Body: body})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) get(id int64) Page {
	f.t.Helper()
	p, err := f.pages.Get(ctx, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) rename(u user.User, id int64, title string) error {
	p := f.get(id)
	p.Title = title
	return f.pages.Update(ctx, id, u.ID, p.Revision, p.Content)
}

func (f *fixture) resolve(title string) int64 {
	f.t.Helper()
	id, found, err := f.pages.Resolve(ctx, title)
	if err != nil {
		f.t.Fatal(err)
	}
	if !found {
		return 0
	}
	return id
}

func titles(pages []Page) []string {
	var ts []string
	for _, p := range pages {
		ts = append(ts, p.Title)
	}
	return ts
}

func TestCreateRecordsCreator(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "  Coffee machine ", "Descale monthly.")
	p := f.get(id)
	if p.Title != "Coffee machine" || p.Body != "Descale monthly." || p.CreatedBy != "Anna" || p.ChangedBy != "Anna" ||
		!p.ChangedAt.Equal(f.now) || p.Revision != 1 {
		t.Errorf("page = %+v", p)
	}
}

func TestTitleAlreadyUsed(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Coffee machine", "")
	_, err := f.pages.Create(ctx, f.marco.ID, Content{Title: " coffee MACHINE ", Type: Blank})
	var taken *TitleTakenError
	if !errors.As(err, &taken) || taken.PageID != id || taken.Title != "Coffee machine" || taken.Deleted {
		t.Errorf("err = %v (%+v), want title taken by page %d", err, taken, id)
	}
}

func TestEmptyTitle(t *testing.T) {
	f := newFixture(t)
	for _, title := range []string{"", "  \t"} {
		if _, err := f.pages.Create(ctx, f.anna.ID, Content{Title: title, Type: Blank}); !errors.Is(err, ErrEmptyTitle) {
			t.Errorf("Create(%q) = %v, want ErrEmptyTitle", title, err)
		}
	}
	id := f.create(f.anna, "Pasta", "")
	if err := f.rename(f.anna, id, " "); !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("rename to blank = %v, want ErrEmptyTitle", err)
	}
}

func TestEditBody(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Pasta", "Boil water.")
	f.now = f.now.Add(time.Hour)
	p := f.get(id)
	p.Body = "Boil salted water."
	if err := f.pages.Update(ctx, id, f.marco.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	p = f.get(id)
	if p.Body != "Boil salted water." || p.ChangedBy != "Marco" || p.CreatedBy != "Anna" || !p.ChangedAt.Equal(f.now) {
		t.Errorf("page = %+v", p)
	}
}

func TestRenameKeepsFormerTitleAsLink(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Coffee maker", "")
	if err := f.rename(f.anna, id, "Coffee machine"); err != nil {
		t.Fatal(err)
	}
	if got := f.resolve("coffee maker"); got != id {
		t.Errorf("former title resolves to %d, want %d", got, id)
	}
	if got := f.resolve("Coffee machine"); got != id {
		t.Errorf("new title resolves to %d, want %d", got, id)
	}
	// The former title stays taken, by the renamed page.
	_, err := f.pages.Create(ctx, f.marco.ID, Content{Title: "Coffee maker", Type: Blank})
	var taken *TitleTakenError
	if !errors.As(err, &taken) || taken.PageID != id || taken.Title != "Coffee machine" {
		t.Errorf("creating with the former title = %v, want taken by the renamed page", err)
	}
	// Renaming back to it is allowed, and so is a change of case only.
	if err := f.rename(f.anna, id, "Coffee maker"); err != nil {
		t.Errorf("renaming back = %v", err)
	}
	if err := f.rename(f.anna, id, "coffee Maker"); err != nil {
		t.Errorf("changing case = %v", err)
	}
	if got := f.resolve("Coffee machine"); got != id {
		t.Errorf("after renaming back, the other title resolves to %d, want %d", got, id)
	}
	if err := f.rename(f.anna, id, "Espresso machine"); err != nil {
		t.Errorf("renaming again = %v", err)
	}
	for _, title := range []string{"Coffee maker", "Coffee machine", "Espresso machine"} {
		if got := f.resolve(title); got != id {
			t.Errorf("%q resolves to %d, want %d", title, got, id)
		}
	}
}

func TestRenameToTakenTitle(t *testing.T) {
	f := newFixture(t)
	pasta := f.create(f.anna, "Pasta", "")
	sauce := f.create(f.anna, "Tomato sauce", "")
	var taken *TitleTakenError
	if err := f.rename(f.anna, sauce, "PASTA"); !errors.As(err, &taken) || taken.PageID != pasta {
		t.Errorf("rename = %v, want taken by page %d", err, pasta)
	}
	if p := f.get(sauce); p.Title != "Tomato sauce" {
		t.Errorf("refused rename changed the title to %q", p.Title)
	}
}

func TestBacklinks(t *testing.T) {
	f := newFixture(t)
	sauce := f.create(f.anna, "Tomato sauce", "Itself: [[Tomato sauce]]")
	f.create(f.anna, "Weekly menu", "Monday: [[tomato sauce|pasta al pomodoro]]")
	f.create(f.anna, "Pasta", "With [[Tomato sauce]].")
	f.create(f.anna, "Pizza", "No link.")
	got, err := f.pages.Backlinks(ctx, sauce)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Pasta", "Weekly menu"}; !slices.Equal(titles(got), want) {
		t.Errorf("backlinks = %q, want %q", titles(got), want)
	}
}

func TestBacklinksLeaveOutDeletedPages(t *testing.T) {
	f := newFixture(t)
	sauce := f.create(f.anna, "Tomato sauce", "")
	pasta := f.create(f.anna, "Pasta", "With [[Tomato sauce]].")
	if err := f.pages.Delete(ctx, pasta, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.pages.Backlinks(ctx, sauce); len(got) != 0 {
		t.Errorf("backlinks = %q", titles(got))
	}
}

func TestBacklinksFollowRenamesAndEdits(t *testing.T) {
	f := newFixture(t)
	machine := f.create(f.anna, "Coffee maker", "")
	descale := f.create(f.anna, "Descaling", "See [[Coffee maker]].")
	if err := f.rename(f.anna, machine, "Coffee machine"); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.pages.Backlinks(ctx, machine); !slices.Equal(titles(got), []string{"Descaling"}) {
		t.Errorf("after rename, backlinks = %q", titles(got))
	}
	p := f.get(descale)
	p.Body = "No more link."
	if err := f.pages.Update(ctx, descale, f.anna.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.pages.Backlinks(ctx, machine); len(got) != 0 {
		t.Errorf("after removing the link, backlinks = %q", titles(got))
	}
}

func TestListNewestChangeFirstWithTypeFilter(t *testing.T) {
	f := newFixture(t)
	pasta := f.create(f.anna, "Pasta", "")
	f.now = f.now.Add(time.Minute)
	if _, err := f.pages.Create(ctx, f.anna.ID, Content{Title: "Pizza", Type: Recipe}); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	f.create(f.anna, "Irrigation", "")
	// Pasta changed last, in the same second as nothing else.
	f.now = f.now.Add(time.Minute)
	p := f.get(pasta)
	p.Type = Recipe
	if err := f.pages.Update(ctx, pasta, f.marco.ID, p.Revision, p.Content); err != nil {
		t.Fatal(err)
	}

	all, err := f.pages.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Pasta", "Irrigation", "Pizza"}; !slices.Equal(titles(all), want) {
		t.Errorf("list = %q, want %q", titles(all), want)
	}
	if all[0].ChangedBy != "Marco" || !all[0].ChangedAt.Equal(f.now) || all[0].Type != Recipe {
		t.Errorf("first = %+v", all[0])
	}
}

func TestListOrdersChangesInTheSameSecond(t *testing.T) {
	f := newFixture(t)
	f.create(f.anna, "First", "")
	f.create(f.anna, "Second", "")
	all, _ := f.pages.List(ctx)
	if want := []string{"Second", "First"}; !slices.Equal(titles(all), want) {
		t.Errorf("list = %q, want %q", titles(all), want)
	}
}

func TestDisabledAuthorKeepsName(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.marco, "Pasta", "")
	if err := f.users.Disable(ctx, "marco"); err != nil {
		t.Fatal(err)
	}
	if p := f.get(id); p.CreatedBy != "Marco" || p.ChangedBy != "Marco" {
		t.Errorf("page = %+v", p)
	}
}

func TestDeleteAndRestore(t *testing.T) {
	f := newFixture(t)
	sauce := f.create(f.anna, "Tomato sauce", "Simmer.")
	f.create(f.anna, "Pasta", "With [[Tomato sauce]].")
	f.now = f.now.Add(time.Hour)
	if err := f.pages.Delete(ctx, sauce, f.marco.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := f.pages.Get(ctx, sauce); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get deleted = %v, want ErrNotFound", err)
	}
	if all, _ := f.pages.List(ctx); !slices.Equal(titles(all), []string{"Pasta"}) {
		t.Errorf("list = %q", titles(all))
	}
	if got := f.resolve("Tomato sauce"); got != 0 {
		t.Errorf("deleted page still resolves, to %d", got)
	}
	deleted, err := f.pages.Deleted(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0].DeletedBy != "Marco" || !deleted[0].DeletedAt.Equal(f.now) ||
		!deleted[0].RemovedAt().Equal(f.now.Add(30*day)) {
		t.Fatalf("deleted = %+v", deleted)
	}

	_, err = f.pages.Create(ctx, f.anna.ID, Content{Title: "tomato sauce", Type: Blank})
	var taken *TitleTakenError
	if !errors.As(err, &taken) || taken.PageID != sauce || !taken.Deleted {
		t.Errorf("creating with a deleted page's title = %v, want taken by deleted page %d", err, sauce)
	}

	if err := f.pages.Restore(ctx, sauce); err != nil {
		t.Fatal(err)
	}
	if p := f.get(sauce); p.Body != "Simmer." || p.Revision != 1 || !p.DeletedAt.IsZero() {
		t.Errorf("restored = %+v", p)
	}
	if got := f.resolve("Tomato sauce"); got != sauce {
		t.Errorf("restored page resolves to %d", got)
	}
	if deleted, _ := f.pages.Deleted(ctx); len(deleted) != 0 {
		t.Errorf("deleted after restore = %q", titles(deleted))
	}
}

func TestRemovalAfter30Days(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Old notes", "")
	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30*day + time.Second)

	// Gone before any cleanup has run.
	if deleted, _ := f.pages.Deleted(ctx); len(deleted) != 0 {
		t.Errorf("deleted = %q", titles(deleted))
	}
	if err := f.pages.Restore(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Restore = %v, want ErrNotFound", err)
	}
	if _, err := f.pages.Create(ctx, f.anna.ID, Content{Title: "Old notes", Type: Blank}); err != nil {
		t.Errorf("creating with the removed page's title = %v", err)
	}
	var revisions int
	if err := f.pages.DB.QueryRow("SELECT COUNT(*) FROM page_revision WHERE page_id = ?", id).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if revisions != 0 {
		t.Errorf("removed page left %d revisions", revisions)
	}
}

func TestRemoveExpired(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Old notes", "")
	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30*day - time.Second)
	if err := f.pages.RemoveExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if deleted, _ := f.pages.Deleted(ctx); len(deleted) != 1 {
		t.Fatalf("removed before 30 days: deleted = %q", titles(deleted))
	}
	f.now = f.now.Add(time.Second)
	if err := f.pages.RemoveExpired(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.pages.DB.QueryRow("SELECT COUNT(*) FROM page WHERE id = ?", id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("page still stored after 30 days")
	}
}

func TestRestoredBeforeRemoval(t *testing.T) {
	f := newFixture(t)
	id := f.create(f.anna, "Notes", "")
	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(29 * day)
	if err := f.pages.Restore(ctx, id); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(2 * day)
	if err := f.pages.RemoveExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if p := f.get(id); p.Title != "Notes" {
		t.Errorf("page = %+v", p)
	}
}
