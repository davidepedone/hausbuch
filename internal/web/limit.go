package web

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// requestLimit is how many requests an address may make in a minute to what can be reached without signing in: room
// for a household behind one address and some mistyping, too little for guessing passwords from one address.
const requestLimit = 20

const tooManyRequests = "Too many attempts from your address. Try again in a minute."

// limiter counts requests per address in the current minute. The counts are emptied when the minute changes, so they
// never hold more than one minute's addresses.
type limiter struct {
	mu     sync.Mutex
	minute int64
	counts map[string]int
}

// count counts a request from address at now and returns how many the address has made this minute.
func (l *limiter) count(address string, now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if m := now.Unix() / 60; m != l.minute || l.counts == nil {
		l.minute, l.counts = m, map[string]int{}
	}
	l.counts[address]++
	return l.counts[address]
}

// limited refuses the requests to h of an address over the request limit: in JSON for the passkey endpoints, whose
// script shows the message, and with the login page for the others.
func (s *server) limited(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := s.limits.count(s.address(r), s.users.Now())
		if n <= requestLimit {
			h(w, r)
			return
		}
		if n == requestLimit+1 { // once a minute, however many follow
			s.logEvent(r, slog.LevelWarn, "request limit hit", "unknown", "route", r.Pattern) // a link's path holds its token
		}
		if strings.HasPrefix(r.URL.Path, "/login/passkey/") {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": tooManyRequests})
			return
		}
		s.renderLogin(w, r, http.StatusTooManyRequests, loginData{Error: tooManyRequests})
	}
}
