package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/davidepedone/hausbuch/internal/user"
)

type linkData struct {
	Token string
	User  user.User // zero when the link is not valid
}

// linkPage shows whose account a sign-in link opens, without using it up: link previews fetch it too.
func (s *server) linkPage(w http.ResponseWriter, r *http.Request) {
	u, err := s.users.LinkUser(r.Context(), r.PathValue("token"))
	if err != nil && !errors.Is(err, user.ErrNoLink) {
		s.fail(w, err)
		return
	}
	status := http.StatusOK
	if err != nil {
		s.logEvent(r, slog.LevelWarn, "sign-in link not valid", "unknown")
		status = http.StatusNotFound
	}
	s.render(w, r, status, "link", linkData{Token: r.PathValue("token"), User: u})
}

func (s *server) useLink(w http.ResponseWriter, r *http.Request) {
	u, err := s.users.UseLink(r.Context(), r.PathValue("token"))
	if errors.Is(err, user.ErrNoLink) {
		s.logEvent(r, slog.LevelWarn, "sign-in link not valid", "unknown")
		s.render(w, r, http.StatusNotFound, "link", linkData{})
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.logEvent(r, slog.LevelInfo, "signed in", u.Username, "method", "link")
	s.signIn(w, r, u, "/")
}
