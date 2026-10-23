package server

import (
	"context"
	"net/http"

	"inventar/internal/store"
)

type ortJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func alsOrtJSON(o store.Ort) ortJSON { return ortJSON{ID: o.ID, Name: o.Name} }

func alsOrteJSON(orte []store.Ort) []ortJSON {
	out := make([]ortJSON, 0, len(orte))
	for _, o := range orte {
		out = append(out, alsOrtJSON(o))
	}
	return out
}

type wohnungJSON struct {
	ID     int64     `json:"id"`
	Name   string    `json:"name"`
	Raeume []ortJSON `json:"raeume"`
}

type lagerortKnotenJSON struct {
	ID     int64                `json:"id"`
	Name   string               `json:"name"`
	Moebel *ortJSON             `json:"moebel"`
	Unter  []lagerortKnotenJSON `json:"unter"`
}

func alsBaumJSON(lagerorte []store.Lagerort) []lagerortKnotenJSON {
	out := make([]lagerortKnotenJSON, 0, len(lagerorte))
	for _, l := range lagerorte {
		k := lagerortKnotenJSON{ID: l.ID, Name: l.Name, Unter: alsBaumJSON(l.Unter)}
		if l.Moebel != nil {
			m := alsOrtJSON(*l.Moebel)
			k.Moebel = &m
		}
		out = append(out, k)
	}
	return out
}

func (s *server) wohnungenAuflisten(w http.ResponseWriter, r *http.Request) {
	wohnungen, err := s.Store.WohnungenAuflisten(r.Context())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	out := make([]wohnungJSON, 0, len(wohnungen))
	for _, wo := range wohnungen {
		out = append(out, wohnungJSON{ID: wo.ID, Name: wo.Name, Raeume: alsOrteJSON(wo.Raeume)})
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"wohnungen": out})
}

func (s *server) wohnungAnlegen(w http.ResponseWriter, r *http.Request) {
	name, ok := leseName(w, r)
	if !ok {
		return
	}
	o, err := s.Store.WohnungAnlegen(r.Context(), name, s.Jetzt())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsOrtJSON(o))
}

func (s *server) raumAnlegen(w http.ResponseWriter, r *http.Request) {
	wohnungID, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	name, ok := leseName(w, r)
	if !ok {
		return
	}
	o, err := s.Store.RaumAnlegen(r.Context(), wohnungID, name, s.Jetzt())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsOrtJSON(o))
}

// umbenennen benennt die Wohnung, den Raum oder Lagerort mit der ID im Pfad um.
func (s *server) umbenennen(aendern func(ctx context.Context, id int64, name string) (store.Ort, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pfadID(w, r, "id")
		if !ok {
			return
		}
		name, ok := leseName(w, r)
		if !ok {
			return
		}
		o, err := aendern(r.Context(), id, name)
		if err != nil {
			s.storeFehler(w, r, err)
			return
		}
		schreibeJSON(w, http.StatusOK, alsOrtJSON(o))
	}
}

// loeschen löscht das Element mit der ID im Pfad.
func (s *server) loeschen(loeschen func(ctx context.Context, id int64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pfadID(w, r, "id")
		if !ok {
			return
		}
		if err := loeschen(r.Context(), id); err != nil {
			s.storeFehler(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) raumAbrufen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	raum, err := s.Store.RaumAbrufen(r.Context(), id)
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{
		"id": raum.ID, "name": raum.Name, "wohnung": alsOrtJSON(raum.Wohnung), "lagerorte": alsBaumJSON(raum.Lagerorte),
	})
}
