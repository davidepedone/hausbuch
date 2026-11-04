package main

import (
	"io"
	"path/filepath"
	"testing"
)

func TestHealthURL(t *testing.T) {
	for addr, want := range map[string]string{
		":8080":          "http://127.0.0.1:8080/healthz",
		"0.0.0.0:8080":   "http://127.0.0.1:8080/healthz",
		"[::]:8080":      "http://127.0.0.1:8080/healthz",
		"127.0.0.1:9000": "http://127.0.0.1:9000/healthz",
		"[::1]:9000":     "http://[::1]:9000/healthz",
	} {
		got, err := healthURL(addr)
		if err != nil || got != want {
			t.Errorf("healthURL(%q) = %q, %v; want %q", addr, got, err, want)
		}
	}
	if _, err := healthURL("8080"); err == nil {
		t.Error("healthURL accepted an address without a port separator")
	}
}

func TestServeRefusesUnknownDefaultLanguage(t *testing.T) {
	t.Setenv("HAUSBUCH_DEFAULT_LANGUAGE", "de")
	t.Setenv("HAUSBUCH_DB_PATH", filepath.Join(t.TempDir(), "hausbuch.db"))
	t.Setenv("HAUSBUCH_LISTEN", "127.0.0.1:0")
	if code := run([]string{"serve"}, nil, io.Discard, io.Discard); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
