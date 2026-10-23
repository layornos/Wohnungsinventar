package server

import (
	"net/http"
	"strconv"
	"strings"

	"inventar/internal/store"
)

type pfadJSON struct {
	Wohnung   benanntJSON   `json:"wohnung"`
	Raum      benanntJSON   `json:"raum"`
	Lagerorte []benanntJSON `json:"lagerorte"`
}

func alsPfadJSON(p store.Pfad) pfadJSON {
	return pfadJSON{Wohnung: alsBenanntJSON(p.Wohnung), Raum: alsBenanntJSON(p.Raum), Lagerorte: alsBenannteJSON(p.Lagerorte)}
}

type lagerortJSON struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Pfad      pfadJSON      `json:"pfad"`
	Moebel    *benanntJSON  `json:"moebel"`
	Unter     []benanntJSON `json:"unter"`
	Unterbaum []int64       `json:"unterbaum"`
}

func alsLagerortJSON(l store.LagerortDetail) lagerortJSON {
	out := lagerortJSON{ID: l.ID, Name: l.Name, Pfad: alsPfadJSON(l.Pfad), Unter: alsBenannteJSON(l.Unter), Unterbaum: l.Unterbaum}
	if l.Moebel != nil {
		m := alsBenanntJSON(*l.Moebel)
		out.Moebel = &m
	}
	return out
}

func ungueltigerPlatz(w http.ResponseWriter, meldung string) {
	schreibeFehler(w, http.StatusBadRequest, "ungueltiger_platz", meldung)
}

func elternFehlt(w http.ResponseWriter) {
	ungueltigerPlatz(w, "Ein Lagerort liegt entweder in einem Raum oder in einem anderen Lagerort")
}

// platzAus macht aus optionalen IDs eine Platzierung; beide zugleich sind nicht erlaubt.
func platzAus(w http.ResponseWriter, lagerortID, raumID int64) (store.Platz, bool) {
	if lagerortID != 0 && raumID != 0 {
		ungueltigerPlatz(w, "Entweder ein Lagerort oder ein Raum, nicht beides")
		return store.Platz{}, false
	}
	return store.Platz{LagerortID: lagerortID, RaumID: raumID}, true
}

// platzAusFormular liest die Platzierung aus den Formularfeldern "lagerort" und "raum".
func platzAusFormular(w http.ResponseWriter, f formular) (store.Platz, bool) {
	var ids [2]int64
	for i, feld := range []string{"lagerort", "raum"} {
		if wert := f.feld(feld); wert != "" {
			id, err := strconv.ParseInt(wert, 10, 64)
			if err != nil {
				schreibeFehler(w, http.StatusBadRequest, "unbekannter_platz", "Diesen Raum oder Lagerort gibt es nicht")
				return store.Platz{}, false
			}
			ids[i] = id
		}
	}
	return platzAus(w, ids[0], ids[1])
}

func (s *server) platzierungSetzen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		LagerortID int64 `json:"lagerortId"`
		RaumID     int64 `json:"raumId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	p, ok := platzAus(w, req.LagerortID, req.RaumID)
	if !ok {
		return
	}
	item, err := s.Store.PlatzierungSetzen(r.Context(), id, p, s.Jetzt())
	s.itemAntworten(w, r, item, err)
}

func (s *server) lagerortAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		RaumID   int64  `json:"raumId"`
		ElternID int64  `json:"elternId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	neu := store.NeuerLagerort{Name: strings.TrimSpace(req.Name), RaumID: req.RaumID, ElternID: req.ElternID}
	if neu.Name == "" {
		nameFehlt(w)
		return
	}
	if (neu.RaumID == 0) == (neu.ElternID == 0) {
		elternFehlt(w)
		return
	}
	l, err := s.Store.LagerortAnlegen(r.Context(), neu, s.Jetzt())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsLagerortJSON(l))
}

func (s *server) lagerortAbrufen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	l, err := s.Store.LagerortAbrufen(r.Context(), id)
	s.lagerortAntworten(w, r, l, err)
}

func (s *server) lagerortAntworten(w http.ResponseWriter, r *http.Request, l store.LagerortDetail, err error) {
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsLagerortJSON(l))
}

// moebelSetzen ordnet den Lagerort einem Item zu; {"itemId": null} löst die Zuordnung.
func (s *server) moebelSetzen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		ItemID int64 `json:"itemId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	l, err := s.Store.MoebelSetzen(r.Context(), id, req.ItemID)
	s.lagerortAntworten(w, r, l, err)
}

func (s *server) lagerortVerschieben(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		RaumID   int64 `json:"raumId"`
		ElternID int64 `json:"elternId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	if (req.RaumID == 0) == (req.ElternID == 0) {
		elternFehlt(w)
		return
	}
	l, err := s.Store.LagerortVerschieben(r.Context(), id, req.RaumID, req.ElternID)
	s.lagerortAntworten(w, r, l, err)
}

func (s *server) lagerortLoeschen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.LagerortLoeschen(r.Context(), id, s.Jetzt()); err != nil {
		s.storeFehler(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type platzJSON struct {
	Art  store.PlatzArt `json:"art"`
	ID   int64          `json:"id"`
	Pfad string         `json:"pfad"`
}

func (s *server) plaetzeAuflisten(w http.ResponseWriter, r *http.Request) {
	plaetze, err := s.Store.PlaetzeAuflisten(r.Context())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	out := make([]platzJSON, 0, len(plaetze))
	for _, p := range plaetze {
		out = append(out, platzJSON{Art: p.Art, ID: p.ID, Pfad: p.Pfad})
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"plaetze": out})
}
