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

// Store ist der Ausschnitt des Datenzugriffs, den die API braucht (ADR-0002).
type Store interface {
	NutzerPerEmail(ctx context.Context, email string) (store.Nutzer, error)
	SitzungAnlegen(ctx context.Context, tokenHash string, personID int64, jetzt, laeuftAb time.Time) error
	SitzungsNutzer(ctx context.Context, tokenHash string, jetzt time.Time) (store.Nutzer, error)
	SitzungLoeschen(ctx context.Context, tokenHash string) error

	PersonenAuflisten(ctx context.Context) ([]store.Person, error)
	PersonAnlegen(ctx context.Context, name string, jetzt time.Time) (store.Person, error)
	PersonUmbenennen(ctx context.Context, id int64, name string) (store.Person, error)
	PersonLoeschen(ctx context.Context, id int64) error

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

	WohnungenAuflisten(ctx context.Context) ([]store.Wohnung, error)
	WohnungAnlegen(ctx context.Context, name string, jetzt time.Time) (store.Ort, error)
	WohnungUmbenennen(ctx context.Context, id int64, name string) (store.Ort, error)
	WohnungLoeschen(ctx context.Context, id int64) error
	RaumAnlegen(ctx context.Context, wohnungID int64, name string, jetzt time.Time) (store.Ort, error)
	RaumAbrufen(ctx context.Context, id int64) (store.Raum, error)
	RaumUmbenennen(ctx context.Context, id int64, name string) (store.Ort, error)
	RaumLoeschen(ctx context.Context, id int64) error
	LagerortAnlegen(ctx context.Context, neu store.NeuerLagerort, jetzt time.Time) (store.LagerortDetail, error)
	LagerortAbrufen(ctx context.Context, id int64) (store.LagerortDetail, error)
	LagerortUmbenennen(ctx context.Context, id int64, name string) (store.Ort, error)
	LagerortVerschieben(ctx context.Context, id, raumID, elternID int64) (store.LagerortDetail, error)
	LagerortLoeschen(ctx context.Context, id int64, jetzt time.Time) error
	MoebelSetzen(ctx context.Context, lagerortID, itemID int64) (store.LagerortDetail, error)
	PlaetzeAuflisten(ctx context.Context) ([]store.PlatzEintrag, error)
	PlatzierungSetzen(ctx context.Context, itemID int64, p store.Platz, jetzt time.Time) (store.Item, error)
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
	geschuetzt.HandleFunc("GET /api/lagerorte/{id}/wegfall", s.wegfallVorschau(func(f *store.ItemFilter, id int64) { f.LagerortID = id }))
	geschuetzt.HandleFunc("GET /api/items/{id}/wegfall", s.wegfallVorschau(func(f *store.ItemFilter, id int64) { f.MoebelID = id }))
	geschuetzt.HandleFunc("PUT /api/lagerorte/{id}/moebel", s.moebelSetzen)
	geschuetzt.HandleFunc("PUT /api/items/{id}/platzierung", s.platzierungSetzen)
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
	case errors.Is(err, store.ErrUnbekanntePerson):
		schreibeFehler(w, http.StatusBadRequest, "unbekannte_person", meldungUnbekanntePerson)
	default:
		s.internerFehler(w, r, err)
	}
}
