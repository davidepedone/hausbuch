package web

import (
	"slices"
	"strconv"
	"sync"
	"time"
)

// An open editor sends a sign of activity every 30 seconds (see edit.html); one silent for editorTimeout counts as
// closed: a closed tab, a sleeping phone.
const editorTimeout = 2 * time.Minute

// editors tracks who has which page open in an editor, to warn others. Only a warning: the revision check on save is
// what keeps edits from overwriting each other, so losing this on a restart costs at most one ping of warnings.
type editors struct {
	mu   sync.Mutex
	open map[editorKey]*editor
}

type editorKey struct{ page, user int64 }

type editor struct {
	name  string    // display name
	since time.Time // when the editor was opened
	seen  time.Time // last sign of activity
}

// touch records that the user has the page open in an editor, since now unless it already was.
func (e *editors) touch(page, user int64, name string, now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := editorKey{page, user}
	if ed, ok := e.open[k]; ok && now.Sub(ed.seen) < editorTimeout {
		ed.seen = now
		return
	}
	e.open[k] = &editor{name: name, since: now, seen: now}
}

// close records that the user saved or left the page's editor.
func (e *editors) close(page, user int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.open, editorKey{page, user})
}

// others returns the editors of the page open by users other than user, longest open first, forgetting abandoned ones.
func (e *editors) others(page, user int64, now time.Time) []editor {
	e.mu.Lock()
	defer e.mu.Unlock()
	var eds []editor
	for k, ed := range e.open {
		switch {
		case now.Sub(ed.seen) >= editorTimeout:
			delete(e.open, k)
		case k.page == page && k.user != user:
			eds = append(eds, *ed)
		}
	}
	slices.SortFunc(eds, func(a, b editor) int { return a.since.Compare(b.since) })
	return eds
}

// duration says how long d is, for "has been editing it for …".
func duration(d time.Duration) string {
	plural := func(n int, unit string) string {
		if n == 1 {
			return "1 " + unit
		}
		return strconv.Itoa(n) + " " + unit + "s"
	}
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		return plural(int(d/time.Minute), "minute")
	}
	return plural(int(d/time.Hour), "hour")
}
