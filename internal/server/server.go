// Package server stellt die JSON-API und den eingebetteten Client bereit.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"inventar/internal/config"
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
	s := &server{Deps: d}

	// Alles außer der Anmeldung braucht eine Sitzung, auch unbekannte Endpunkte.
	geschuetzt := http.NewServeMux()
	geschuetzt.HandleFunc("POST /api/abmeldung", s.abmelden)
	geschuetzt.HandleFunc("GET /api/ich", s.ich)
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
