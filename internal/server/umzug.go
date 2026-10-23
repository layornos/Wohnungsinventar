package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inventar/internal/store"
)

type umzugJSON struct {
	ID              int64       `json:"id"`
	AlteWohnung     benanntJSON `json:"alteWohnung"`
	NeueWohnung     benanntJSON `json:"neueWohnung"`
	Laufend         bool        `json:"laufend"`
	ErstelltAm      time.Time   `json:"erstelltAm"`
	AbgeschlossenAm *time.Time  `json:"abgeschlossenAm"`
	Boxen           []boxJSON   `json:"boxen"`
}

type boxJSON struct {
	ID          int64  `json:"id"`
	Nummer      int    `json:"nummer"`
	Bezeichnung string `json:"bezeichnung"`
	Anzahl      int    `json:"anzahl"`
}

func alsBoxJSON(b store.Box) boxJSON {
	return boxJSON{ID: b.ID, Nummer: b.Nummer, Bezeichnung: b.Bezeichnung, Anzahl: b.Anzahl}
}

func alsUmzugJSON(u store.Umzug) umzugJSON {
	out := umzugJSON{
		ID: u.ID, AlteWohnung: alsBenanntJSON(u.AlteWohnung), NeueWohnung: alsBenanntJSON(u.NeueWohnung),
		Laufend: u.Laufend, ErstelltAm: u.ErstelltAm, AbgeschlossenAm: u.AbgeschlossenAm, Boxen: []boxJSON{},
	}
	for _, b := range u.Boxen {
		out.Boxen = append(out.Boxen, alsBoxJSON(b))
	}
	return out
}

// laufenderUmzug liefert {"umzug": …} oder {"umzug": null}, wenn keiner läuft.
func (s *server) laufenderUmzug(w http.ResponseWriter, r *http.Request) {
	u, err := s.Store.LaufenderUmzug(r.Context())
	if errors.Is(err, store.ErrKeinUmzug) {
		schreibeJSON(w, http.StatusOK, map[string]any{"umzug": nil})
		return
	}
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"umzug": alsUmzugJSON(u)})
}

func (s *server) umzugAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AlteWohnungID int64 `json:"alteWohnungId"`
		NeueWohnungID int64 `json:"neueWohnungId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	u, err := s.Store.UmzugAnlegen(r.Context(), req.AlteWohnungID, req.NeueWohnungID, s.Jetzt())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsUmzugJSON(u))
}

func (s *server) umzuegeAuflisten(w http.ResponseWriter, r *http.Request) {
	umzuege, err := s.Store.UmzuegeAuflisten(r.Context())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	out := make([]umzugJSON, 0, len(umzuege))
	for _, u := range umzuege {
		out = append(out, alsUmzugJSON(u))
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"umzuege": out})
}

func (s *server) umzugAbrufen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	u, err := s.Store.UmzugAbrufen(r.Context(), id)
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsUmzugJSON(u))
}

func (s *server) boxAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bezeichnung string `json:"bezeichnung"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	b, err := s.Store.BoxAnlegen(r.Context(), strings.TrimSpace(req.Bezeichnung), s.Jetzt())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsBoxJSON(b))
}

func (s *server) boxUmbenennen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Bezeichnung string `json:"bezeichnung"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	b, err := s.Store.BoxUmbenennen(r.Context(), id, strings.TrimSpace(req.Bezeichnung))
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsBoxJSON(b))
}

type umzugsangabenJSON struct {
	Herkunftsraum *benanntJSON `json:"herkunftsraum"`
	Zielraum      *benanntJSON `json:"zielraum"`
	Box           *boxKurzJSON `json:"box"`
}

// boxKurzJSON ist eine Box in den Umzugsangaben eines Items, ohne Inhalt.
type boxKurzJSON struct {
	ID          int64  `json:"id"`
	Nummer      int    `json:"nummer"`
	Bezeichnung string `json:"bezeichnung"`
}

func alsUmzugsangabenJSON(a *store.Umzugsangaben) *umzugsangabenJSON {
	if a == nil {
		return nil
	}
	out := &umzugsangabenJSON{Herkunftsraum: alsBenanntZeiger(a.Herkunftsraum), Zielraum: alsBenanntZeiger(a.Zielraum)}
	if a.Box != nil {
		out.Box = &boxKurzJSON{ID: a.Box.ID, Nummer: a.Box.Nummer, Bezeichnung: a.Box.Bezeichnung}
	}
	return out
}

func alsBenanntZeiger(b *store.Benannt) *benanntJSON {
	if b == nil {
		return nil
	}
	j := alsBenanntJSON(*b)
	return &j
}

// leseAngabenAenderung liest {"herkunftsraumId", "zielraumId", "boxId"}: fehlende Felder bleiben,
// wie sie sind, null setzt zurück. weitere nennt zusätzliche Felder desselben Bodys und wohin sie gelesen werden.
func leseAngabenAenderung(w http.ResponseWriter, r *http.Request, weitere map[string]any) (store.AngabenAenderung, bool) {
	var felder map[string]json.RawMessage
	if !leseJSON(w, r, &felder) {
		return store.AngabenAenderung{}, false
	}
	var a store.AngabenAenderung
	for name, ziel := range map[string]**int64{"herkunftsraumId": &a.HerkunftsraumID, "zielraumId": &a.ZielraumID, "boxId": &a.BoxID} {
		roh, da := felder[name]
		if !da {
			continue
		}
		var id *int64
		if err := json.Unmarshal(roh, &id); err != nil {
			schreibeFehler(w, http.StatusBadRequest, "ungueltige_anfrage", "Ungültige ID für "+name)
			return store.AngabenAenderung{}, false
		}
		if id == nil {
			id = new(int64)
		}
		*ziel = id
	}
	for name, ziel := range weitere {
		if roh, da := felder[name]; da {
			if err := json.Unmarshal(roh, ziel); err != nil {
				schreibeFehler(w, http.StatusBadRequest, "ungueltige_anfrage", "Ungültiger Wert für "+name)
				return store.AngabenAenderung{}, false
			}
		}
	}
	return a, true
}

func (s *server) umzugsangabenSetzen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	a, ok := leseAngabenAenderung(w, r, nil)
	if !ok {
		return
	}
	item, err := s.Store.UmzugsangabenSetzen(r.Context(), id, a, s.Jetzt())
	s.itemAntworten(w, r, item, err)
}

// umzugsangabenSammeln setzt Box, Zielraum oder Herkunftsraum für mehrere Items ({"itemIds": […], …}).
func (s *server) umzugsangabenSammeln(w http.ResponseWriter, r *http.Request) {
	var itemIDs []int64
	a, ok := leseAngabenAenderung(w, r, map[string]any{"itemIds": &itemIDs})
	if !ok {
		return
	}
	if len(itemIDs) == 0 {
		schreibeFehler(w, http.StatusBadRequest, "ungueltige_anfrage", "Bitte mindestens ein Item wählen")
		return
	}
	if err := s.Store.UmzugsangabenSammeln(r.Context(), itemIDs, a, s.Jetzt()); err != nil {
		s.storeFehler(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// packlisteAntworten liefert die Box mit allen Items darin samt Umzugsangaben.
func (s *server) packlisteAntworten(w http.ResponseWriter, r *http.Request, b store.Box, err error) {
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	items, _, err := s.Store.ItemsAuflisten(r.Context(), store.ItemFilter{UmzugID: b.UmzugID, BoxID: b.ID, Anzahl: maxVollstaendig})
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"box": alsBoxJSON(b), "items": alsItemListeJSON(items)})
}

func (s *server) boxAbrufen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	b, err := s.Store.BoxAbrufen(r.Context(), id)
	s.packlisteAntworten(w, r, b, err)
}

func (s *server) boxPerNummer(w http.ResponseWriter, r *http.Request) {
	nummer, err := strconv.Atoi(r.PathValue("nummer"))
	if err != nil {
		schreibeFehler(w, http.StatusNotFound, "nicht_gefunden", "Diese Box gibt es nicht")
		return
	}
	b, err := s.Store.BoxPerNummer(r.Context(), nummer)
	s.packlisteAntworten(w, r, b, err)
}

// zielraeume liefert für jeden Raum der neuen Wohnung die Boxen und Items, die dorthin gehören.
func (s *server) zielraeume(w http.ResponseWriter, r *http.Request) {
	uebersicht, err := s.Store.Zielraumuebersicht(r.Context())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	type zielraumJSON struct {
		Raum  benanntJSON      `json:"raum"`
		Boxen []boxJSON        `json:"boxen"`
		Items []itemListenJSON `json:"items"`
	}
	out := []zielraumJSON{}
	for _, z := range uebersicht {
		boxen := []boxJSON{}
		for _, b := range z.Boxen {
			boxen = append(boxen, alsBoxJSON(b))
		}
		out = append(out, zielraumJSON{Raum: alsBenanntJSON(z.Raum), Boxen: boxen, Items: alsItemListeJSON(z.Items)})
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"zielraeume": out})
}

// umzugAbschliessen schließt den laufenden Umzug ab. Gibt es Items der alten Wohnung ohne Zielraum,
// verlangt er {"bestaetigt": true} und antwortet sonst mit diesen Items.
func (s *server) umzugAbschliessen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bestaetigt bool `json:"bestaetigt"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	u, ohne, err := s.Store.UmzugAbschliessen(r.Context(), req.Bestaetigt, s.Jetzt())
	if errors.Is(err, store.ErrItemsOhneZielraum) {
		schreibeJSON(w, http.StatusConflict, map[string]any{
			"code":    "items_ohne_zielraum",
			"meldung": "Diese Items der alten Wohnung haben noch keinen Zielraum und blieben, wo sie sind",
			"items":   alsItemListeJSON(ohne),
		})
		return
	}
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsUmzugJSON(u))
}
