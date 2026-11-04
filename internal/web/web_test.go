package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	if rec := get(t, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHomeUsesLayout(t *testing.T) {
	rec := get(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{`name="viewport"`, `href="/static/app.css"`, `src="/static/htmx.min.js"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("home page lacks %s", want)
		}
	}
}

func TestStaticAssetsAreEmbedded(t *testing.T) {
	rec := get(t, "/static/htmx.min.js")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "htmx") {
		t.Fatalf("status = %d, body starts %.40q", rec.Code, rec.Body.String())
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	if rec := get(t, "/nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
