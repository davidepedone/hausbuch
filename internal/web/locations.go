package web

import (
	"net/http"
	"net/url"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// locationURL is the page of the rows kept in a location.
func locationURL(location string) string { return "/locations/" + url.PathEscape(location) }

// locationGroup is the rows of one section kept in a location.
type locationGroup struct {
	Section stock.Section
	Items   []stock.Item
}

type locationData struct {
	Location string
	Groups   []locationGroup
}

func (s *server) locations(w http.ResponseWriter, r *http.Request) {
	places, err := s.stock.Places(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "locations", places)
}

func (s *server) location(w http.ResponseWriter, r *http.Request) {
	d := locationData{Location: r.PathValue("name")}
	items, err := s.stock.InLocation(r.Context(), d.Location)
	if err != nil {
		s.fail(w, err)
		return
	}
	for _, it := range items {
		if n := len(d.Groups); n == 0 || d.Groups[n-1].Section != it.Section {
			d.Groups = append(d.Groups, locationGroup{Section: it.Section})
		}
		d.Groups[len(d.Groups)-1].Items = append(d.Groups[len(d.Groups)-1].Items, it)
	}
	s.render(w, r, http.StatusOK, "location", d)
}
