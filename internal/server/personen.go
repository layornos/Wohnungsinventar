package server

import (
	"net/http"
	"strings"

	"inventar/internal/store"
)

type personJSON struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Nutzer bool   `json:"nutzer"`
}

func alsPersonJSON(p store.Person) personJSON {
	return personJSON{ID: p.ID, Name: p.Name, Nutzer: p.Nutzer}
}

func alsPersonenJSON(personen []store.Person) []personJSON {
	out := make([]personJSON, 0, len(personen))
	for _, p := range personen {
		out = append(out, alsPersonJSON(p))
	}
	return out
}

func (s *server) personenAuflisten(w http.ResponseWriter, r *http.Request) {
	personen, err := s.Store.PersonenAuflisten(r.Context())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"personen": alsPersonenJSON(personen)})
}

// leseName liest einen Body {"name": …} und verlangt einen nicht leeren Namen.
func leseName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req struct {
		Name string `json:"name"`
	}
	if !leseJSON(w, r, &req) {
		return "", false
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		nameFehlt(w)
		return "", false
	}
	return name, true
}

func (s *server) personAnlegen(w http.ResponseWriter, r *http.Request) {
	name, ok := leseName(w, r)
	if !ok {
		return
	}
	p, err := s.Store.PersonAnlegen(r.Context(), name, s.Jetzt())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsPersonJSON(p))
}

func (s *server) personUmbenennen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	name, ok := leseName(w, r)
	if !ok {
		return
	}
	p, err := s.Store.PersonUmbenennen(r.Context(), id, name)
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsPersonJSON(p))
}

func (s *server) personLoeschen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.PersonLoeschen(r.Context(), id); err != nil {
		s.storeFehler(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
