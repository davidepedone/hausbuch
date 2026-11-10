package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davidepedone/hausbuch/internal/markdown"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
)

// ingredientRow is an ingredient on the check page, with its tick and name as the form holds them.
type ingredientRow struct {
	stock.Ingredient
	Rows   []matchedRow // the pantry rows it matches
	Name   string
	Ticked bool
}

// matchedRow is a pantry row an ingredient matches, with the whole months since it last changed from three on, 0
// before: a row untouched for long may have been used up without the stock being updated.
type matchedRow struct {
	stock.Item
	Months int
}

// monthsSince is the whole calendar months from t to now.
func monthsSince(t, now time.Time) int {
	t, now = t.Local(), now.Local()
	m := (now.Year()-t.Year())*12 + int(now.Month()-t.Month())
	if now.Day() < t.Day() {
		m--
	}
	return m
}

type pantryData struct {
	Page  page.Page
	Rows  []ingredientRow
	Error string
}

// recipeFromPath returns the recipe named by the {id} of the path, or answers 404 for a page of another type.
func (s *server) recipeFromPath(w http.ResponseWriter, r *http.Request) (page.Page, bool) {
	p, ok := s.pageFromPath(w, r)
	if ok && p.Type != page.Recipe {
		http.Error(w, "Page not found.", http.StatusNotFound)
		return p, false
	}
	return p, ok
}

// checkRecipe returns the recipe's ingredients checked against the pantry and the list, each with its name and the
// missing ones ticked.
func (s *server) checkRecipe(r *http.Request, p page.Page) ([]ingredientRow, error) {
	rows, err := s.stock.List(r.Context(), stock.Pantry)
	if err != nil {
		return nil, err
	}
	entries, err := s.stock.Entries(r.Context())
	if err != nil {
		return nil, err
	}
	now := s.stock.Now()
	var checked []ingredientRow
	for _, i := range stock.Check(markdown.ListUnder(p.Body, "Ingredients"), rows, entries) {
		row := ingredientRow{Ingredient: i, Name: i.Text, Ticked: i.Missing()}
		for _, it := range i.Rows {
			m := matchedRow{Item: it}
			if months := monthsSince(it.ChangedAt, now); months >= 3 {
				m.Months = months
			}
			row.Rows = append(row.Rows, m)
		}
		checked = append(checked, row)
	}
	return checked, nil
}

func (s *server) pantryCheck(w http.ResponseWriter, r *http.Request) {
	p, ok := s.recipeFromPath(w, r)
	if !ok {
		return
	}
	rows, err := s.checkRecipe(r, p)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "pantry", pantryData{Page: p, Rows: rows})
}

// addIngredients adds the ticked ingredients to the list as Pantry entries, under the names in their fields.
func (s *server) addIngredients(w http.ResponseWriter, r *http.Request) {
	p, ok := s.recipeFromPath(w, r)
	if !ok {
		return
	}
	rows, err := s.checkRecipe(r, p)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad form.", http.StatusBadRequest)
		return
	}
	for i := range rows {
		rows[i].Name = r.PostForm.Get("name-" + strconv.Itoa(i))
		rows[i].Ticked = false
	}
	ticked := 0
	for _, v := range r.PostForm["add"] {
		if i, err := strconv.Atoi(v); err == nil && i >= 0 && i < len(rows) {
			rows[i].Ticked = true
			ticked++
		}
	}
	if ticked == 0 {
		s.render(w, r, http.StatusBadRequest, "pantry", pantryData{Page: p, Rows: rows, Error: "Tick what to add."})
		return
	}
	for _, row := range rows {
		if !row.Ticked || strings.TrimSpace(row.Name) == "" {
			continue
		}
		if err := s.stock.AddEntry(r.Context(), currentUser(r).ID, stock.Entry{Name: row.Name, Section: stock.Pantry}); err != nil {
			s.fail(w, err)
			return
		}
	}
	http.Redirect(w, r, "/shopping", http.StatusSeeOther)
}
