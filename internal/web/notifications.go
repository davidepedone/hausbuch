package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/davidepedone/hausbuch/internal/push"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
)

// notificationsData is the account page's notifications section.
type notificationsData struct {
	Key       string   // the VAPID public key browsers subscribe with
	Endpoints []string // subscribed in this session, so the page can tell whether this device is
	Time      string   // HH:MM
	Sections  map[stock.Section]bool
	Saved     bool
	Error     string
}

// EndpointsJSON is Endpoints for the page's script, an array even when empty.
func (d notificationsData) EndpointsJSON() string {
	b, err := json.Marshal(append([]string{}, d.Endpoints...))
	if err != nil {
		panic(err) // strings always marshal
	}
	return string(b)
}

// sectionChoice is a section's checkbox.
type sectionChoice struct {
	Section stock.Section
	Checked bool
}

// Choices are the sections' checkboxes, in their usual order.
func (d notificationsData) Choices() []sectionChoice {
	var c []sectionChoice
	for _, section := range stock.Sections {
		c = append(c, sectionChoice{section, d.Sections[section]})
	}
	return c
}

func (s *server) routeNotifications(signedIn *http.ServeMux) {
	signedIn.HandleFunc("POST /account/notifications", s.saveNotificationChoice)
	signedIn.HandleFunc("POST /account/notifications/subscribe", s.subscribe)
	signedIn.HandleFunc("POST /account/notifications/unsubscribe", s.unsubscribe)
}

// notifications fills the account page's notifications section for the request's user and session.
func (s *server) notifications(r *http.Request, d *notificationsData) error {
	keys, err := s.push.Keys(r.Context())
	if err != nil {
		return err
	}
	d.Key = keys.Public
	if d.Endpoints, err = s.push.Endpoints(r.Context(), sessionHash(r)); err != nil {
		return err
	}
	if d.Time == "" {
		c, err := s.push.Choice(r.Context(), currentUser(r).ID)
		if err != nil {
			return err
		}
		d.Time = fmt.Sprintf("%02d:%02d", c.Minutes/60, c.Minutes%60)
		d.Sections = c.Sections
	}
	return nil
}

func (s *server) saveNotificationChoice(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	c := push.Choice{Sections: map[stock.Section]bool{}}
	for _, section := range stock.Sections {
		c.Sections[section] = slices.Contains(r.PostForm["section"], string(section))
	}
	d := notificationsData{Time: r.PostFormValue("time"), Sections: c.Sections}
	t, err := time.Parse("15:04", d.Time)
	if err != nil {
		d.Error = "Give the time as hours and minutes, such as 09:00."
		s.renderAccount(w, r, http.StatusBadRequest, passwordData{Notifications: &d})
		return
	}
	c.Minutes = t.Hour()*60 + t.Minute()
	if err := s.push.SetChoice(r.Context(), currentUser(r).ID, c); err != nil {
		s.fail(w, err)
		return
	}
	d.Saved = true
	s.renderAccount(w, r, http.StatusOK, passwordData{Notifications: &d})
}

func (s *server) subscribe(w http.ResponseWriter, r *http.Request) {
	var sub push.Subscription
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&sub); err != nil {
		http.Error(w, "Not a push subscription.", http.StatusBadRequest)
		return
	}
	err := s.push.Subscribe(r.Context(), sessionHash(r), currentUser(r).ID, sub)
	switch {
	case errors.Is(err, push.ErrSubscription):
		http.Error(w, "Not a push subscription.", http.StatusBadRequest)
	case err != nil:
		s.fail(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) unsubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&body); err != nil {
		http.Error(w, "No endpoint given.", http.StatusBadRequest)
		return
	}
	if err := s.push.Unsubscribe(r.Context(), sessionHash(r), body.Endpoint); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sessionHash is the stored form of the request's session; requireSession has already checked the cookie.
func sessionHash(r *http.Request) []byte {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	return user.SessionHash(c.Value)
}
