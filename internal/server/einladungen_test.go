package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"inventar/internal/server"
)

// testUhr ist eine steuerbare Uhr für den Server, damit Tests den Ablauf prüfen können.
type testUhr struct {
	mu    sync.Mutex
	jetzt time.Time
}

func (u *testUhr) Jetzt() time.Time {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.jetzt
}

func (u *testUhr) vorstellen(d time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.jetzt = u.jetzt.Add(d)
}

func newTestEnvMitUhr(t *testing.T) (*testEnv, *testUhr) {
	t.Helper()
	uhr := &testUhr{jetzt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	return newTestEnvMit(t, func(d *server.Deps) { d.Jetzt = uhr.Jetzt }), uhr
}

// einladen legt eine Einladung an und liefert den Token aus dem Link.
func (c *apiClient) einladen(body map[string]any) (token string, res response) {
	c.t.Helper()
	res = c.mustStatus(http.StatusCreated, "POST", "/api/einladungen", body)
	link, _ := res.Body["link"].(string)
	u, err := url.Parse(link)
	if err != nil || u.Fragment == "" {
		c.t.Fatalf("Link %q ohne Token", link)
	}
	return u.Fragment, res
}

func annahme(token, email string) map[string]any {
	return map[string]any{"token": token, "email": email, "passwort": "kims-passwort"}
}

func TestNachDemAnnehmenKannSichDieNeuePersonAnmelden(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, _ := alex.einladen(map[string]any{"name": "Kim"})

	kim := e.client()
	angenommen := kim.mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "Kim@Example.org"))
	if angenommen.Body["name"] != "Kim" || angenommen.Body["email"] != "kim@example.org" {
		t.Fatalf("angenommen: %s", angenommen.Raw)
	}
	// Nach dem Annehmen ist Kim direkt angemeldet …
	if ich := kim.mustStatus(http.StatusOK, "GET", "/api/ich", nil); ich.Body["name"] != "Kim" {
		t.Fatalf("ich: %s", ich.Raw)
	}
	// … und kann sich später mit E-Mail und Passwort anmelden.
	e.client().mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "kim@example.org", "passwort": "kims-passwort"})
}

func TestLinkWirdAusDerAufgerufenenAdresseGebildet(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/einladungen", strings.NewReader(`{"name":"Kim"}`))
	req.Host = "inventar.example.duckdns.org"
	// Der Cookie-Jar ordnet Cookies dem Host zu, deshalb das Sitzungs-Cookie selbst mitgeben.
	for _, c := range alex.http.Jar.Cookies(mustURL(t, e.srv.URL)) {
		req.AddCookie(c)
	}
	res := alex.send(req)
	if res.Status != http.StatusCreated {
		t.Fatalf("Status %d: %s", res.Status, res.Raw)
	}
	link, _ := res.Body["link"].(string)
	if !strings.HasPrefix(link, "http://inventar.example.duckdns.org/einladung#") {
		t.Fatalf("Link %q", link)
	}
	// Mindestens 128 Bit, URL-sicher kodiert.
	token := strings.TrimPrefix(link, "http://inventar.example.duckdns.org/einladung#")
	if len(token) < 22 || strings.ContainsAny(token, "+/=") {
		t.Fatalf("Token %q", token)
	}
}

func TestOffeneEinladungZeigtNameUndErsteller(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().einladen(map[string]any{"name": "Kim"})

	res := e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/pruefen", map[string]any{"token": token})
	if res.Body["name"] != "Kim" || res.Body["ersteller"] != "Alex" {
		t.Fatalf("Prüfen: %s", res.Raw)
	}
}

func TestEinladungLaeuftNachSiebenTagenAb(t *testing.T) {
	e, uhr := newTestEnvMitUhr(t)
	alex := e.loggedIn()
	token, _ := alex.einladen(map[string]any{"name": "Kim"})

	uhr.vorstellen(7*24*time.Hour - time.Minute)
	e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/pruefen", map[string]any{"token": token})

	uhr.vorstellen(2 * time.Minute)
	kim := e.client()
	for _, pfad := range []string{"/api/einladungen/pruefen", "/api/einladungen/annehmen"} {
		res := kim.mustStatus(http.StatusGone, "POST", pfad, annahme(token, "kim@example.org"))
		if res.errorCode() != "einladung_abgelaufen" || res.Body["name"] != nil {
			t.Errorf("%s: %s", pfad, res.Raw)
		}
	}
	kim.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)
	if liste := alex.mustStatus(http.StatusOK, "GET", "/api/einladungen", nil); len(liste.Body["einladungen"].([]any)) != 0 {
		t.Fatalf("abgelaufene Einladung ist noch offen: %s", liste.Raw)
	}
}

func TestEinladungIstNurEinmalVerwendbar(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().einladen(map[string]any{"name": "Kim"})
	e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))

	andere := e.client()
	for _, pfad := range []string{"/api/einladungen/pruefen", "/api/einladungen/annehmen"} {
		res := andere.mustStatus(http.StatusGone, "POST", pfad, annahme(token, "kim2@example.org"))
		if res.errorCode() != "einladung_verwendet" {
			t.Errorf("%s: %s", pfad, res.Raw)
		}
	}
	andere.mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "kim2@example.org", "passwort": "kims-passwort"})
}

func TestZurueckgezogeneEinladungFunktioniertNichtMehr(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, angelegt := alex.einladen(map[string]any{"name": "Kim"})
	id := angelegt.Body["id"]

	liste := alex.mustStatus(http.StatusOK, "GET", "/api/einladungen", nil)
	offen := liste.Body["einladungen"].([]any)
	if len(offen) != 1 {
		t.Fatalf("offene Einladungen: %s", liste.Raw)
	}
	eintrag := offen[0].(map[string]any)
	if eintrag["name"] != "Kim" || eintrag["ersteller"] != "Alex" || eintrag["laeuftAbAm"] == nil || eintrag["link"] != nil {
		t.Fatalf("Eintrag: %v", eintrag)
	}

	alex.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/einladungen/%v", id), nil)
	if liste := alex.mustStatus(http.StatusOK, "GET", "/api/einladungen", nil); len(liste.Body["einladungen"].([]any)) != 0 {
		t.Fatalf("zurückgezogene Einladung ist noch offen: %s", liste.Raw)
	}
	for _, pfad := range []string{"/api/einladungen/pruefen", "/api/einladungen/annehmen"} {
		res := e.client().mustStatus(http.StatusGone, "POST", pfad, annahme(token, "kim@example.org"))
		if res.errorCode() != "einladung_zurueckgezogen" {
			t.Errorf("%s: %s", pfad, res.Raw)
		}
	}
	alex.mustStatus(http.StatusNotFound, "DELETE", "/api/einladungen/999", nil)
}

func TestAngenommeneEinladungLaesstSichNichtZurueckziehen(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, angelegt := alex.einladen(map[string]any{"name": "Kim"})
	e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))

	res := alex.mustStatus(http.StatusGone, "DELETE", fmt.Sprintf("/api/einladungen/%v", angelegt.Body["id"]), nil)
	if res.errorCode() != "einladung_verwendet" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
	e.client().mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "kim@example.org", "passwort": "kims-passwort"})
}

func TestVerknuepftePersonWirdNutzerUndBleibtBesitzer(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	kimID := alex.personAnlegen("Kim")
	item := alex.itemAnlegen("Kuscheltier", []int64{kimID}, testJPEG(t))
	token, angelegt := alex.einladen(map[string]any{"personId": kimID})
	if angelegt.Body["name"] != "Kim" || angelegt.Body["personId"] != float64(kimID) {
		t.Fatalf("angelegt: %s", angelegt.Raw)
	}

	kim := e.client()
	kim.mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))
	if got := kim.ich(); got != kimID {
		t.Fatalf("Kim ist Person %d, erwartet %d", got, kimID)
	}
	nachher := kim.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", item.Body["id"]), nil)
	if got := besitzerNamen(nachher); len(got) != 1 || got[0] != "Kim" {
		t.Fatalf("Besitzer: %v", got)
	}
	// Keine zweite Person Kim, und Kim hat jetzt einen Nutzerzugang.
	personen := alex.mustStatus(http.StatusOK, "GET", "/api/personen", nil).Body["personen"].([]any)
	if len(personen) != 2 {
		t.Fatalf("Personen: %v", personen)
	}
	for _, p := range personen {
		if p := p.(map[string]any); p["name"] == "Kim" && p["nutzer"] != true {
			t.Fatalf("Kim ist kein Nutzer: %v", p)
		}
	}
}

func TestNeuePersonWirdBeimAnnehmenAngelegt(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, angelegt := alex.einladen(map[string]any{"name": " Kim "})
	if angelegt.Body["personId"] != nil || angelegt.Body["name"] != "Kim" {
		t.Fatalf("angelegt: %s", angelegt.Raw)
	}
	// Vor dem Annehmen gibt es die Person noch nicht.
	if p := alex.mustStatus(http.StatusOK, "GET", "/api/personen", nil).Body["personen"].([]any); len(p) != 1 {
		t.Fatalf("Personen vor dem Annehmen: %v", p)
	}
	e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))

	nutzer := alex.mustStatus(http.StatusOK, "GET", "/api/nutzer", nil)
	liste := nutzer.Body["nutzer"].([]any)
	if len(liste) != 2 || liste[0].(map[string]any)["name"] != "Alex" || liste[1].(map[string]any)["email"] != "kim@example.org" {
		t.Fatalf("Nutzer: %s", nutzer.Raw)
	}
	if strings.Contains(string(nutzer.Raw), "argon2") {
		t.Fatalf("Nutzerliste enthält Passwort-Hashes: %s", nutzer.Raw)
	}
}

func TestEinladungOhneNamenOderMitUnbekannterPersonWirdAbgelehnt(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	if res := alex.mustStatus(http.StatusBadRequest, "POST", "/api/einladungen", map[string]any{"name": " "}); res.errorCode() != "name_fehlt" {
		t.Errorf("ohne Namen: %s", res.Raw)
	}
	if res := alex.mustStatus(http.StatusBadRequest, "POST", "/api/einladungen", map[string]any{"personId": 999}); res.errorCode() != "unbekannte_person" {
		t.Errorf("unbekannte Person: %s", res.Raw)
	}
}

func TestBeimAnnehmenIstDieEmailEindeutig(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, _ := alex.einladen(map[string]any{"name": "Kim"})

	kim := e.client()
	res := kim.mustStatus(http.StatusConflict, "POST", "/api/einladungen/annehmen", annahme(token, "ALEX@example.org"))
	if res.errorCode() != "email_vergeben" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
	// Abgelehnt heißt: nichts ist passiert, die Einladung bleibt offen.
	kim.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)
	if p := alex.mustStatus(http.StatusOK, "GET", "/api/personen", nil).Body["personen"].([]any); len(p) != 1 {
		t.Fatalf("Personen: %v", p)
	}
	kim.mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))
}

func TestAnnehmenPrueftEmailUndPasswort(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().einladen(map[string]any{"name": "Kim"})
	kim := e.client()
	if res := kim.mustStatus(http.StatusBadRequest, "POST", "/api/einladungen/annehmen", annahme(token, "kim")); res.errorCode() != "email_ungueltig" {
		t.Errorf("E-Mail: %s", res.Raw)
	}
	res := kim.mustStatus(http.StatusBadRequest, "POST", "/api/einladungen/annehmen", map[string]any{"token": token, "email": "kim@example.org", "passwort": "kurz"})
	if res.errorCode() != "passwort_zu_kurz" {
		t.Errorf("Passwort: %s", res.Raw)
	}
	kim.mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(token, "kim@example.org"))
}

func TestPersonDieSchonNutzerIstKannNichtEingeladenWerden(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	res := alex.mustStatus(http.StatusConflict, "POST", "/api/einladungen", map[string]any{"personId": alex.ich()})
	if res.errorCode() != "bereits_nutzer" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestVerknuepftePersonDieInzwischenNutzerIstKannNichtAnnehmen(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	kimID := alex.personAnlegen("Kim")
	erste, _ := alex.einladen(map[string]any{"personId": kimID})
	zweite, _ := alex.einladen(map[string]any{"personId": kimID})
	e.client().mustStatus(http.StatusOK, "POST", "/api/einladungen/annehmen", annahme(erste, "kim@example.org"))

	// Schon das Prüfen lehnt ab, damit die Seite gleich erklärt, was los ist, statt erst das Formular zu zeigen.
	for _, pfad := range []string{"/api/einladungen/pruefen", "/api/einladungen/annehmen"} {
		res := e.client().mustStatus(http.StatusConflict, "POST", pfad, annahme(zweite, "kim2@example.org"))
		if res.errorCode() != "bereits_nutzer" {
			t.Fatalf("%s: Fehlercode %q", pfad, res.errorCode())
		}
	}
	// Die zweite Einladung taucht nicht mehr als offen auf.
	if liste := alex.mustStatus(http.StatusOK, "GET", "/api/einladungen", nil); len(liste.Body["einladungen"].([]any)) != 0 {
		t.Fatalf("offene Einladungen: %s", liste.Raw)
	}
}

func TestOeffentlicheEndpunkteVerratenOhneGueltigenTokenNichts(t *testing.T) {
	e := newTestEnv(t)
	e.loggedIn().einladen(map[string]any{"name": "Kim"})
	for _, token := range []string{"", "falsch"} {
		for _, pfad := range []string{"/api/einladungen/pruefen", "/api/einladungen/annehmen"} {
			res := e.client().mustStatus(http.StatusNotFound, "POST", pfad, annahme(token, "kim@example.org"))
			if res.errorCode() != "einladung_ungueltig" || strings.Contains(string(res.Raw), "Kim") || strings.Contains(string(res.Raw), "Alex") {
				t.Errorf("%s mit %q: %s", pfad, token, res.Raw)
			}
		}
	}
	e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "kim@example.org", "passwort": "kims-passwort"})
}

func TestEinladungsverwaltungBrauchtEineSitzung(t *testing.T) {
	e := newTestEnv(t)
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/einladungen"},
		{"POST", "/api/einladungen"},
		{"DELETE", "/api/einladungen/1"},
		{"GET", "/api/nutzer"},
	} {
		res := e.client().do(req.method, req.path, map[string]any{"name": "Kim"})
		if res.Status != http.StatusUnauthorized || res.errorCode() != "nicht_angemeldet" {
			t.Errorf("%s %s: Status %d, Fehlercode %q", req.method, req.path, res.Status, res.errorCode())
		}
	}
}
