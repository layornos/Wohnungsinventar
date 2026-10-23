// Package server stellt die JSON-API und den eingebetteten Client bereit.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"inventar/internal/config"
	"inventar/internal/foto"
	"inventar/internal/mail"
	"inventar/internal/store"
	"inventar/internal/webui"
)

// Store ist der Ausschnitt des Datenzugriffs, den die API braucht (ADR-0002),
// gegliedert nach den fachlichen Bereichen.
type Store interface {
	ZugangStore
	PersonenStore
	ItemStore
	OrteStore
	UmzugStore
}

type UmzugStore interface {
	LaufenderUmzug(ctx context.Context) (store.Umzug, error)
	UmzugAbrufen(ctx context.Context, id int64) (store.Umzug, error)
	UmzuegeAuflisten(ctx context.Context) ([]store.Umzug, error)
	UmzugAnlegen(ctx context.Context, alteWohnungID, neueWohnungID int64, jetzt time.Time) (store.Umzug, error)
	BoxAnlegen(ctx context.Context, bezeichnung string, jetzt time.Time) (store.Box, error)
	BoxAbrufen(ctx context.Context, id int64) (store.Box, error)
	BoxPerNummer(ctx context.Context, nummer int) (store.Box, error)
	BoxUmbenennen(ctx context.Context, id int64, bezeichnung string) (store.Box, error)
	BoxLoeschen(ctx context.Context, id int64) error
	UmzugsangabenSetzen(ctx context.Context, itemID int64, a store.AngabenAenderung, jetzt time.Time) (store.Item, error)
	UmzugsangabenSammeln(ctx context.Context, itemIDs []int64, a store.AngabenAenderung, jetzt time.Time) error
	Zielraumuebersicht(ctx context.Context) ([]store.ZielraumEintrag, error)
	UmzugAbschliessen(ctx context.Context, jetzt time.Time) (store.Umzug, error)
}

type ZugangStore interface {
	NutzerPerEmail(ctx context.Context, email string) (store.Nutzer, error)
	SitzungAnlegen(ctx context.Context, tokenHash string, personID int64, jetzt, laeuftAb time.Time) error
	SitzungsNutzer(ctx context.Context, tokenHash string, jetzt time.Time) (store.Nutzer, error)
	SitzungLoeschen(ctx context.Context, tokenHash string) error
}

type PersonenStore interface {
	PersonenAuflisten(ctx context.Context) ([]store.Person, error)
	PersonAnlegen(ctx context.Context, name string, jetzt time.Time) (store.Person, error)
	PersonUmbenennen(ctx context.Context, id int64, name string) (store.Person, error)
	PersonLoeschen(ctx context.Context, id int64) error
}

type ItemStore interface {
	ItemAnlegen(ctx context.Context, neu store.NeuesItem, jetzt time.Time) (store.Item, error)
	ItemAbrufen(ctx context.Context, id int64) (store.Item, error)
	ItemsAuflisten(ctx context.Context, f store.ItemFilter) ([]store.ItemListenEintrag, bool, error)
	ItemBearbeiten(ctx context.Context, id int64, name, beschreibung string, verbleib store.Verbleib, jetzt time.Time) (store.Item, error)
	ItemLoeschen(ctx context.Context, id int64, jetzt time.Time) ([]string, error)
	BesitzerHinzufuegen(ctx context.Context, itemID, personID int64, jetzt time.Time) (store.Item, error)
	BesitzerEntfernen(ctx context.Context, itemID, personID int64, jetzt time.Time) (store.Item, error)
	FotoAbrufen(ctx context.Context, id int64) (store.Foto, error)
	FotosHinzufuegen(ctx context.Context, itemID int64, fotos []store.NeuesFoto, jetzt time.Time) (store.Item, error)
	HauptfotoSetzen(ctx context.Context, itemID, fotoID int64, jetzt time.Time) (store.Item, error)
	FotoLoeschen(ctx context.Context, fotoID int64, jetzt time.Time) (string, store.Item, error)
	PlatzierungSetzen(ctx context.Context, itemID int64, p store.Platz, jetzt time.Time) (store.Item, error)
}

// OrteStore umfasst Wohnungen, Räume und Lagerorte.
type OrteStore interface {
	WohnungenAuflisten(ctx context.Context) ([]store.Wohnung, error)
	WohnungAnlegen(ctx context.Context, name string, jetzt time.Time) (store.Benannt, error)
	WohnungUmbenennen(ctx context.Context, id int64, name string) (store.Benannt, error)
	WohnungLoeschen(ctx context.Context, id int64) error
	RaumAnlegen(ctx context.Context, wohnungID int64, name string, jetzt time.Time) (store.Benannt, error)
	RaumAbrufen(ctx context.Context, id int64) (store.Raum, error)
	RaumUmbenennen(ctx context.Context, id int64, name string) (store.Benannt, error)
	RaumLoeschen(ctx context.Context, id int64) error
	LagerortAnlegen(ctx context.Context, neu store.NeuerLagerort, jetzt time.Time) (store.LagerortDetail, error)
	LagerortAbrufen(ctx context.Context, id int64) (store.LagerortDetail, error)
	LagerortUmbenennen(ctx context.Context, id int64, name string) (store.Benannt, error)
	LagerortVerschieben(ctx context.Context, id, raumID, elternID int64) (store.LagerortDetail, error)
	LagerortLoeschen(ctx context.Context, id int64, jetzt time.Time) error
	MoebelSetzen(ctx context.Context, lagerortID, itemID int64) (store.LagerortDetail, error)
	PlaetzeAuflisten(ctx context.Context) ([]store.PlatzEintrag, error)
}

type Deps struct {
	Config config.Config
	Store  Store
	Mailer mail.Mailer
	Logger *slog.Logger
	Jetzt  func() time.Time
	// Client sind die Dateien des gebauten Clients; ohne Angabe die eingebetteten.
	Client fs.FS
}

type server struct {
	Deps
	fotos foto.Speicher
}

func New(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Client == nil {
		d.Client = webui.Dist()
	}
	if d.Jetzt == nil {
		d.Jetzt = time.Now
	}
	maxFoto := d.Config.MaxFotoGroesse
	if maxFoto == 0 {
		maxFoto = config.StandardMaxFotoGroesse
	}
	s := &server{Deps: d, fotos: foto.Speicher{
		Verzeichnis: filepath.Join(d.Config.Datenverzeichnis, store.FotoVerzeichnis),
		MaxGroesse:  maxFoto,
	}}

	// Alles außer der Anmeldung braucht eine Sitzung, auch unbekannte Endpunkte.
	geschuetzt := http.NewServeMux()
	geschuetzt.HandleFunc("POST /api/abmeldung", s.abmelden)
	geschuetzt.HandleFunc("GET /api/ich", s.ich)
	geschuetzt.HandleFunc("GET /api/personen", s.personenAuflisten)
	geschuetzt.HandleFunc("POST /api/personen", s.personAnlegen)
	geschuetzt.HandleFunc("PUT /api/personen/{id}", s.personUmbenennen)
	geschuetzt.HandleFunc("DELETE /api/personen/{id}", s.personLoeschen)
	geschuetzt.HandleFunc("GET /api/items", s.itemsAuflisten)
	geschuetzt.HandleFunc("POST /api/items", s.itemAnlegen)
	geschuetzt.HandleFunc("GET /api/items/{id}", s.itemAbrufen)
	geschuetzt.HandleFunc("PUT /api/items/{id}", s.itemBearbeiten)
	geschuetzt.HandleFunc("DELETE /api/items/{id}", s.itemLoeschen)
	geschuetzt.HandleFunc("PUT /api/items/{id}/besitzer/{personId}", s.besitzerAendern(s.Store.BesitzerHinzufuegen))
	geschuetzt.HandleFunc("DELETE /api/items/{id}/besitzer/{personId}", s.besitzerAendern(s.Store.BesitzerEntfernen))
	geschuetzt.HandleFunc("POST /api/items/{id}/fotos", s.fotosHinzufuegen)
	geschuetzt.HandleFunc("PUT /api/items/{id}/hauptfoto", s.hauptfotoSetzen)
	geschuetzt.HandleFunc("GET /api/fotos/{id}", s.fotoAusliefern(false))
	geschuetzt.HandleFunc("DELETE /api/fotos/{id}", s.fotoLoeschen)
	geschuetzt.HandleFunc("GET /api/fotos/{id}/vorschau", s.fotoAusliefern(true))
	geschuetzt.HandleFunc("GET /api/wohnungen", s.wohnungenAuflisten)
	geschuetzt.HandleFunc("POST /api/wohnungen", s.wohnungAnlegen)
	geschuetzt.HandleFunc("PUT /api/wohnungen/{id}", s.umbenennen(s.Store.WohnungUmbenennen))
	geschuetzt.HandleFunc("DELETE /api/wohnungen/{id}", s.loeschen(s.Store.WohnungLoeschen))
	geschuetzt.HandleFunc("POST /api/wohnungen/{id}/raeume", s.raumAnlegen)
	geschuetzt.HandleFunc("GET /api/raeume/{id}", s.raumAbrufen)
	geschuetzt.HandleFunc("PUT /api/raeume/{id}", s.umbenennen(s.Store.RaumUmbenennen))
	geschuetzt.HandleFunc("DELETE /api/raeume/{id}", s.loeschen(s.Store.RaumLoeschen))
	geschuetzt.HandleFunc("GET /api/plaetze", s.plaetzeAuflisten)
	geschuetzt.HandleFunc("POST /api/lagerorte", s.lagerortAnlegen)
	geschuetzt.HandleFunc("GET /api/lagerorte/{id}", s.lagerortAbrufen)
	geschuetzt.HandleFunc("PUT /api/lagerorte/{id}", s.umbenennen(s.Store.LagerortUmbenennen))
	geschuetzt.HandleFunc("DELETE /api/lagerorte/{id}", s.lagerortLoeschen)
	geschuetzt.HandleFunc("PUT /api/lagerorte/{id}/eltern", s.lagerortVerschieben)
	geschuetzt.HandleFunc("GET /api/lagerorte/{id}/wegfall", s.wegfallVorschau(
		func(ctx context.Context, id int64) error { _, err := s.Store.LagerortAbrufen(ctx, id); return err },
		func(f *store.ItemFilter, id int64) { f.LagerortID = id }))
	geschuetzt.HandleFunc("GET /api/items/{id}/wegfall", s.wegfallVorschau(
		func(ctx context.Context, id int64) error { _, err := s.Store.ItemAbrufen(ctx, id); return err },
		func(f *store.ItemFilter, id int64) { f.MoebelID = id }))
	geschuetzt.HandleFunc("PUT /api/lagerorte/{id}/moebel", s.moebelSetzen)
	geschuetzt.HandleFunc("PUT /api/items/{id}/platzierung", s.platzierungSetzen)
	geschuetzt.HandleFunc("GET /api/umzug", s.laufenderUmzug)
	geschuetzt.HandleFunc("POST /api/umzug", s.umzugAnlegen)
	geschuetzt.HandleFunc("POST /api/umzug/boxen", s.boxAnlegen)
	geschuetzt.HandleFunc("GET /api/umzug/boxen/{nummer}", s.boxPerNummer)
	geschuetzt.HandleFunc("GET /api/umzug/zielraeume", s.zielraeume)
	geschuetzt.HandleFunc("POST /api/umzug/abschluss", s.umzugAbschliessen)
	geschuetzt.HandleFunc("GET /api/boxen/{id}", s.boxAbrufen)
	geschuetzt.HandleFunc("PUT /api/boxen/{id}", s.boxUmbenennen)
	geschuetzt.HandleFunc("DELETE /api/boxen/{id}", s.loeschen(s.Store.BoxLoeschen))
	geschuetzt.HandleFunc("PATCH /api/items/{id}/umzug", s.umzugsangabenSetzen)
	geschuetzt.HandleFunc("POST /api/umzug/sammeln", s.umzugsangabenSammeln)
	geschuetzt.HandleFunc("GET /api/umzuege", s.umzuegeAuflisten)
	geschuetzt.HandleFunc("GET /api/umzuege/{id}", s.umzugAbrufen)
	geschuetzt.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		schreibeFehler(w, http.StatusNotFound, "nicht_gefunden", "Unbekannter Endpunkt")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/anmeldung", s.anmelden)
	mux.Handle("/api/", s.nurMitSitzung(geschuetzt))
	mux.Handle("/", webui.Handler(d.Client))
	return mux
}

type apiFehler struct {
	Code    string `json:"code"`
	Meldung string `json:"meldung"`
}

func schreibeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func schreibeFehler(w http.ResponseWriter, status int, code, meldung string) {
	schreibeJSON(w, status, apiFehler{Code: code, Meldung: meldung})
}

func (s *server) internerFehler(w http.ResponseWriter, r *http.Request, err error) {
	s.Logger.Error("interner Fehler", "pfad", r.URL.Path, "fehler", err)
	schreibeFehler(w, http.StatusInternalServerError, "intern", "Interner Fehler")
}

// leseJSON liest einen kleinen JSON-Body und antwortet bei Fehlern selbst.
func leseJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		var zuGross *http.MaxBytesError
		if errors.As(err, &zuGross) {
			schreibeFehler(w, http.StatusRequestEntityTooLarge, "zu_gross", "Anfrage zu groß")
		} else {
			schreibeFehler(w, http.StatusBadRequest, "ungueltige_anfrage", "Ungültiges JSON")
		}
		return false
	}
	return true
}

// pfadID liest eine numerische ID aus dem Pfad; unbrauchbare IDs gelten als nicht gefunden.
func pfadID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		schreibeFehler(w, http.StatusNotFound, "nicht_gefunden", "Nicht gefunden")
		return 0, false
	}
	return id, true
}

// storeFehler übersetzt fachliche Fehler des Stores in API-Fehler.
func (s *server) storeFehler(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNichtGefunden):
		schreibeFehler(w, http.StatusNotFound, "nicht_gefunden", "Nicht gefunden")
	case errors.Is(err, store.ErrHatNutzerzugang):
		schreibeFehler(w, http.StatusConflict, "hat_nutzerzugang", "Personen mit Nutzerzugang können nicht gelöscht werden")
	case errors.Is(err, store.ErrLetzterBesitzer):
		schreibeFehler(w, http.StatusConflict, "letzter_besitzer", meldungBesitzerFehlt)
	case errors.Is(err, store.ErrLetztesFoto):
		schreibeFehler(w, http.StatusConflict, "letztes_foto", meldungFotoFehlt)
	case errors.Is(err, store.ErrWohnungNichtLeer):
		schreibeFehler(w, http.StatusConflict, "wohnung_nicht_leer", "Die Wohnung hat noch Räume")
	case errors.Is(err, store.ErrRaumNichtLeer):
		schreibeFehler(w, http.StatusConflict, "raum_nicht_leer", "Im Raum sind noch Lagerorte oder Items")
	case errors.Is(err, store.ErrZyklus):
		schreibeFehler(w, http.StatusConflict, "zyklus", "Ein Lagerort kann nicht in sich selbst oder einem seiner Unterlagerorte liegen")
	case errors.Is(err, store.ErrMoebelInSichSelbst):
		schreibeFehler(w, http.StatusConflict, "moebel_in_sich_selbst", "Ein Möbelstück kann nicht in einem eigenen Lagerort liegen")
	case errors.Is(err, store.ErrUnbekannterPlatz):
		schreibeFehler(w, http.StatusBadRequest, "unbekannter_platz", "Diesen Raum oder Lagerort gibt es nicht")
	case errors.Is(err, store.ErrKeinUmzug):
		schreibeFehler(w, http.StatusConflict, "kein_umzug", "Es läuft kein Umzug")
	case errors.Is(err, store.ErrUmzugLaeuft):
		schreibeFehler(w, http.StatusConflict, "umzug_laeuft", "Es läuft schon ein Umzug")
	case errors.Is(err, store.ErrUmzugAbgeschlossen):
		schreibeFehler(w, http.StatusConflict, "umzug_abgeschlossen", "Der Umzug ist abgeschlossen und lässt sich nicht mehr ändern")
	case errors.Is(err, store.ErrGleicheWohnung):
		schreibeFehler(w, http.StatusBadRequest, "gleiche_wohnung", "Alte und neue Wohnung müssen verschieden sein")
	case errors.Is(err, store.ErrUnbekannteWohnung):
		schreibeFehler(w, http.StatusBadRequest, "unbekannte_wohnung", "Diese Wohnung gibt es nicht")
	case errors.Is(err, store.ErrWohnungImUmzug):
		schreibeFehler(w, http.StatusConflict, "wohnung_im_umzug", "Die Wohnung gehört zu einem Umzug")
	case errors.Is(err, store.ErrRaumImUmzug):
		schreibeFehler(w, http.StatusConflict, "raum_im_umzug", "Der Raum ist Herkunfts- oder Zielraum in einem Umzug")
	case errors.Is(err, store.ErrBoxNichtLeer):
		schreibeFehler(w, http.StatusConflict, "box_nicht_leer", "Nur leere Boxen lassen sich löschen")
	case errors.Is(err, store.ErrUnbekannteBox):
		schreibeFehler(w, http.StatusBadRequest, "unbekannte_box", "Diese Box gibt es im laufenden Umzug nicht")
	case errors.Is(err, store.ErrZielraumFalsch):
		schreibeFehler(w, http.StatusBadRequest, "zielraum_nicht_in_neuer_wohnung", "Der Zielraum muss in der neuen Wohnung liegen")
	case errors.Is(err, store.ErrUnbekanntesItem):
		schreibeFehler(w, http.StatusBadRequest, "unbekanntes_item", "Dieses Item gibt es nicht")
	case errors.Is(err, store.ErrUnbekanntePerson):
		schreibeFehler(w, http.StatusBadRequest, "unbekannte_person", meldungUnbekanntePerson)
	default:
		s.internerFehler(w, r, err)
	}
}
