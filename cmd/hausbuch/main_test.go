package main

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/davidepedone/hausbuch/internal/stock"
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

func TestLeadTimes(t *testing.T) {
	lead, err := leadTimes()
	if err != nil || lead[stock.Medicines] != 30 || lead[stock.Pantry] != 7 {
		t.Errorf("defaults: %v %v", lead, err)
	}
	t.Setenv("HAUSBUCH_PANTRY_LEAD_DAYS", "3")
	t.Setenv("HAUSBUCH_MEDICINES_LEAD_DAYS", "0")
	if lead, err := leadTimes(); err != nil || lead[stock.Medicines] != 0 || lead[stock.Pantry] != 3 {
		t.Errorf("configured: %v %v", lead, err)
	}
}

func TestServeRefusesInvalidLeadTime(t *testing.T) {
	t.Setenv("HAUSBUCH_DB_PATH", filepath.Join(t.TempDir(), "hausbuch.db"))
	t.Setenv("HAUSBUCH_LISTEN", "127.0.0.1:0")
	for _, key := range []string{"HAUSBUCH_MEDICINES_LEAD_DAYS", "HAUSBUCH_PANTRY_LEAD_DAYS"} {
		for _, v := range []string{"-1", "seven", "1.5"} {
			t.Run(key+"="+v, func(t *testing.T) {
				t.Setenv(key, v)
				if code := run([]string{"serve"}, nil, io.Discard, io.Discard); code != 1 {
					t.Errorf("exit code %d, want 1", code)
				}
			})
		}
	}
}
