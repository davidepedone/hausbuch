package web

import (
	"net/http"
	"strings"

	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
)

type searchData struct {
	Query   string
	Results []page.Result
	Stock   []stockResults // one per section with matches, in the sections' order
}

type stockResults struct {
	Section stock.Section
	Items   []stock.Item
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	partial := r.Header.Get("HX-Request") != ""
	if partial && q == "" {
		return // the header's panel closes
	}
	results, err := s.pages.Search(r.Context(), q)
	if err != nil {
		s.fail(w, err)
		return
	}
	items, err := s.stock.Search(r.Context(), q)
	if err != nil {
		s.fail(w, err)
		return
	}
	d := searchData{Query: q, Results: results}
	for _, it := range items {
		if n := len(d.Stock); n == 0 || d.Stock[n-1].Section != it.Section {
			d.Stock = append(d.Stock, stockResults{Section: it.Section})
		}
		d.Stock[len(d.Stock)-1].Items = append(d.Stock[len(d.Stock)-1].Items, it)
	}
	if partial {
		s.renderPart(w, "search", "results", d)
		return
	}
	s.render(w, r, http.StatusOK, "search", d)
}
