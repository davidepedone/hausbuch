package web

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/davidepedone/hausbuch/internal/markdown"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
)

// renderBody turns a body into HTML, with [[Title]] links leading to the page or, when there is none, to creating it.
func (s *server) renderBody(r *http.Request, body string) (template.HTML, error) {
	return markdown.Render(body, func(title string) (markdown.Link, error) {
		id, found, err := s.pages.Resolve(r.Context(), title)
		if err != nil || !found {
			return markdown.Link{Href: newPageURL(title), Missing: true}, err
		}
		return markdown.Link{Href: pageURL(id)}, nil
	})
}

func pageURL(id int64) string { return "/pages/" + strconv.FormatInt(id, 10) }

func newPageURL(title string) string { return "/pages/new?title=" + url.QueryEscape(title) }

type homeData struct {
	Warnings []stock.Item // expired and expiring items
	Pages    []page.Page
	Type     page.Type // the type filtered by, or empty
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	t := page.Type(r.URL.Query().Get("type"))
	if !slices.Contains(page.Types, t) {
		t = ""
	}
	pages, err := s.pages.List(r.Context(), t)
	if err != nil {
		s.fail(w, err)
		return
	}
	warnings, err := s.stock.Warnings(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "home", homeData{Warnings: warnings, Pages: pages, Type: t})
}

type viewData struct {
	Page      page.Page
	Body      template.HTML
	Backlinks []page.Page
}

func (s *server) viewPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	body, err := s.renderBody(r, p.Body)
	if err != nil {
		s.fail(w, err)
		return
	}
	backlinks, err := s.pages.Backlinks(r.Context(), p.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "page", viewData{Page: p, Body: body, Backlinks: backlinks})
}

// pageFromPath returns the page named by the {id} of the path, or answers 404.
func (s *server) pageFromPath(w http.ResponseWriter, r *http.Request) (page.Page, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Page not found.", http.StatusNotFound)
		return page.Page{}, false
	}
	p, err := s.pages.Get(r.Context(), id)
	if errors.Is(err, page.ErrNotFound) {
		http.Error(w, "Page not found.", http.StatusNotFound)
		return page.Page{}, false
	}
	if err != nil {
		s.fail(w, err)
		return page.Page{}, false
	}
	return p, true
}

type editData struct {
	ID        int64 // 0 for a new page
	Revision  int   // the revision editing started from
	Content   page.Content
	Skeleton  string // for a new page, the skeleton its body was prefilled with
	Languages []string
	Error     string
	Taken     *page.TitleTakenError
	Conflict  *page.ConflictError
	Changes   []page.DiffLine // with a conflict, what changed in the body since the revision editing started from
}

func (s *server) newPage(w http.ResponseWriter, r *http.Request) {
	body := page.Skeleton(page.Blank, s.defaultLanguage)
	s.renderEditor(w, r, http.StatusOK, editData{Content: page.Content{
		Title: r.URL.Query().Get("title"), Type: page.Blank, Language: s.defaultLanguage, Body: body,
	}, Skeleton: body})
}

// skeleton answers a change of type or language in the new page form with the body to show: the skeleton of the new
// choice while the body is still the skeleton it was prefilled with, the body unchanged once the user has written.
func (s *server) skeleton(w http.ResponseWriter, r *http.Request) {
	c, prefilled := s.content(r), lf(r.PostFormValue("skeleton"))
	if c.Body == prefilled {
		c.Body = page.Skeleton(c.Type, c.Language)
		prefilled = c.Body
	}
	s.renderPart(w, "edit", "body", editData{Content: c, Skeleton: prefilled})
}

type editingData struct {
	Page    page.Page
	Editors []editingView
}

type editingView struct{ Name, For string }

// editPage opens the editor, unless someone else has it open: then it warns first, and opens it only "anyway".
func (s *server) editPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	u, now := currentUser(r), s.pages.Now()
	if others := s.editors.others(p.ID, u.ID, now); len(others) > 0 && r.URL.Query().Get("anyway") == "" {
		data := editingData{Page: p}
		for _, ed := range others {
			data.Editors = append(data.Editors, editingView{Name: ed.name, For: duration(now.Sub(ed.since))})
		}
		s.render(w, r, http.StatusOK, "editing", data)
		return
	}
	s.editors.touch(p.ID, u.ID, u.DisplayName, now)
	s.renderEditor(w, r, http.StatusOK, editData{ID: p.ID, Revision: p.Revision, Content: p.Content})
}

// stillEditing is the open editor's sign of activity.
func (s *server) stillEditing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Page not found.", http.StatusNotFound)
		return
	}
	u := currentUser(r)
	s.editors.touch(id, u.ID, u.DisplayName, s.pages.Now())
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) cancelEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Page not found.", http.StatusNotFound)
		return
	}
	s.editors.close(id, currentUser(r).ID)
	http.Redirect(w, r, pageURL(id), http.StatusSeeOther)
}

func (s *server) renderEditor(w http.ResponseWriter, r *http.Request, status int, data editData) {
	data.Languages = page.Languages
	s.render(w, r, status, "edit", data)
}

// content reads the editor's form.
func (s *server) content(r *http.Request) page.Content {
	t := page.Type(r.PostFormValue("type"))
	if !slices.Contains(page.Types, t) {
		t = page.Blank
	}
	lang := r.PostFormValue("language")
	if !slices.Contains(page.Languages, lang) {
		lang = s.defaultLanguage
	}
	return page.Content{Title: r.PostFormValue("title"), Type: t, Language: lang, Body: lf(r.PostFormValue("body"))}
}

// lf turns the CRLF line breaks browsers send for form fields into the LF ones stored.
func lf(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func (s *server) createPage(w http.ResponseWriter, r *http.Request) {
	c := s.content(r)
	id, err := s.pages.Create(r.Context(), currentUser(r).ID, c)
	if s.refused(w, r, editData{Content: c, Skeleton: lf(r.PostFormValue("skeleton"))}, err) {
		return
	}
	http.Redirect(w, r, pageURL(id), http.StatusSeeOther)
}

func (s *server) savePage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	c := s.content(r)
	base, _ := strconv.Atoi(r.PostFormValue("revision"))
	err := s.pages.Update(r.Context(), p.ID, currentUser(r).ID, base, c)
	if s.refused(w, r, editData{ID: p.ID, Revision: base, Content: c}, err) {
		return
	}
	s.editors.close(p.ID, currentUser(r).ID)
	http.Redirect(w, r, pageURL(p.ID), http.StatusSeeOther)
}

// refused answers a save that failed: with the editor, still holding the user's text, when the title or a change
// made meanwhile is the problem. It reports whether there was anything to answer.
func (s *server) refused(w http.ResponseWriter, r *http.Request, data editData, err error) bool {
	var taken *page.TitleTakenError
	var conflict *page.ConflictError
	status := http.StatusBadRequest
	switch {
	case err == nil:
		return false
	case errors.Is(err, page.ErrEmptyTitle):
		data.Error = "Give the page a title."
	case errors.As(err, &taken):
		data.Taken, status = taken, http.StatusConflict
	case errors.As(err, &conflict):
		from, err1 := s.pages.Revision(r.Context(), data.ID, data.Revision)
		to, err2 := s.pages.Revision(r.Context(), data.ID, conflict.Revision)
		if err := errors.Join(err1, err2); err != nil {
			s.fail(w, err)
			return true
		}
		// Saving again from the current revision is what the user does once they have reviewed the change.
		data.Conflict, data.Changes, data.Revision = conflict, page.Diff(from.Body, to.Body), conflict.Revision
		status = http.StatusConflict
	case errors.Is(err, page.ErrNotFound):
		http.Error(w, "Page not found.", http.StatusNotFound)
		return true
	default:
		s.fail(w, err)
		return true
	}
	s.renderEditor(w, r, status, data)
	return true
}

// preview answers the editor's preview request with the body as the page view would show it.
func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	body, err := s.renderBody(r, lf(r.PostFormValue("body")))
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (s *server) deletePage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	if err := s.pages.Delete(r.Context(), p.ID, currentUser(r).ID); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) deletedPages(w http.ResponseWriter, r *http.Request) {
	pages, err := s.pages.Deleted(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "deleted", pages)
}

func (s *server) restorePage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err == nil {
		err = s.pages.Restore(r.Context(), id)
	}
	if err != nil {
		if errors.Is(err, page.ErrNotFound) || errors.Is(err, strconv.ErrSyntax) || errors.Is(err, strconv.ErrRange) {
			http.Error(w, "No deleted page to restore.", http.StatusNotFound)
			return
		}
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, pageURL(id), http.StatusSeeOther)
}
