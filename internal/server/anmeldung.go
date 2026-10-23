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

func toNutzerJSON(n store.Nutzer) nutzerJSON {
	return nutzerJSON{PersonID: n.PersonID, Name: n.Name, Email: n.Email}
}

func (s *server) anmelden(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Passwort string `json:"passwort"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	n, err := s.Store.NutzerByEmail(r.Context(), req.Email)
	hash := n.PasswortHash
	if errors.Is(err, store.ErrNichtGefunden) {
		hash = auth.DummyHash
	} else if err != nil {
		s.internalError(w, r, err)
		return
	}
	ok, err := auth.PruefePasswort(hash, req.Passwort)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !ok || n.PersonID == 0 {
		writeError(w, http.StatusUnauthorized, "anmeldung_fehlgeschlagen", "E-Mail oder Passwort falsch")
		return
	}
	if err := s.starteSitzung(w, r, n.PersonID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toNutzerJSON(n))
}

// starteSitzung legt eine Sitzung an und setzt das Cookie.
func (s *server) starteSitzung(w http.ResponseWriter, r *http.Request, personID int64) error {
	token, hash, err := auth.NeuesToken()
	if err != nil {
		return err
	}
	now := s.Now()
	if err := s.Store.CreateSitzung(r.Context(), hash, personID, now, now.Add(sitzungsDauer)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sitzungsCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sitzungsDauer.Seconds()),
		HttpOnly: true,
		Secure:   s.Config.TLSEnabled(),
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *server) abmelden(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sitzungsCookie); err == nil {
		if err := s.Store.DeleteSitzung(r.Context(), auth.TokenHash(c.Value)); err != nil {
			s.internalError(w, r, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sitzungsCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.Config.TLSEnabled(),
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) ich(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toNutzerJSON(nutzerAus(r.Context())))
}

type nutzerKey struct{}

func nutzerAus(ctx context.Context) store.Nutzer {
	n, _ := ctx.Value(nutzerKey{}).(store.Nutzer)
	return n
}

// requireNutzer lässt nur Anfragen mit gültiger Sitzung durch.
func (s *server) requireNutzer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sitzungsCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "nicht_angemeldet", "Nicht angemeldet")
			return
		}
		n, err := s.Store.SitzungNutzer(r.Context(), auth.TokenHash(c.Value), s.Now())
		if errors.Is(err, store.ErrNichtGefunden) {
			writeError(w, http.StatusUnauthorized, "nicht_angemeldet", "Nicht angemeldet")
			return
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nutzerKey{}, n)))
	})
}
