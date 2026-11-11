package web

import "testing"

func TestPasskeyConfig(t *testing.T) {
	for _, c := range []struct {
		url    string
		rpID   string // "" for passkeys off
		origin string
		bad    bool
	}{
		{url: ""},
		{url: "http://hausbuch.local"},
		{url: "http://192.168.1.10:8080/"},
		{url: "https://home.example.org", rpID: "home.example.org", origin: "https://home.example.org"},
		{url: "https://home.example.org:8443/", rpID: "home.example.org", origin: "https://home.example.org:8443"},
		{url: "http://localhost:8080", rpID: "localhost", origin: "http://localhost:8080"},
		{url: "home.example.org", bad: true},
		{url: "https://home.example.org/hausbuch", bad: true},
		{url: "ftp://home.example.org", bad: true},
	} {
		w, err := PasskeyConfig(c.url)
		switch {
		case c.bad:
			if err == nil {
				t.Errorf("%q accepted", c.url)
			}
		case err != nil:
			t.Errorf("%q: %v", c.url, err)
		case c.rpID == "" && w != nil:
			t.Errorf("%q: passkeys on", c.url)
		case c.rpID != "" && (w == nil || w.Config.RPID != c.rpID || len(w.Config.RPOrigins) != 1 || w.Config.RPOrigins[0] != c.origin):
			t.Errorf("%q: %+v", c.url, w)
		}
	}
}
