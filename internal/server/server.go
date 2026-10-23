// Package server stellt die JSON-API und den eingebetteten Client bereit.
package server

import (
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

type Deps struct {
	Config config.Config
	Store  *store.Store
	Mailer mail.Mailer
	Logger *slog.Logger
	Now    func() time.Time
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
	if d.Now == nil {
		d.Now = time.Now
	}
	s := &server{Deps: d}

	api := http.NewServeMux()
	api.HandleFunc("POST /api/anmeldung", s.anmelden)
	api.HandleFunc("POST /api/abmeldung", s.abmelden)
	api.Handle("GET /api/ich", s.requireNutzer(http.HandlerFunc(s.ich)))
	api.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "nicht_gefunden", "Unbekannter Endpunkt")
	})

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/", webui.Handler(d.Client))
	return mux
}

type apiError struct {
	Code    string `json:"code"`
	Meldung string `json:"meldung"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, meldung string) {
	writeJSON(w, status, apiError{Code: code, Meldung: meldung})
}

func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.Logger.Error("interner Fehler", "pfad", r.URL.Path, "fehler", err)
	writeError(w, http.StatusInternalServerError, "intern", "Interner Fehler")
}

// readJSON liest einen kleinen JSON-Body und antwortet bei Fehlern selbst.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "zu_gross", "Anfrage zu groß")
		} else {
			writeError(w, http.StatusBadRequest, "ungueltige_anfrage", "Ungültiges JSON")
		}
		return false
	}
	return true
}
