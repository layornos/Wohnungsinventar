package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"inventar/internal/auth"
	"inventar/internal/store"
)

const einladungsDauer = 7 * 24 * time.Hour

type einladungJSON struct {
	ID int64 `json:"id"`
	// PersonID ist null, wenn die Person erst beim Annehmen angelegt wird.
	PersonID   *int64    `json:"personId"`
	Name       string    `json:"name"`
	Ersteller  string    `json:"ersteller"`
	ErstelltAm time.Time `json:"erstelltAm"`
	LaeuftAbAm time.Time `json:"laeuftAbAm"`
}

func alsEinladungJSON(e store.Einladung) einladungJSON {
	j := einladungJSON{ID: e.ID, Name: e.Name, Ersteller: e.Ersteller, ErstelltAm: e.ErstelltAm, LaeuftAbAm: e.LaeuftAbAm}
	if e.PersonID != 0 {
		j.PersonID = &e.PersonID
	}
	return j
}

// einladungsFehler übersetzt die fachlichen Fehler rund um Einladungen, alles andere wie storeFehler.
func (s *server) einladungsFehler(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrEinladungAbgelaufen):
		schreibeFehler(w, http.StatusGone, "einladung_abgelaufen", "Diese Einladung ist abgelaufen. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrEinladungVerwendet):
		schreibeFehler(w, http.StatusGone, "einladung_verwendet", "Diese Einladung wurde schon verwendet. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrEinladungZurueckgezogen):
		schreibeFehler(w, http.StatusGone, "einladung_zurueckgezogen", "Diese Einladung wurde zurückgezogen. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrBereitsNutzer):
		schreibeFehler(w, http.StatusConflict, "bereits_nutzer", "Diese Person ist bereits Nutzer")
	case errors.Is(err, store.ErrEmailVergeben):
		schreibeFehler(w, http.StatusConflict, "email_vergeben", "Diese E-Mail-Adresse ist bereits vergeben")
	default:
		s.storeFehler(w, r, err)
	}
}

func einladungUngueltig(w http.ResponseWriter) {
	schreibeFehler(w, http.StatusNotFound, "einladung_ungueltig", "Diese Einladung gibt es nicht. Prüfe den Link oder bitte um einen neuen.")
}

// einladungsLink baut den Link aus der Adresse, unter der die App gerade aufgerufen wird,
// damit er im Heimnetz und über das VPN gleich ist. Der Token steht im Fragment und
// erreicht so beim Öffnen des Links weder den Server noch dessen Logs.
func einladungsLink(r *http.Request, token string) string {
	schema := "http"
	if r.TLS != nil {
		schema = "https"
	}
	return schema + "://" + r.Host + "/einladung#" + token
}

func (s *server) einladungAnlegen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PersonID int64  `json:"personId"`
		Name     string `json:"name"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	neu := store.NeueEinladung{PersonID: req.PersonID, ErstellerID: nutzerAus(r.Context()).PersonID}
	if neu.PersonID == 0 {
		neu.Name = strings.TrimSpace(req.Name)
		if neu.Name == "" {
			nameFehlt(w)
			return
		}
	}
	token, hash, err := auth.NeuesToken()
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	neu.TokenHash = hash
	jetzt := s.Jetzt()
	e, err := s.Store.EinladungAnlegen(r.Context(), neu, jetzt, jetzt.Add(einladungsDauer))
	if err != nil {
		s.einladungsFehler(w, r, err)
		return
	}
	// Der Link wird nur hier einmalig geliefert; gespeichert ist nur der Hash.
	schreibeJSON(w, http.StatusCreated, struct {
		einladungJSON
		Link string `json:"link"`
	}{alsEinladungJSON(e), einladungsLink(r, token)})
}

func (s *server) einladungenAuflisten(w http.ResponseWriter, r *http.Request) {
	liste, err := s.Store.EinladungenOffen(r.Context(), s.Jetzt())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	out := make([]einladungJSON, 0, len(liste))
	for _, e := range liste {
		out = append(out, alsEinladungJSON(e))
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"einladungen": out})
}

func (s *server) einladungZurueckziehen(w http.ResponseWriter, r *http.Request) {
	id, ok := pfadID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.EinladungZurueckziehen(r.Context(), id, s.Jetzt()); err != nil {
		s.einladungsFehler(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) nutzerAuflisten(w http.ResponseWriter, r *http.Request) {
	liste, err := s.Store.NutzerAuflisten(r.Context())
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	out := make([]nutzerJSON, 0, len(liste))
	for _, n := range liste {
		out = append(out, alsNutzerJSON(n))
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"nutzer": out})
}

// Der Token kommt im Body statt im Pfad, damit er nicht in Logs von Anfragepfaden landet.
type tokenAnfrage struct {
	Token string `json:"token"`
}

// einladungPruefen ist öffentlich und verrät nur bei einer offenen Einladung Name und Ersteller.
func (s *server) einladungPruefen(w http.ResponseWriter, r *http.Request) {
	var req tokenAnfrage
	if !leseJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		einladungUngueltig(w)
		return
	}
	e, err := s.Store.EinladungPerToken(r.Context(), auth.TokenHash(req.Token), s.Jetzt())
	if errors.Is(err, store.ErrNichtGefunden) {
		einladungUngueltig(w)
		return
	}
	if err != nil {
		s.einladungsFehler(w, r, err)
		return
	}
	schreibeJSON(w, http.StatusOK, map[string]any{"name": e.Name, "ersteller": e.Ersteller, "laeuftAbAm": e.LaeuftAbAm})
}

// einladungAnnehmen ist öffentlich: Die eingeladene Person wird Nutzer und ist danach angemeldet.
func (s *server) einladungAnnehmen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		tokenAnfrage
		Email    string `json:"email"`
		Passwort string `json:"passwort"`
	}
	if !leseJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		einladungUngueltig(w)
		return
	}
	email := store.EmailNormalisieren(req.Email)
	if !strings.Contains(email, "@") {
		schreibeFehler(w, http.StatusBadRequest, "email_ungueltig", "Bitte eine gültige E-Mail-Adresse angeben")
		return
	}
	passwortHash, err := auth.PasswortHashen(req.Passwort)
	if errors.Is(err, auth.ErrPasswortZuKurz) {
		schreibeFehler(w, http.StatusBadRequest, "passwort_zu_kurz", auth.ErrPasswortZuKurz.Error())
		return
	}
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	sitzung, sitzungsHash, err := auth.NeuesToken()
	if err != nil {
		s.internerFehler(w, r, err)
		return
	}
	jetzt := s.Jetzt()
	n, err := s.Store.EinladungAnnehmen(r.Context(), store.Annahme{
		TokenHash:       auth.TokenHash(req.Token),
		Email:           email,
		PasswortHash:    passwortHash,
		SitzungsHash:    sitzungsHash,
		SitzungLaeuftAb: jetzt.Add(sitzungsDauer),
	}, jetzt)
	if errors.Is(err, store.ErrNichtGefunden) {
		einladungUngueltig(w)
		return
	}
	if err != nil {
		s.einladungsFehler(w, r, err)
		return
	}
	s.sitzungsCookieSetzen(w, sitzung, int(sitzungsDauer.Seconds()))
	schreibeJSON(w, http.StatusOK, alsIchJSON(n))
}
