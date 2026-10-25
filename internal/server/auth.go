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
	sessionCookie   = "session"
	sessionDuration = 90 * 24 * time.Hour
)

type userJSON struct {
	PersonID int64  `json:"personId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func toUserJSON(u store.User) userJSON {
	return userJSON{PersonID: u.PersonID, Name: u.Name, Email: u.Email}
}

// meJSON is the logged-in user together with their settings, which other users do not see.
type meJSON struct {
	userJSON
	// Theme is null as long as the user has not chosen one.
	Theme *store.Theme `json:"theme"`
	// ReminderMail says whether the user gets the monthly reminder mail.
	ReminderMail bool `json:"reminderMail"`
}

func toMeJSON(u store.User) meJSON {
	out := meJSON{userJSON: toUserJSON(u), ReminderMail: u.ReminderMail}
	if u.Theme != store.NoTheme {
		out.Theme = &u.Theme
	}
	return out
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	u, err := s.Store.UserByEmail(r.Context(), req.Email)
	found := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.internalError(w, r, err)
		return
	}
	hash := u.PasswordHash
	if !found {
		hash = auth.DummyHash
	}
	ok, err := auth.CheckPassword(hash, req.Password)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !found || !ok {
		writeError(w, http.StatusUnauthorized, "login_failed", "E-Mail oder Passwort falsch")
		return
	}
	if err := s.startSession(w, r, u.PersonID); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMeJSON(u))
}

// startSession creates a session and sets the cookie.
func (s *server) startSession(w http.ResponseWriter, r *http.Request, personID int64) error {
	token, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	now := s.Now()
	if err := s.Store.CreateSession(r.Context(), hash, personID, now, now.Add(sessionDuration)); err != nil {
		return err
	}
	s.setSessionCookie(w, token, int(sessionDuration.Seconds()))
	return nil
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := s.Store.DeleteSession(r.Context(), auth.TokenHash(c.Value)); err != nil {
			s.internalError(w, r, err)
			return
		}
	}
	s.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// setSessionCookie sets the cookie; maxAge -1 deletes it in the browser.
func (s *server) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.Config.WithTLS() || s.Config.BehindProxy,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toMeJSON(userFrom(r.Context())))
}

func (s *server) setTheme(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Theme store.Theme `json:"theme"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !req.Theme.Selectable() {
		writeError(w, http.StatusBadRequest, "unknown_theme", "Unbekanntes Design")
		return
	}
	u, err := s.Store.SetTheme(r.Context(), userFrom(r.Context()).PersonID, req.Theme)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMeJSON(u))
}

type userKey struct{}

func userFrom(ctx context.Context) store.User {
	u, _ := ctx.Value(userKey{}).(store.User)
	return u
}

// requireSession only lets requests with a valid session through.
func (s *server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notLoggedIn := func() {
			writeError(w, http.StatusUnauthorized, "not_logged_in", "Nicht angemeldet")
		}
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			notLoggedIn()
			return
		}
		u, err := s.Store.SessionUser(r.Context(), auth.TokenHash(c.Value), s.Now())
		if errors.Is(err, store.ErrNotFound) {
			notLoggedIn()
			return
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}
