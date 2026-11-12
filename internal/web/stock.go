package web

import (
	"cmp"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// routeStock registers the pages of every section.
func (s *server) routeStock(mux *http.ServeMux) {
	for _, sec := range stock.Sections {
		base := "/" + string(sec)
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) { s.sectionPage(w, r, sec) })
		mux.HandleFunc("POST "+base+"/items", func(w http.ResponseWriter, r *http.Request) { s.addItem(w, r, sec) })
		mux.HandleFunc("GET "+base+"/items/{id}", func(w http.ResponseWriter, r *http.Request) { s.editItem(w, r, sec) })
		mux.HandleFunc("POST "+base+"/items/{id}", func(w http.ResponseWriter, r *http.Request) { s.saveItem(w, r, sec) })
		mux.HandleFunc("GET "+base+"/items/{id}/move", func(w http.ResponseWriter, r *http.Request) { s.moveForm(w, r, sec) })
		mux.HandleFunc("POST "+base+"/items/{id}/move", func(w http.ResponseWriter, r *http.Request) { s.moveItem(w, r, sec) })
		mux.HandleFunc("POST "+base+"/items/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
			s.changeItem(w, r, sec)
		})
		mux.HandleFunc("POST "+base+"/items/restore", func(w http.ResponseWriter, r *http.Request) {
			s.restoreItem(w, r, sec)
		})
		mux.HandleFunc("POST "+base+"/items/to-list", func(w http.ResponseWriter, r *http.Request) {
			s.listRemoved(w, r, sec)
		})
		mux.HandleFunc("GET "+base+"/product", func(w http.ResponseWriter, r *http.Request) { s.product(w, r, sec) })
	}
}

func itemURL(it stock.Item) string {
	return "/" + string(it.Section) + "/items/" + strconv.FormatInt(it.ID, 10)
}

// itemForm is an item's form as the user filled it in, kept as text so that a refused form shows what was entered.
type itemForm struct {
	Name, Quantity, Unit, Category, Location, ExpiresOn, OpenedOn, OpenedDays string
	ExpiresMonth, ExpiresYear                                                 string // the expiry as a month and a year
	Section                                                                   string // where the edit page saves the row
	Error                                                                     string
}

func formOf(v stock.Values) itemForm {
	f := itemForm{Name: v.Name, Quantity: v.Quantity.String(), Unit: v.Unit, Category: v.Category, Location: v.Location,
		ExpiresOn: inputDate(v.ExpiresOn), OpenedOn: inputDate(v.OpenedOn)}
	if v.OpenedDays > 0 {
		f.OpenedDays = strconv.Itoa(v.OpenedDays)
	}
	return f
}

func readForm(r *http.Request) itemForm {
	return itemForm{Name: r.PostFormValue("name"), Quantity: r.PostFormValue("quantity"), Unit: r.PostFormValue("unit"),
		Category: r.PostFormValue("category"), Location: r.PostFormValue("location"), ExpiresOn: r.PostFormValue("expires_on"),
		OpenedOn: r.PostFormValue("opened_on"), OpenedDays: r.PostFormValue("opened_days"),
		ExpiresMonth: r.PostFormValue("expires_month"), ExpiresYear: r.PostFormValue("expires_year")}
}

var (
	errDate           = errors.New("bad date")
	errExpiryNotWhole = errors.New("expiry month without a year, or a year without a month")
)

// values turns the form into values, reporting a field that isn't a number or a date.
func (f itemForm) values() (stock.Values, error) {
	v := stock.Values{Name: f.Name, Unit: f.Unit, Category: f.Category, Location: f.Location}
	var err error
	if v.Quantity, err = stock.ParseQuantity(f.Quantity); err != nil {
		return v, err
	}
	if days := strings.TrimSpace(f.OpenedDays); days != "" {
		if v.OpenedDays, err = strconv.Atoi(days); err != nil {
			return v, stock.ErrOpenedDays
		}
	}
	for _, d := range []struct {
		text string
		to   *time.Time
	}{{f.ExpiresOn, &v.ExpiresOn}, {f.OpenedOn, &v.OpenedOn}} {
		if d.text = strings.TrimSpace(d.text); d.text != "" {
			if *d.to, err = time.Parse(time.DateOnly, d.text); err != nil {
				return v, errDate
			}
		}
	}
	if v.ExpiresOn.IsZero() {
		v.ExpiresOn, err = monthEnd(f.ExpiresMonth, f.ExpiresYear)
	}
	return v, err
}

// expiryField is how the expiry field of a form shows its value: as a month and a year, or as an exact day when it is
// a date other than a month's last day.
type expiryField struct {
	Exact         bool
	Date          string
	Months, Years []option
}

type option struct {
	Value, Label string
	Selected     bool
}

// Expiry lays out the form's expiry for its field. The years run from 2 back, for an item found already expired, to 10
// ahead, and include the one shown.
func (f itemForm) Expiry() expiryField {
	e := expiryField{Date: f.ExpiresOn}
	month, year := strings.TrimSpace(f.ExpiresMonth), strings.TrimSpace(f.ExpiresYear)
	if d, err := time.Parse(time.DateOnly, f.ExpiresOn); err == nil && d.AddDate(0, 0, 1).Day() == 1 {
		month, year = strconv.Itoa(int(d.Month())), strconv.Itoa(d.Year())
	} else if f.ExpiresOn != "" {
		e.Exact = true
	}
	for m := time.January; m <= time.December; m++ {
		v := strconv.Itoa(int(m))
		e.Months = append(e.Months, option{Value: v, Label: m.String(), Selected: v == month})
	}
	now := time.Now().Year()
	years := []int{}
	for y := now - 2; y <= now+10; y++ {
		years = append(years, y)
	}
	if y, err := strconv.Atoi(year); err == nil && !slices.Contains(years, y) {
		years = append(years, y)
		slices.Sort(years)
	}
	for _, y := range years {
		v := strconv.Itoa(y)
		e.Years = append(e.Years, option{Value: v, Label: v, Selected: v == year})
	}
	return e
}

// monthEnd is the last day of a month given as its number and its year, or no date when both are empty.
func monthEnd(month, year string) (time.Time, error) {
	month, year = strings.TrimSpace(month), strings.TrimSpace(year)
	if month == "" && year == "" {
		return time.Time{}, nil
	}
	if month == "" || year == "" {
		return time.Time{}, errExpiryNotWhole
	}
	m, err := strconv.Atoi(month)
	if err != nil || m < 1 || m > 12 {
		return time.Time{}, errDate
	}
	y, err := strconv.Atoi(year)
	if err != nil || y < 1 {
		return time.Time{}, errDate
	}
	// Day 0 of the next month is the month's last day.
	return time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC), nil
}

// refusal is the message for a form the store or the parsing refused, or "" for any other error.
func refusal(err error) string {
	switch {
	case errors.Is(err, stock.ErrEmptyName):
		return "Give the item a name."
	case errors.Is(err, stock.ErrQuantity):
		return "The quantity must be above 0, with at most one decimal."
	case errors.Is(err, stock.ErrOpenedDays):
		return "The days it lasts once opened must be a whole number of at least 1, or empty."
	case errors.Is(err, errDate):
		return "A date isn't valid."
	case errors.Is(err, errExpiryNotWhole):
		return "The expiry date isn't complete."
	case errors.Is(err, stock.ErrSection):
		return "Choose one of the sections."
	}
	return ""
}

type sectionData struct {
	Section    stock.Section
	Groups     []stock.Group
	Names      []string // product names used, for suggestions
	Categories []string
	Locations  []string
	Form       itemForm
	Removed    *stock.Item // the row just removed, to offer undoing it
	Listed     string      // the name of a product just added to the shopping list, to say so
	Added      string      // the name of an item just added, to say so and keep the add form open for the next one
}

// Grouped tells whether some row has a category, so that the list shows its groups and chips.
func (d sectionData) Grouped() bool {
	return len(d.Groups) > 1 || len(d.Groups) == 1 && d.Groups[0].Key != ""
}

func (s *server) sectionData(r *http.Request, sec stock.Section) (sectionData, error) {
	d := sectionData{Section: sec}
	items, err := s.stock.List(r.Context(), sec)
	if err != nil {
		return d, err
	}
	d.Groups = stock.GroupItems(items)
	if d.Names, err = s.stock.Names(r.Context(), sec); err != nil {
		return d, err
	}
	if d.Categories, err = s.stock.Categories(r.Context(), sec); err != nil {
		return d, err
	}
	d.Locations, err = s.stock.Locations(r.Context())
	return d, err
}

// sectionPage shows a section; after an item was added, its add form is ready for the next one, in the same location.
func (s *server) sectionPage(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	q := r.URL.Query()
	d, err := s.sectionData(r, sec)
	if err != nil {
		s.fail(w, err)
		return
	}
	if d.Added = q.Get("added"); d.Added != "" {
		d.Form.Location = q.Get("location")
	}
	s.render(w, r, http.StatusOK, "section", d)
}

func (s *server) section(w http.ResponseWriter, r *http.Request, sec stock.Section, form itemForm) {
	d, err := s.sectionData(r, sec)
	if err != nil {
		s.fail(w, err)
		return
	}
	d.Form = form
	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}
	s.render(w, r, status, "section", d)
}

func (s *server) addItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	form := readForm(r)
	v, err := form.values()
	if err == nil {
		_, err = s.stock.Add(r.Context(), sec, currentUser(r).ID, v)
	}
	if msg := refusal(err); msg != "" {
		form.Error = msg
		s.section(w, r, sec, form)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	added := url.Values{"added": {v.Name}}
	if v.Location != "" {
		added.Set("location", v.Location)
	}
	http.Redirect(w, r, "/"+string(sec)+"?"+added.Encode(), http.StatusSeeOther)
}

type itemData struct {
	Item       stock.Item
	Categories []string
	Locations  []string
	Form       itemForm
}

// itemFromPath returns the row named by the path, answering 404 when it isn't one of the section's.
func (s *server) itemFromPath(w http.ResponseWriter, r *http.Request, sec stock.Section) (stock.Item, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Item not found.", http.StatusNotFound)
		return stock.Item{}, false
	}
	it, err := s.stock.Get(r.Context(), sec, id)
	if errors.Is(err, stock.ErrNotFound) {
		http.Error(w, "Item not found.", http.StatusNotFound)
		return stock.Item{}, false
	}
	if err != nil {
		s.fail(w, err)
		return stock.Item{}, false
	}
	return it, true
}

func (s *server) renderItem(w http.ResponseWriter, r *http.Request, status int, it stock.Item, form itemForm) {
	d := itemData{Item: it, Form: form}
	var err error
	if d.Categories, err = s.stock.Categories(r.Context(), it.Section); err == nil {
		d.Locations, err = s.stock.Locations(r.Context())
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, status, "item", d)
}

func (s *server) editItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	if it, ok := s.itemFromPath(w, r, sec); ok {
		form := formOf(it.Values)
		form.Section = string(sec)
		s.renderItem(w, r, http.StatusOK, it, form)
	}
}

func (s *server) saveItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	it, ok := s.itemFromPath(w, r, sec)
	if !ok {
		return
	}
	form := readForm(r)
	// A form without the field, such as an edit page loaded by an older release, keeps the row in its section.
	form.Section = cmp.Or(r.PostFormValue("section"), string(sec))
	v, err := form.values()
	id := it.ID
	if err == nil {
		id, err = s.stock.Update(r.Context(), sec, it.ID, currentUser(r).ID, v, stock.Section(form.Section))
	}
	if msg := refusal(err); msg != "" {
		form.Error = msg
		s.renderItem(w, r, http.StatusBadRequest, it, form)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/"+form.Section+"#item-"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

type moveData struct {
	Item               stock.Item
	Locations          []string
	Quantity, Location string // as entered
	Error              string
}

func (s *server) renderMove(w http.ResponseWriter, r *http.Request, status int, d moveData) {
	var err error
	if d.Locations, err = s.stock.Locations(r.Context()); err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, status, "move", d)
}

func (s *server) moveForm(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	if it, ok := s.itemFromPath(w, r, sec); ok {
		s.renderMove(w, r, http.StatusOK, moveData{Item: it, Quantity: min(stock.One, it.Quantity).String()})
	}
}

func (s *server) moveItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	it, ok := s.itemFromPath(w, r, sec)
	if !ok {
		return
	}
	d := moveData{Item: it, Quantity: r.PostFormValue("quantity"), Location: r.PostFormValue("location")}
	quantity, err := stock.ParseQuantity(d.Quantity)
	id := it.ID
	if err == nil {
		id, err = s.stock.Move(r.Context(), sec, it.ID, currentUser(r).ID, quantity, d.Location)
	}
	switch {
	case errors.Is(err, stock.ErrQuantity):
		d.Error = "The quantity must be above 0 and at most " + it.Quantity.String() + ", with at most one decimal."
	case errors.Is(err, stock.ErrSameLocation):
		d.Error = "It's already kept there."
	case err != nil:
		s.fail(w, err)
		return
	default:
		http.Redirect(w, r, "/"+string(sec)+"#item-"+strconv.FormatInt(id, 10), http.StatusSeeOther)
		return
	}
	s.renderMove(w, r, http.StatusBadRequest, d)
}

// changeItem applies one of a row's buttons: decrease, increase, open, remove or list, which adds it to the shopping
// list.
func (s *server) changeItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	it, ok := s.itemFromPath(w, r, sec)
	if !ok {
		return
	}
	userID := currentUser(r).ID
	var removed *stock.Item
	var listed string
	var err error
	switch r.PathValue("action") {
	case "decrease":
		removed, err = s.stock.Decrease(r.Context(), sec, it.ID, userID)
	case "increase":
		err = s.stock.Increase(r.Context(), sec, it.ID, userID)
	case "open":
		err = s.stock.Open(r.Context(), sec, it.ID, userID)
	case "remove":
		var gone stock.Item
		if gone, err = s.stock.Remove(r.Context(), sec, it.ID); err == nil {
			removed = &gone
		}
	case "list":
		listed = it.Name
		_, err = s.stock.AddRowToList(r.Context(), sec, it.ID, userID)
	default:
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, stock.ErrNotFound) {
		http.Error(w, "Item not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.answerItems(w, r, sec, removed, listed)
}

// answerItems answers a row's button with the section's list, offering to undo a removal or saying that a product was
// added to the shopping list, or, without HTMX, with a redirect to the section.
func (s *server) answerItems(w http.ResponseWriter, r *http.Request, sec stock.Section, removed *stock.Item, listed string) {
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, "/"+string(sec), http.StatusSeeOther)
		return
	}
	d, err := s.sectionData(r, sec)
	if err != nil {
		s.fail(w, err)
		return
	}
	d.Removed, d.Listed = removed, listed
	s.renderPart(w, "section", "items", d)
}

// restoreItem undoes a removal from the values the undo offer carries.
func (s *server) restoreItem(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	v, err := readForm(r).values()
	createdBy, err1 := strconv.ParseInt(r.PostFormValue("created_by"), 10, 64)
	createdAt, err2 := strconv.ParseInt(r.PostFormValue("created_at"), 10, 64)
	if err := errors.Join(err, err1, err2); err != nil {
		http.Error(w, "The removed item can't be restored.", http.StatusBadRequest)
		return
	}
	it := stock.Item{Values: v, CreatedByID: createdBy, CreatedAt: time.Unix(createdAt, 0)}
	_, err = s.stock.Restore(r.Context(), sec, currentUser(r).ID, it)
	if refusal(err) != "" {
		http.Error(w, "The removed item can't be restored.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.answerItems(w, r, sec, nil, "")
}

// listRemoved adds a product just removed to the shopping list, from the values the undo offer carries.
func (s *server) listRemoved(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	name := r.PostFormValue("name")
	err := s.stock.AddEntry(r.Context(), currentUser(r).ID, stock.Entry{Name: name, Unit: r.PostFormValue("unit"), Section: sec})
	if refusal(err) != "" {
		http.Error(w, "The removed item can't be added to the shopping list.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.answerItems(w, r, sec, nil, strings.TrimSpace(name))
}

// product answers the add form's name field with the unit, category, location and shelf life the product last had, or with
// nothing for a name never used.
func (s *server) product(w http.ResponseWriter, r *http.Request, sec stock.Section) {
	p, found, err := s.stock.Product(r.Context(), sec, r.URL.Query().Get("name"))
	if err != nil {
		s.fail(w, err)
		return
	}
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	form := formOf(stock.Values{Unit: p.Unit, Category: p.Category, Location: p.Location, OpenedDays: p.OpenedDays})
	s.renderPart(w, "section", "productAnswer", sectionData{Section: sec, Form: form})
}

// inputDate is a calendar date as a date field takes it, or "" for none.
func inputDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateOnly)
}

// expiryNote tells how far an item's expiry date is, for an expired or expiring one.
func expiryNote(it stock.Item) string {
	switch {
	case it.State == stock.Expired:
		return "expired"
	case it.DaysLeft == 0:
		return "expires today"
	case it.DaysLeft == 1:
		return "expires tomorrow"
	}
	return "expires in " + strconv.Itoa(it.DaysLeft) + " days"
}
