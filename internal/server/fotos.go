package server

import (
	"errors"
	"net/http"
	"os"

	"inventar/internal/store"
)

// fotoAusliefern liefert das Original oder die Vorschau eines Fotos. Die Adresse enthält den
// zufälligen Dateinamen und passt nur zu genau einer Datei; der Browser darf sie also dauerhaft
// zwischenspeichern, aber nur für sich (private).
func (s *server) fotoAusliefern(vorschau bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pfadID(w, r, "id")
		if !ok {
			return
		}
		f, err := s.Store.FotoAbrufen(r.Context(), id)
		if err == nil && f.Datei != r.PathValue("datei") {
			err = store.ErrNichtGefunden
		}
		if err != nil {
			s.storeFehler(w, r, err)
			return
		}
		pfad, typ := s.fotos.Pfad(f.Datei), f.Typ
		if vorschau {
			pfad, typ = s.fotos.VorschauPfad(f.Datei), "image/jpeg"
		}
		datei, err := os.Open(pfad)
		if errors.Is(err, os.ErrNotExist) {
			schreibeFehler(w, http.StatusNotFound, "nicht_gefunden", "Foto-Datei fehlt")
			return
		}
		if err != nil {
			s.internerFehler(w, r, err)
			return
		}
		defer datei.Close()
		info, err := datei.Stat()
		if err != nil {
			s.internerFehler(w, r, err)
			return
		}
		w.Header().Set("Content-Type", typ)
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, "", info.ModTime(), datei)
	}
}

func (s *server) fotosHinzufuegen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	f, ok := s.leseFormular(w, r)
	if !ok {
		return
	}
	if len(f.fotos) == 0 {
		fotoFehlt(w)
		return
	}
	item, err := s.Store.FotosHinzufuegen(r.Context(), id, f.neueFotos(), s.Jetzt())
	if err != nil {
		s.fotosVerwerfen(f.fotos)
	}
	s.itemAntworten(w, r, item, err)
}

func fotoFehlt(w http.ResponseWriter) {
	schreibeFehler(w, http.StatusBadRequest, "foto_fehlt", meldungFotoFehlt)
}

func (s *server) hauptfotoSetzen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		FotoID int64 `json:"fotoId"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	item, err := s.Store.HauptfotoSetzen(r.Context(), id, req.FotoID, s.Jetzt())
	s.itemAntworten(w, r, item, err)
}

func (s *server) fotoLoeschen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	datei, item, err := s.Store.FotoLoeschen(r.Context(), id, s.Jetzt())
	if err == nil {
		s.dateiLoeschen(datei)
	}
	s.itemAntworten(w, r, item, err)
}
