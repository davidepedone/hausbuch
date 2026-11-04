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
	Pages []page.Page
	Type  page.Type // the type filtered by, or empty
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
	s.render(w, r, http.StatusOK, "home", homeData{Pages: pages, Type: t})
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
	Content   page.Content
	Languages []string
	Error     string
	Taken     *page.TitleTakenError
}

func (s *server) newPage(w http.ResponseWriter, r *http.Request) {
	s.renderEditor(w, r, http.StatusOK, editData{Content: page.Content{
		Title: r.URL.Query().Get("title"), Type: page.Blank, Language: s.defaultLanguage,
	}})
}

func (s *server) editPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	s.renderEditor(w, r, http.StatusOK, editData{ID: p.ID, Content: p.Content})
}

func (s *server) renderEditor(w http.ResponseWriter, r *http.Request, status int, data editData) {
	data.Languages = page.Languages
	s.render(w, r, status, "edit", data)
}

// content reads the editor's form. Browsers send the text area's line breaks as CRLF; they are stored as LF.
func (s *server) content(r *http.Request, t page.Type) page.Content {
	lang := r.PostFormValue("language")
	if !slices.Contains(page.Languages, lang) {
		lang = s.defaultLanguage
	}
	return page.Content{
		Title:    r.PostFormValue("title"),
		Type:     t,
		Language: lang,
		Body:     strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n"),
	}
}

func (s *server) createPage(w http.ResponseWriter, r *http.Request) {
	c := s.content(r, page.Blank)
	id, err := s.pages.Create(r.Context(), currentUser(r).ID, c)
	if s.refused(w, r, editData{Content: c}, err) {
		return
	}
	http.Redirect(w, r, pageURL(id), http.StatusSeeOther)
}

func (s *server) savePage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	c := s.content(r, p.Type)
	err := s.pages.Update(r.Context(), p.ID, currentUser(r).ID, c)
	if s.refused(w, r, editData{ID: p.ID, Content: c}, err) {
		return
	}
	http.Redirect(w, r, pageURL(p.ID), http.StatusSeeOther)
}

// refused answers a save that failed: with the editor, still holding the user's text, when the title is the
// problem. It reports whether there was anything to answer.
func (s *server) refused(w http.ResponseWriter, r *http.Request, data editData, err error) bool {
	var taken *page.TitleTakenError
	status := http.StatusBadRequest
	switch {
	case err == nil:
		return false
	case errors.Is(err, page.ErrEmptyTitle):
		data.Error = "Give the page a title."
	case errors.As(err, &taken):
		data.Taken, status = taken, http.StatusConflict
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
	body, err := s.renderBody(r, strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n"))
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
