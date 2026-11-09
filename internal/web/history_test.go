package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// clock makes the app's time settable; it starts at a fixed time.
func (a *testApp) clock() *time.Time {
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	a.pages.Now = func() time.Time { return now }
	return &now
}

func TestHistoryList(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	now := a.clock()
	path := anna.create("Pasta", "v1")
	for i, b := range []*browser{marco, anna, marco} {
		*now = now.Add(time.Hour)
		b.save(path, "Pasta", "v"+string(rune('2'+i)))
	}
	html := anna.view(path + "/history")
	var at []int
	for n := 4; n >= 1; n-- {
		at = append(at, strings.Index(html, ">Revision "+string(rune('0'+n))))
	}
	for i, pos := range at {
		if pos < 0 || (i > 0 && pos < at[i-1]) {
			t.Fatalf("revisions not all listed newest first (positions %v):\n%s", at, html)
		}
	}
	mustContain(t, html, "Revision 4 (current)",
		now.Local().Format("2 Jan 2006, 15:04")+" by Marco",
		now.Add(-3*time.Hour).Local().Format("2 Jan 2006, 15:04")+" by Anna")
	mustContain(t, anna.view(path), `href="`+path+`/history"`)
}

func TestOpenAnOldRevision(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Pasta", "Boil **water**.")
	b.saveForm(path, url.Values{"title": {"Spaghetti"}, "type": {"recipe"}, "body": {"v2"}})
	html := b.view(path + "/revisions/1")
	mustContain(t, html, "not the current version", ">Pasta</h1>", `<p class="mb-6 text-sm text-stone-500">Blank</p>`, "<strong>water</strong>",
		`action="`+path+`/revisions/1/restore"`)
	if resp := b.get(path + "/revisions/3"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing revision: %s", resp.Status)
	}
}

func TestCompare(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Pasta", "## Steps\nboil\nsalt\nserve\n")
	b.save(path, "Pasta", "## Steps\nboil\npepper\nserve\n")
	html := b.view(path + "/revisions/1/compare")
	mustContain(t, html, "+ pepper</span>", "- salt</span>", "  boil</span>")
}

func TestRestoreRevision(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	path := anna.create("Pasta", "v1")
	for _, body := range []string{"v2", "v3", "v4", "v5"} {
		anna.save(path, "Pasta", body)
	}
	resp := marco.post(path+"/revisions/2/restore", nil)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != path {
		t.Fatalf("restore: %s to %q", resp.Status, resp.Header.Get("Location"))
	}
	if got := renderedBody(t, marco.view(path)); got != "<p>v2</p>\n" {
		t.Errorf("body = %q", got)
	}
	revs, err := a.pages.Revisions(context.Background(), idOf(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 6 || revs[0].Author != "Marco" || revs[0].Body != "v2" {
		t.Errorf("%d revisions, newest %+v", len(revs), revs[0])
	}
}

func TestConflictingSave(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	path := anna.create("Pasta", "boil\nsalt\n")
	anna.view(path + "/edit")
	mustContain(t, marco.view(path+"/edit?anyway=1"), `name="revision" value="1"`)

	if resp := anna.post(path, url.Values{"title": {"Pasta"}, "body": {"boil\npepper\n"},
		"revision": {"1"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("first save: %s", resp.Status)
	}
	marcos := url.Values{"title": {"Pasta"}, "body": {"boil\nsalt\nserve\n"}, "revision": {"1"}}
	resp := marco.post(path, marcos)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("second save: %s", resp.Status)
	}
	mustContain(t, resp.body, "<strong>Anna</strong> changed this page", textarea("boil\nsalt\nserve\n"),
		"+ pepper</span>", "- salt</span>", `name="revision" value="2"`)
	if got := renderedBody(t, anna.view(path)); got != "<p>boil<br>\npepper</p>\n" {
		t.Errorf("conflicting save applied: %q", got)
	}

	// Once reviewed, saving again goes through.
	marcos.Set("revision", "2")
	if resp := marco.post(path, marcos); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("save after review: %s", resp.Status)
	}
}

func TestSomeoneElseIsEditing(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	now := a.clock()
	path := anna.create("Pasta", "")
	anna.view(path + "/edit")
	for range 10 {
		*now = now.Add(30 * time.Second)
		if resp := anna.post(path+"/editing", nil); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("activity signal: %s", resp.Status)
		}
	}
	html := marco.view(path + "/edit")
	mustContain(t, html, "Anna has been editing this page for 5 minutes.",
		`href="`+path+`" class="rounded bg-sage-700 px-4 py-2 font-medium text-white hover:bg-sage-800">Go back</a>`, `href="`+path+`/edit?anyway=1"`)
	if strings.Contains(html, "<textarea") {
		t.Error("editor opened without a choice")
	}
	mustContain(t, marco.view(path+"/edit?anyway=1"), "<textarea")
}

func TestEditorSavedOrCancelled(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	path := anna.create("Pasta", "")

	anna.view(path + "/edit")
	anna.save(path, "Pasta", "saved")
	if html := marco.view(path + "/edit"); strings.Contains(html, "has been editing") {
		t.Error("warning after Anna saved")
	}
	marco.post(path+"/cancel", nil)

	if html := anna.view(path + "/edit"); !strings.Contains(html, "<textarea") {
		t.Error("warning after Marco cancelled")
	}
	if resp := anna.post(path+"/cancel", url.Values{"body": {"unsaved"}}); resp.StatusCode != http.StatusSeeOther ||
		resp.Header.Get("Location") != path {
		t.Fatalf("cancel: %s to %q", resp.Status, resp.Header.Get("Location"))
	}
	if html := marco.view(path + "/edit"); strings.Contains(html, "has been editing") {
		t.Error("warning after Anna cancelled")
	}
	if got := renderedBody(t, anna.view(path)); got != "<p>saved</p>\n" {
		t.Errorf("cancel saved the text: %q", got)
	}
}

func TestEditorAbandoned(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	now := a.clock()
	path := anna.create("Pasta", "")
	anna.view(path + "/edit")

	*now = now.Add(2*time.Minute - time.Second)
	mustContain(t, marco.view(path+"/edit"), "Anna has been editing this page for 1 minute.")
	*now = now.Add(time.Second)
	if html := marco.view(path + "/edit"); strings.Contains(html, "has been editing") {
		t.Error("warning 2 minutes after Anna's last sign of activity")
	}
	marco.post(path+"/cancel", nil)

	// An abandoned editor coming back (a phone waking up) counts from then, not from when it was opened.
	anna.view(path + "/edit")
	*now = now.Add(time.Hour)
	anna.post(path+"/editing", nil)
	*now = now.Add(time.Minute)
	mustContain(t, marco.view(path+"/edit"), "Anna has been editing this page for 1 minute.")
}

func TestOwnEditorInAnotherTab(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	path := anna.create("Pasta", "")
	anna.view(path + "/edit")
	if html := anna.view(path + "/edit"); strings.Contains(html, "has been editing") {
		t.Error("warned about herself")
	}
}

func TestDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		10 * time.Second: "less than a minute", time.Minute: "1 minute", 5*time.Minute + 59*time.Second: "5 minutes",
		time.Hour: "1 hour", 3*time.Hour + 20*time.Minute: "3 hours",
	} {
		if got := duration(d); got != want {
			t.Errorf("duration(%v) = %q, want %q", d, got, want)
		}
	}
}
