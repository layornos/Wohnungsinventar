package server

import (
	"net/http"
	"strings"

	"inventar/internal/store"
)

type personJSON struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	IsUser bool   `json:"isUser"`
}

func toPersonJSON(p store.Person) personJSON {
	return personJSON{ID: p.ID, Name: p.Name, IsUser: p.IsUser}
}

func toPersonsJSON(persons []store.Person) []personJSON {
	out := make([]personJSON, 0, len(persons))
	for _, p := range persons {
		out = append(out, toPersonJSON(p))
	}
	return out
}

func (s *server) listPersons(w http.ResponseWriter, r *http.Request) {
	persons, err := s.Store.ListPersons(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"persons": toPersonsJSON(persons)})
}

// readName reads a body {"name": …} and requires a non-empty name.
func readName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return "", false
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		nameMissing(w)
		return "", false
	}
	return name, true
}

func (s *server) createPerson(w http.ResponseWriter, r *http.Request) {
	name, ok := readName(w, r)
	if !ok {
		return
	}
	p, err := s.Store.CreatePerson(r.Context(), name, s.Now())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPersonJSON(p))
}

func (s *server) renamePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	name, ok := readName(w, r)
	if !ok {
		return
	}
	p, err := s.Store.RenamePerson(r.Context(), id, name)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPersonJSON(p))
}

func (s *server) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.DeletePerson(r.Context(), id); err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
