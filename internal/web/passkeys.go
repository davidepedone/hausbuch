package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/davidepedone/hausbuch/internal/user"
)

var errPublicURL = errors.New("HAUSBUCH_URL must be the address users open, such as https://home.example.org")

// PasskeyConfig returns the WebAuthn settings for the address users open, publicURL, or nil when passkeys can't be
// offered there: browsers allow them only over HTTPS, or on localhost.
func PasskeyConfig(publicURL string) (*webauthn.WebAuthn, error) {
	if publicURL == "" {
		return nil, nil
	}
	u, err := url.Parse(publicURL)
	if err != nil || u.Host == "" || u.Scheme != "https" && u.Scheme != "http" || u.Path != "" && u.Path != "/" ||
		u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errPublicURL
	}
	if u.Scheme == "http" && u.Hostname() != "localhost" {
		return nil, nil
	}
	return webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: "hausbuch",
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
	})
}

// ceremonyLifetime is how long a passkey prompt can stay open between its two requests.
const ceremonyLifetime = 5 * time.Minute

const ceremonyCookie = "hausbuch_ceremony"

// ceremony is what a passkey sign-in or registration keeps between its begin and finish requests. userID is 0 for a
// sign-in.
type ceremony struct {
	data    webauthn.SessionData
	userID  int64
	expires time.Time
}

// ceremonies are kept in memory: a restart in the middle of one only means trying again.
type ceremonies struct {
	mu sync.Mutex
	m  map[string]ceremony
}

func (c *ceremonies) start(w http.ResponseWriter, r *http.Request, data *webauthn.SessionData, userID int64) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	id := base64.RawURLEncoding.EncodeToString(b)
	now := time.Now()
	c.mu.Lock()
	for k, v := range c.m {
		if now.After(v.expires) {
			delete(c.m, k)
		}
	}
	c.m[id] = ceremony{data: *data, userID: userID, expires: now.Add(ceremonyLifetime)}
	c.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: ceremonyCookie, Value: id, Path: "/", MaxAge: int(ceremonyLifetime.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r)})
	return nil
}

// finish returns and forgets the request's ceremony, if it is one of userID's and still open.
func (c *ceremonies) finish(r *http.Request, userID int64) (webauthn.SessionData, bool) {
	cookie, err := r.Cookie(ceremonyCookie)
	if err != nil {
		return webauthn.SessionData{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cer, ok := c.m[cookie.Value]
	delete(c.m, cookie.Value)
	return cer.data, ok && cer.userID == userID && time.Now().Before(cer.expires)
}

// passkeyUser is an account as the WebAuthn library sees it.
type passkeyUser struct {
	user.User
	handle      []byte
	credentials []webauthn.Credential
}

func (u passkeyUser) WebAuthnID() []byte                         { return u.handle }
func (u passkeyUser) WebAuthnName() string                       { return u.Username }
func (u passkeyUser) WebAuthnDisplayName() string                { return u.DisplayName }
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// passkeyUser loads the account's handle and passkeys.
func (s *server) passkeyUser(ctx context.Context, u user.User) (passkeyUser, error) {
	handle, err := s.users.WebAuthnID(ctx, u.ID)
	if err != nil {
		return passkeyUser{}, err
	}
	keys, err := s.users.Passkeys(ctx, u.ID)
	if err != nil {
		return passkeyUser{}, err
	}
	pu := passkeyUser{User: u, handle: handle}
	for _, k := range keys {
		var c webauthn.Credential
		if err := json.Unmarshal(k.Credential, &c); err != nil {
			return passkeyUser{}, err
		}
		pu.credentials = append(pu.credentials, c)
	}
	return pu, nil
}

// routePasskeys registers the passkey endpoints: the sign-in ones on open, the others on signedIn.
func (s *server) routePasskeys(open, signedIn *http.ServeMux) {
	open.HandleFunc("POST /login/passkey/begin", s.beginPasskeySignIn)
	open.HandleFunc("POST /login/passkey/finish", s.finishPasskeySignIn)
	signedIn.HandleFunc("GET /account/passkeys/offer", s.passkeyOffer)
	signedIn.HandleFunc("POST /account/passkeys/begin", s.beginPasskeyRegistration)
	signedIn.HandleFunc("POST /account/passkeys/finish", s.finishPasskeyRegistration)
	signedIn.HandleFunc("POST /account/passkeys/remove", s.removePasskey)
}

// writeJSON answers a passkey request; the browser side reads "error" to tell the user what went wrong.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) passkeyError(w http.ResponseWriter, status int, msg string, err error) {
	if err != nil {
		slog.Info("passkey refused", "err", err)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *server) beginPasskeySignIn(w http.ResponseWriter, r *http.Request) {
	if s.passkeys == nil {
		http.NotFound(w, r)
		return
	}
	options, data, err := s.passkeys.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err == nil {
		err = s.ceremonies.start(w, r, data, 0)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, options)
}

const passkeyFailed = "The passkey sign-in didn't complete. Try again, or sign in with your password."

func (s *server) finishPasskeySignIn(w http.ResponseWriter, r *http.Request) {
	if s.passkeys == nil {
		http.NotFound(w, r)
		return
	}
	data, ok := s.ceremonies.finish(r, 0)
	if !ok {
		s.passkeyError(w, http.StatusBadRequest, passkeyFailed, nil)
		return
	}
	var signedIn user.User
	found, credential, err := s.passkeys.FinishPasskeyLogin(func(_, handle []byte) (webauthn.User, error) {
		u, err := s.users.UserByHandle(r.Context(), handle)
		if err != nil {
			return nil, err
		}
		signedIn = u
		return s.passkeyUser(r.Context(), u)
	}, data, r)
	if err != nil {
		s.passkeyError(w, http.StatusUnauthorized, passkeyFailed, err)
		return
	}
	// The password lock doesn't apply: a passkey can't be guessed.
	if signedIn.Disabled || found == nil {
		s.passkeyError(w, http.StatusUnauthorized, passkeyFailed, nil)
		return
	}
	record, err := json.Marshal(credential)
	if err == nil {
		err = s.users.UsePasskey(r.Context(), credential.ID, record)
	}
	var sess user.Session
	if err == nil {
		sess, err = s.users.CreateSession(r.Context(), signedIn)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	setSessionCookie(w, r, sess)
	writeJSON(w, http.StatusOK, map[string]string{"next": localPath(r.URL.Query().Get("next"))})
}

// passkeyOffer asks a user without passkeys, just signed in, to create one before going on to next.
func (s *server) passkeyOffer(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "offer", localPath(r.URL.Query().Get("next")))
}

func (s *server) beginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if s.passkeys == nil {
		http.NotFound(w, r)
		return
	}
	pu, err := s.passkeyUser(r.Context(), *currentUser(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	var exclude []protocol.CredentialDescriptor
	for _, c := range pu.credentials {
		exclude = append(exclude, c.Descriptor())
	}
	options, data, err := s.passkeys.BeginRegistration(pu,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired), webauthn.WithExclusions(exclude))
	if err == nil {
		err = s.ceremonies.start(w, r, data, pu.ID)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, options)
}

func (s *server) finishPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if s.passkeys == nil {
		http.NotFound(w, r)
		return
	}
	const failed = "The passkey wasn't created. Try again."
	u := currentUser(r)
	data, ok := s.ceremonies.finish(r, u.ID)
	if !ok {
		s.passkeyError(w, http.StatusBadRequest, failed, nil)
		return
	}
	pu, err := s.passkeyUser(r.Context(), *u)
	if err != nil {
		s.fail(w, err)
		return
	}
	credential, err := s.passkeys.FinishRegistration(pu, data, r)
	if err != nil {
		s.passkeyError(w, http.StatusBadRequest, failed, err)
		return
	}
	record, err := json.Marshal(credential)
	if err == nil {
		err = s.users.AddPasskey(r.Context(), u.ID, credential.ID, record)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"next": localPath(r.URL.Query().Get("next"))})
}

func (s *server) removePasskey(w http.ResponseWriter, r *http.Request) {
	id, err := base64.RawURLEncoding.DecodeString(r.PostFormValue("id"))
	if err != nil || len(id) == 0 {
		http.Error(w, "Passkey not found.", http.StatusNotFound)
		return
	}
	switch err := s.users.RemovePasskey(r.Context(), currentUser(r).ID, id); {
	case errors.Is(err, user.ErrNoPasskey):
		http.Error(w, "Passkey not found.", http.StatusNotFound)
	case err != nil:
		s.fail(w, err)
	default:
		http.Redirect(w, r, "/account/password", http.StatusSeeOther)
	}
}
