package web

import (
	"net/http"
	"strings"

	"github.com/davidepedone/hausbuch/internal/page"
)

type searchData struct {
	Query   string
	Results []page.Result
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	results, err := s.pages.Search(r.Context(), q)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "search", searchData{Query: q, Results: results})
}
