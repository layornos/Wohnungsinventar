package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"inventar/internal/auth"
	"inventar/internal/store"
)

const (
	sitzungsCookie = "sitzung"
	sitzungsDauer  = 90 * 24 * time.Hour
)

type nutzerJSON struct {
	PersonID int64  `json:"personId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func alsNutzerJSON(n store.Nutzer) nutzerJSON {
	return nutzerJSON{PersonID: n.PersonID, Name: n.Name, Email: n.Email}
}

// ichJSON ist der angemeldete Nutzer samt seinen Einstellungen, die andere Nutzer nicht sehen.
type ichJSON struct {
	nutzerJSON
	// Design ist null, solange der Nutzer keins gewählt hat.
	Design *store.Design `json:"design"`
}

func alsIchJSON(n store.Nutzer) ichJSON {
	out := ichJSON{nutzerJSON: alsNutzerJSON(n)}
	if n.Design != store.KeinDesign {
		out.Design = &n.Design
	}
	return out
}

func (s *server) anmelden(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Passwort string `json:"passwort"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	n, err := s.Store.NutzerPerEmail(r.Context(), req.Email)
	gefunden := err == nil
	if err != nil && !errors.Is(err, store.ErrNichtGefunden) {
		s.internerFehler(w, r, err)
		return
	}
	hash := n.PasswortHash
	if !gefunden {
		hash = auth.DummyHash
	}
	passt, err := auth.PasswortPruefen(hash, req.Passwort)
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	if !gefunden || !passt {
		schreibeFehler(w, http.StatusUnauthorized, "anmeldung_fehlgeschlagen", "E-Mail oder Passwort falsch")
		return
	}
	if err := s.sitzungStarten(w, r, n.PersonID); err != nil {
		s.internerFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsIchJSON(n))
}

// sitzungStarten legt eine Sitzung an und setzt das Cookie.
func (s *server) sitzungStarten(w http.ResponseWriter, r *http.Request, personID int64) error {
	token, hash, err := auth.NeuesToken()
	if err != nil {
		return err
	}
	jetzt := s.Jetzt()
	if err := s.Store.SitzungAnlegen(r.Context(), hash, personID, jetzt, jetzt.Add(sitzungsDauer)); err != nil {
		return err
	}
	s.sitzungsCookieSetzen(w, token, int(sitzungsDauer.Seconds()))
	return nil
}

func (s *server) abmelden(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sitzungsCookie); err == nil {
		if err := s.Store.SitzungLoeschen(r.Context(), auth.TokenHash(c.Value)); err != nil {
			s.internerFehler(w, r, err)
			return
		}
	}
	s.sitzungsCookieSetzen(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// sitzungsCookieSetzen setzt das Cookie; maxAge -1 löscht es im Browser.
func (s *server) sitzungsCookieSetzen(w http.ResponseWriter, wert string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sitzungsCookie,
		Value:    wert,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.Config.MitTLS(),
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *server) ich(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, alsIchJSON(nutzerAus(r.Context())))
}

func (s *server) designSetzen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Design store.Design `json:"design"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	if !req.Design.Waehlbar() {
		schreibeFehler(w, http.StatusBadRequest, "unbekanntes_design", "Unbekanntes Design")
		return
	}
	n, err := s.Store.DesignSetzen(r.Context(), nutzerAus(r.Context()).PersonID, req.Design)
	if err != nil {
		s.storeFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, alsIchJSON(n))
}

type nutzerKey struct{}

func nutzerAus(ctx context.Context) store.Nutzer {
	n, _ := ctx.Value(nutzerKey{}).(store.Nutzer)
	return n
}

// nurMitSitzung lässt nur Anfragen mit gültiger Sitzung durch.
func (s *server) nurMitSitzung(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nichtAngemeldet := func() {
			schreibeFehler(w, http.StatusUnauthorized, "nicht_angemeldet", "Nicht angemeldet")
		}
		c, err := r.Cookie(sitzungsCookie)
		if err != nil {
			nichtAngemeldet()
			return
		}
		n, err := s.Store.SitzungsNutzer(r.Context(), auth.TokenHash(c.Value), s.Jetzt())
		if errors.Is(err, store.ErrNichtGefunden) {
			nichtAngemeldet()
			return
		}
		if err != nil {
			s.internerFehler(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nutzerKey{}, n)))
	})
}
