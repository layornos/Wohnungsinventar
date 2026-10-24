package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inventar/internal/foto"
	"inventar/internal/store"
)

const (
	maxFotosProAnfrage = 10
	maxFormularTeile   = 50
	maxFeldLaenge      = 10_000
	standardProSeite   = 50
	maxProSeite        = 100
	// maxVollstaendig begrenzt Listen, die vollständig sein sollen, z. B. Wegfall-Vorschau und Packliste.
	maxVollstaendig = 10_000
)

type itemJSON struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Beschreibung string          `json:"beschreibung"`
	Verbleib     *store.Verbleib `json:"verbleib"`
	ErstelltAm   time.Time       `json:"erstelltAm"`
	GeaendertAm  time.Time       `json:"geaendertAm"`
	Besitzer     []personJSON    `json:"besitzer"`
	Fotos        []fotoJSON      `json:"fotos"`
	// Platzierung ist null, solange das Item nicht platziert ist.
	Platzierung     *pfadJSON     `json:"platzierung"`
	MoebelLagerorte []benanntJSON `json:"moebelLagerorte"`
	// Umzug sind die Angaben im laufenden Umzug, sonst null.
	Umzug *umzugsangabenJSON `json:"umzug"`
}

type fotoJSON struct {
	ID          int64  `json:"id"`
	Hauptfoto   bool   `json:"hauptfoto"`
	URL         string `json:"url"`
	VorschauURL string `json:"vorschauUrl"`
}

// verbleibJSON stellt Offen als null dar.
func verbleibJSON(v store.Verbleib) *store.Verbleib {
	if v == store.Offen {
		return nil
	}
	return &v
}

// fotoURL enthält neben der ID den zufälligen Dateinamen: Fotos bleiben dauerhaft im
// Browser-Cache, die Adresse muss sich also ändern, sobald sich die Datei ändert.
func fotoURL(id int64, datei string) string {
	return fmt.Sprintf("/api/fotos/%d/%s", id, datei)
}

func vorschauURL(fotoID int64, datei string) string { return fotoURL(fotoID, datei) + "/vorschau" }

func alsItemJSON(item store.Item) itemJSON {
	out := itemJSON{
		ID:              item.ID,
		Name:            item.Name,
		Beschreibung:    item.Beschreibung,
		Verbleib:        verbleibJSON(item.Verbleib),
		ErstelltAm:      item.ErstelltAm,
		GeaendertAm:     item.GeaendertAm,
		Besitzer:        alsPersonenJSON(item.Besitzer),
		Fotos:           []fotoJSON{},
		MoebelLagerorte: alsBenannteJSON(item.MoebelLagerorte),
		Umzug:           alsUmzugsangabenJSON(item.Umzug),
	}
	if item.Platzierung != nil {
		p := alsPfadJSON(*item.Platzierung)
		out.Platzierung = &p
	}
	for _, f := range item.Fotos {
		out.Fotos = append(out.Fotos, fotoJSON{ID: f.ID, Hauptfoto: f.Hauptfoto, URL: fotoURL(f.ID, f.Datei), VorschauURL: vorschauURL(f.ID, f.Datei)})
	}
	return out
}

// formular ist eine gelesene multipart-Anfrage; die Fotos sind bereits gespeichert.
type formular struct {
	felder map[string][]string
	fotos  []foto.Datei
}

func (f formular) feld(name string) string {
	if v := f.felder[name]; len(v) > 0 {
		return strings.TrimSpace(v[0])
	}
	return ""
}

func (f formular) neueFotos() []store.NeuesFoto {
	out := make([]store.NeuesFoto, 0, len(f.fotos))
	for _, d := range f.fotos {
		out = append(out, store.NeuesFoto{Datei: d.Name, Typ: d.Typ})
	}
	return out
}

// leseFormular liest eine multipart-Anfrage und speichert die Fotos (Feld "foto") dabei gleich.
// Bei Fehlern antwortet es selbst und verwirft bereits gespeicherte Fotos.
func (s *server) leseFormular(w http.ResponseWriter, r *http.Request) (formular, bool) {
	f := formular{felder: map[string][]string{}}
	abbrechen := func(status int, code, meldung string) (formular, bool) {
		s.fotosVerwerfen(f.fotos)
		schreibeFehler(w, status, code, meldung)
		return formular{}, false
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return abbrechen(http.StatusBadRequest, "ungueltige_anfrage", "Formulardaten erwartet")
	}
	for teile := 0; ; teile++ {
		teil, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return f, true
		}
		if err != nil || teile >= maxFormularTeile {
			return abbrechen(http.StatusBadRequest, "ungueltige_anfrage", "Ungültige Formulardaten")
		}
		if teil.FormName() == "foto" {
			if len(f.fotos) >= maxFotosProAnfrage {
				return abbrechen(http.StatusBadRequest, "zu_viele_fotos", fmt.Sprintf("Höchstens %d Fotos auf einmal", maxFotosProAnfrage))
			}
			d, err := s.fotos.Speichern(teil)
			switch {
			case errors.Is(err, foto.ErrFalscherTyp):
				return abbrechen(http.StatusBadRequest, "falscher_dateityp", "Nur JPEG-, PNG- und WebP-Fotos")
			case errors.Is(err, foto.ErrZuGross):
				return abbrechen(http.StatusRequestEntityTooLarge, "zu_gross",
					fmt.Sprintf("Fotos dürfen höchstens %d MB groß sein", s.fotos.MaxGroesse>>20))
			case err != nil:
				s.fotosVerwerfen(f.fotos)
				s.internerFehler(w, r, err)
				return formular{}, false
			}
			f.fotos = append(f.fotos, d)
			continue
		}
		wert, err := io.ReadAll(io.LimitReader(teil, maxFeldLaenge+1))
		if err != nil || len(wert) > maxFeldLaenge {
			return abbrechen(http.StatusBadRequest, "ungueltige_anfrage", "Formularfeld zu lang")
		}
		f.felder[teil.FormName()] = append(f.felder[teil.FormName()], string(wert))
	}
}

// fotosVerwerfen entfernt gespeicherte Fotos, die keinem Item zugeordnet wurden.
func (s *server) fotosVerwerfen(dateien []foto.Datei) {
	for _, d := range dateien {
		s.dateiLoeschen(d.Name)
	}
}

func (s *server) dateiLoeschen(name string) {
	if err := s.fotos.Loeschen(name); err != nil {
		s.Logger.Error("Foto-Datei löschen", "datei", name, "fehler", err)
	}
}

func verbleibLesen(w http.ResponseWriter, wert string) (store.Verbleib, bool) {
	v := store.Verbleib(wert)
	if !v.Gueltig() {
		schreibeFehler(w, http.StatusBadRequest, "ungueltiger_verbleib", "Unbekannter Verbleib")
		return "", false
	}
	return v, true
}

const (
	meldungBesitzerFehlt    = "Jedes Item braucht mindestens einen Besitzer"
	meldungFotoFehlt        = "Jedes Item braucht mindestens ein Foto"
	meldungUnbekanntePerson = "Diese Person gibt es nicht"
	meldungUnbekannteBox    = "Diese Box gibt es im laufenden Umzug nicht"
)

func nameFehlt(w http.ResponseWriter) {
	schreibeFehler(w, http.StatusBadRequest, "name_fehlt", "Bitte einen Namen angeben")
}

// neuesItemLesen prüft die Formularangaben eines neuen Items und antwortet bei Fehlern selbst.
func neuesItemLesen(w http.ResponseWriter, f formular) (store.NeuesItem, bool) {
	neu := store.NeuesItem{Name: f.feld("name"), Beschreibung: f.feld("beschreibung"), Fotos: f.neueFotos()}
	if neu.Name == "" {
		nameFehlt(w)
		return neu, false
	}
	v, ok := verbleibLesen(w, f.feld("verbleib"))
	if !ok {
		return neu, false
	}
	neu.Verbleib = v
	for _, wert := range f.felder["besitzer"] {
		id, err := strconv.ParseInt(strings.TrimSpace(wert), 10, 64)
		if err != nil {
			schreibeFehler(w, http.StatusBadRequest, "unbekannte_person", meldungUnbekanntePerson)
			return neu, false
		}
		neu.BesitzerIDs = append(neu.BesitzerIDs, id)
	}
	if len(neu.BesitzerIDs) == 0 {
		schreibeFehler(w, http.StatusBadRequest, "besitzer_fehlt", meldungBesitzerFehlt)
		return neu, false
	}
	if len(neu.Fotos) == 0 {
		fotoFehlt(w)
		return neu, false
	}
	if neu.Platz, ok = platzAusFormular(w, f); !ok {
		return neu, false
	}
	if box := f.feld("box"); box != "" {
		id, err := strconv.ParseInt(box, 10, 64)
		if err != nil {
			schreibeFehler(w, http.StatusBadRequest, "unbekannte_box", meldungUnbekannteBox)
			return neu, false
		}
		neu.BoxID = id
	}
	return neu, true
}

func (s *server) itemAnlegen(w http.ResponseWriter, r *http.Request) {
	f, ok := s.leseFormular(w, r)
	if !ok {
		return
	}
	neu, ok := neuesItemLesen(w, f)
	if !ok {
		s.fotosVerwerfen(f.fotos)
		return
	}
	item, err := s.Store.ItemAnlegen(r.Context(), neu, s.Jetzt())
	if err != nil {
		s.fotosVerwerfen(f.fotos)
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, alsItemJSON(item))
}

func (s *server) itemAbrufen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	item, err := s.Store.ItemAbrufen(r.Context(), id)
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsItemJSON(item))
}

type itemListenJSON struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	Verbleib    *store.Verbleib    `json:"verbleib"`
	VorschauURL string             `json:"vorschauUrl"`
	Platz       string             `json:"platz"`
	Umzug       *umzugsangabenJSON `json:"umzug"`
}

func alsItemListeJSON(eintraege []store.ItemListenEintrag) []itemListenJSON {
	items := make([]itemListenJSON, 0, len(eintraege))
	for _, e := range eintraege {
		items = append(items, itemListenJSON{
			ID: e.ID, Name: e.Name, Verbleib: verbleibJSON(e.Verbleib), VorschauURL: vorschauURL(e.HauptfotoID, e.HauptfotoDatei),
			Platz: e.Platz, Umzug: alsUmzugsangabenJSON(e.Umzug),
		})
	}
	return items
}

// itemsAuflisten filtert mit q (Text im Namen), besitzer (Person-ID), verbleib (inkl. "offen"),
// raum, lagerort (beide inkl. verschachtelter Lagerorte), ohneLagerort sowie box, herkunftsraum und
// ohneZielraum im laufenden Umzug und liefert seitenweise
// mit seite (ab 1) und proSeite.
func (s *server) itemsAuflisten(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.ItemFilter{Text: strings.TrimSpace(q.Get("q")), Anzahl: standardProSeite}
	ungueltig := func(meldung string) {
		schreibeFehler(w, http.StatusBadRequest, "ungueltige_anfrage", meldung)
	}
	for _, feld := range []struct {
		name string
		ziel *int64
	}{
		{"besitzer", &filter.BesitzerID}, {"raum", &filter.RaumID}, {"lagerort", &filter.LagerortID},
		{"box", &filter.BoxID}, {"herkunftsraum", &filter.HerkunftsraumID},
	} {
		if wert := q.Get(feld.name); wert != "" {
			id, err := strconv.ParseInt(wert, 10, 64)
			if err != nil {
				ungueltig("Ungültige ID für " + feld.name)
				return
			}
			*feld.ziel = id
		}
	}
	filter.OhneLagerort = q.Get("ohneLagerort") != ""
	filter.OhneZielraum = q.Get("ohneZielraum") != ""
	if v := q.Get("verbleib"); v != "" {
		verbleib := store.Offen
		if v != "offen" {
			var ok bool
			if verbleib, ok = verbleibLesen(w, v); !ok {
				return
			}
		}
		filter.Verbleib = &verbleib
	}
	seite := 1
	if p := q.Get("seite"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 {
			ungueltig("Ungültige Seite")
			return
		}
		seite = n
	}
	if p := q.Get("proSeite"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > maxProSeite {
			ungueltig(fmt.Sprintf("proSeite muss zwischen 1 und %d liegen", maxProSeite))
			return
		}
		filter.Anzahl = n
	}
	filter.Versatz = (seite - 1) * filter.Anzahl

	s.itemListeAntworten(w, r, filter)
}

func (s *server) itemListeAntworten(w http.ResponseWriter, r *http.Request, filter store.ItemFilter) {
	eintraege, weitere, err := s.Store.ItemsAuflisten(r.Context(), filter)
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"items": alsItemListeJSON(eintraege), "weitere": weitere})
}

// wegfallVorschau liefert alle Items, die beim Löschen ohne Lagerort dastünden. pruefen stellt sicher,
// dass es den Lagerort oder das Möbelstück gibt; filtern setzt ihn im Filter.
func (s *server) wegfallVorschau(pruefen func(ctx context.Context, id int64) error, filtern func(*store.ItemFilter, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pfadID(w, r, "id")
		if !ok {
			return
		}
		if err := pruefen(r.Context(), id); err != nil {
			s.storeFehler(w, r, err)
			return
		}
		filter := store.ItemFilter{Anzahl: maxVollstaendig}
		filtern(&filter, id)
		s.itemListeAntworten(w, r, filter)
	}
}

func (s *server) itemBearbeiten(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Name         string  `json:"name"`
		Beschreibung string  `json:"beschreibung"`
		Verbleib     *string `json:"verbleib"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		nameFehlt(w)
		return
	}
	verbleib := store.Offen
	if req.Verbleib != nil {
		var ok bool
		if verbleib, ok = verbleibLesen(w, *req.Verbleib); !ok {
			return
		}
	}
	item, err := s.Store.ItemBearbeiten(r.Context(), id, name, strings.TrimSpace(req.Beschreibung), verbleib, s.Jetzt())
	s.itemAntworten(w, r, item, err)
}

// itemAntworten antwortet nach einer Änderung mit dem Item oder dem Fehler.
func (s *server) itemAntworten(w http.ResponseWriter, r *http.Request, item store.Item, err error) {
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsItemJSON(item))
}

func (s *server) itemLoeschen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	dateien, err := s.Store.ItemLoeschen(r.Context(), id, s.Jetzt())
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	for _, d := range dateien {
		s.dateiLoeschen(d)
	}
	w.WriteHeader(http.StatusNoContent)
}

// besitzerAendern fügt die Person im Pfad als Besitzer hinzu oder entfernt sie.
func (s *server) besitzerAendern(aendern func(ctx context.Context, itemID, personID int64, jetzt time.Time) (store.Item, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemID, ok := pfadID(w, r, "id")
		if !ok {
			return
		}
		personID, ok := pfadID(w, r, "personId")
		if !ok {
			return
		}
		item, err := aendern(r.Context(), itemID, personID, s.Jetzt())
		s.itemAntworten(w, r, item, err)
	}
}
