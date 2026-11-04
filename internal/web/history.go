package web

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/davidepedone/hausbuch/internal/page"
)

type historyData struct {
	Page      page.Page
	Revisions []page.Revision
}

func (s *server) history(w http.ResponseWriter, r *http.Request) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return
	}
	revs, err := s.pages.Revisions(r.Context(), p.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "history", historyData{Page: p, Revisions: revs})
}

type revisionData struct {
	Page     page.Page
	Revision page.Revision
	Body     template.HTML
	Changes  []page.DiffLine
}

// revisionFromPath returns the page and its revision named by the path, or answers 404.
func (s *server) revisionFromPath(w http.ResponseWriter, r *http.Request) (page.Page, page.Revision, bool) {
	p, ok := s.pageFromPath(w, r)
	if !ok {
		return page.Page{}, page.Revision{}, false
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	var rev page.Revision
	if err == nil {
		rev, err = s.pages.Revision(r.Context(), p.ID, n)
	}
	if errors.Is(err, page.ErrNotFound) || errors.Is(err, strconv.ErrSyntax) || errors.Is(err, strconv.ErrRange) {
		http.Error(w, "Revision not found.", http.StatusNotFound)
		return page.Page{}, page.Revision{}, false
	}
	if err != nil {
		s.fail(w, err)
		return page.Page{}, page.Revision{}, false
	}
	return p, rev, true
}

func (s *server) revision(w http.ResponseWriter, r *http.Request) {
	p, rev, ok := s.revisionFromPath(w, r)
	if !ok {
		return
	}
	body, err := s.renderBody(r, rev.Body)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, r, http.StatusOK, "revision", revisionData{Page: p, Revision: rev, Body: body})
}

// compare shows what changed in the body from the revision to the current version.
func (s *server) compare(w http.ResponseWriter, r *http.Request) {
	p, rev, ok := s.revisionFromPath(w, r)
	if !ok {
		return
	}
	s.render(w, r, http.StatusOK, "compare",
		revisionData{Page: p, Revision: rev, Changes: page.Diff(rev.Body, p.Body)})
}

// restoreRevision saves the revision's content as the page's new revision.
func (s *server) restoreRevision(w http.ResponseWriter, r *http.Request) {
	p, rev, ok := s.revisionFromPath(w, r)
	if !ok {
		return
	}
	err := s.pages.Update(r.Context(), p.ID, currentUser(r).ID, p.Revision, rev.Content)
	if s.refused(w, r, editData{ID: p.ID, Revision: p.Revision, Content: rev.Content}, err) {
		return
	}
	http.Redirect(w, r, pageURL(p.ID), http.StatusSeeOther)
}

// joinNames lists names as "Anna", "Anna and Marco", "Anna, Marco and Paolo".
func joinNames(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
