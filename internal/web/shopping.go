package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// routeShopping registers the shopping list's pages.
func (s *server) routeShopping(mux *http.ServeMux) {
	mux.HandleFunc("GET /shopping", func(w http.ResponseWriter, r *http.Request) { s.shopping(w, r, entryForm{}) })
	mux.HandleFunc("GET /shopping/entries", s.entries)
	mux.HandleFunc("POST /shopping/entries", s.addEntry)
	mux.HandleFunc("POST /shopping/entries/{id}/{action}", s.changeEntry)
	mux.HandleFunc("POST /shopping/clear", s.clearBought)
	mux.HandleFunc("GET /shopping/product", s.listProduct)
	mux.HandleFunc("GET /shopping/entries/{id}/put-away", s.putAwayForm)
	mux.HandleFunc("POST /shopping/entries/{id}/put-away", s.putAway)
}

// entryForm is the list's add form as the user filled it in.
type entryForm struct {
	Name, Quantity, Unit, Section, Shop string
	Error                               string
}

type shoppingData struct {
	Form   entryForm
	Names  []string
	Shops  []string
	ToBuy  []stock.EntryGroup
	Bought []stock.Entry
	Chips  []stock.Chip
}

// Clearable tells whether some bought entry has no section, so that the list offers to clear those.
func (d shoppingData) Clearable() bool {
	for _, e := range d.Bought {
		if e.Section == "" {
			return true
		}
	}
	return false
}

// entriesData is the list's entries part, from the current list.
func (s *server) entriesData(r *http.Request) (shoppingData, error) {
	var d shoppingData
	entries, err := s.stock.Entries(r.Context())
	if err != nil {
		return d, err
	}
	d.ToBuy, d.Bought = stock.GroupEntries(entries)
	d.Chips = stock.ShopChips(entries)
	return d, nil
}

func (s *server) shopping(w http.ResponseWriter, r *http.Request, form entryForm) {
	d, err := s.entriesData(r)
	if err == nil {
		d.Names, err = s.stock.ListNames(r.Context())
	}
	if err == nil {
		d.Shops, err = s.stock.Shops(r.Context())
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	d.Form = form
	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}
	s.render(w, r, status, "shopping", d)
}

// entries answers the list's refresh with its entries part.
func (s *server) entries(w http.ResponseWriter, r *http.Request) {
	d, err := s.entriesData(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.renderPart(w, "shopping", "entries", d)
}

func (s *server) addEntry(w http.ResponseWriter, r *http.Request) {
	form := entryForm{Name: r.PostFormValue("name"), Quantity: r.PostFormValue("quantity"), Unit: r.PostFormValue("unit"),
		Section: r.PostFormValue("section"), Shop: r.PostFormValue("shop")}
	e := stock.Entry{Name: form.Name, Unit: form.Unit, Section: stock.Section(form.Section), Shop: form.Shop}
	var err error
	if strings.TrimSpace(form.Quantity) != "" {
		e.Quantity, err = stock.ParseQuantity(form.Quantity)
	}
	if err == nil {
		err = s.stock.AddEntry(r.Context(), currentUser(r).ID, e)
	}
	if msg := refusal(err); msg != "" {
		form.Error = msg
		s.shopping(w, r, form)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/shopping", http.StatusSeeOther)
}

// changeEntry applies one of an entry's buttons: tick, untick or remove.
func (s *server) changeEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}
	switch r.PathValue("action") {
	case "tick":
		err = s.stock.Tick(r.Context(), id, true)
	case "untick":
		err = s.stock.Tick(r.Context(), id, false)
	case "remove":
		err = s.stock.RemoveEntry(r.Context(), id)
	default:
		http.NotFound(w, r)
		return
	}
	// With HTMX, an entry someone else removed or merged meanwhile just gets the current list.
	if errors.Is(err, stock.ErrNotFound) && r.Header.Get("HX-Request") == "" {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}
	if err != nil && !errors.Is(err, stock.ErrNotFound) {
		s.fail(w, err)
		return
	}
	s.answerEntries(w, r)
}

func (s *server) clearBought(w http.ResponseWriter, r *http.Request) {
	if err := s.stock.ClearBought(r.Context()); err != nil {
		s.fail(w, err)
		return
	}
	s.answerEntries(w, r)
}

// answerEntries answers an entry's button with the list's entries, or, without HTMX, with a redirect to the list.
func (s *server) answerEntries(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, "/shopping", http.StatusSeeOther)
		return
	}
	s.entries(w, r)
}

// listProduct answers the add form's name field with the section and the shop an entry of that name can take, or with
// nothing when there is neither.
func (s *server) listProduct(w http.ResponseWriter, r *http.Request) {
	section, shop, err := s.stock.ListProduct(r.Context(), r.URL.Query().Get("name"))
	if err != nil {
		s.fail(w, err)
		return
	}
	if section == "" && shop == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.renderPart(w, "shopping", "listProduct", entryForm{Section: string(section), Shop: shop})
}

type putAwayData struct {
	Entry      stock.Entry
	Form       itemForm
	Categories []string
	Locations  []string
}

// boughtEntry returns the bought entry with a section named by the path, answering 404 for any other.
func (s *server) boughtEntry(w http.ResponseWriter, r *http.Request) (stock.Entry, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var e stock.Entry
	if err == nil {
		e, err = s.stock.Entry(r.Context(), id)
	}
	if err == nil && (!e.Bought || e.Section == "") {
		err = stock.ErrNotFound
	}
	if err != nil && !errors.Is(err, stock.ErrNotFound) {
		s.fail(w, err)
		return e, false
	}
	if err != nil {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return e, false
	}
	return e, true
}

func (s *server) renderPutAway(w http.ResponseWriter, r *http.Request, status int, d putAwayData) {
	var err error
	if d.Categories, err = s.stock.Categories(r.Context(), d.Entry.Section); err == nil {
		d.Locations, err = s.stock.Locations(r.Context(), d.Entry.Section)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, status, "putaway", d)
}

// putAwayForm shows the section's add form filled in from the entry and from what the section remembers of it.
func (s *server) putAwayForm(w http.ResponseWriter, r *http.Request) {
	e, ok := s.boughtEntry(w, r)
	if !ok {
		return
	}
	p, _, err := s.stock.Product(r.Context(), e.Section, e.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	form := formOf(stock.Values{Name: e.Name, Quantity: max(e.Quantity, stock.One), Unit: e.Unit, Category: p.Category,
		Location: p.Location, OpenedDays: p.OpenedDays})
	s.renderPutAway(w, r, http.StatusOK, putAwayData{Entry: e, Form: form})
}

func (s *server) putAway(w http.ResponseWriter, r *http.Request) {
	e, ok := s.boughtEntry(w, r)
	if !ok {
		return
	}
	form := readForm(r)
	v, err := form.values()
	var id int64
	if err == nil {
		id, err = s.stock.PutAway(r.Context(), e.ID, currentUser(r).ID, v)
	}
	if msg := refusal(err); msg != "" {
		form.Error = msg
		s.renderPutAway(w, r, http.StatusBadRequest, putAwayData{Entry: e, Form: form})
		return
	}
	if errors.Is(err, stock.ErrNotFound) { // put away by someone else meanwhile
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/"+string(e.Section)+"#item-"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}
