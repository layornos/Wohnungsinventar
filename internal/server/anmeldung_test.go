package server_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestPerUnterbefehlAngelegterNutzerKannSichAnmelden(t *testing.T) {
	e := newTestEnv(t)
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")

	c := e.client()
	c.mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "Alex@Example.org", "passwort": "geheim-geheim"})

	me := c.mustStatus(http.StatusOK, "GET", "/api/ich", nil)
	if me.Body["name"] != "Alex" || me.Body["email"] != "alex@example.org" {
		t.Fatalf("unerwarteter Nutzer: %s", me.Raw)
	}
}

func TestMitFalschemPasswortEntstehtKeineSitzung(t *testing.T) {
	e := newTestEnv(t)
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")

	c := e.client()
	res := c.mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "falsch-falsch"})
	if res.errorCode() != "anmeldung_fehlgeschlagen" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
	c.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)
}

func TestUnbekannteEmailWirdWieFalschesPasswortBehandelt(t *testing.T) {
	e := newTestEnv(t)
	res := e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "niemand@example.org", "passwort": "egal-egal"})
	if res.errorCode() != "anmeldung_fehlgeschlagen" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestOhneSitzungIstDieAPIGesperrt(t *testing.T) {
	e := newTestEnv(t)
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/ich"},
		{"POST", "/api/abmeldung"},
		{"GET", "/api/gibt-es-nicht"},
	} {
		res := e.client().do(req.method, req.path, nil)
		if res.Status != http.StatusUnauthorized || res.errorCode() != "nicht_angemeldet" {
			t.Errorf("%s %s: Status %d, Fehlercode %q", req.method, req.path, res.Status, res.errorCode())
		}
	}
}

func TestNachDemAbmeldenIstDieSitzungUngueltig(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	cookie := c.http.Jar.Cookies(mustURL(t, e.srv.URL))

	c.mustStatus(http.StatusNoContent, "POST", "/api/abmeldung", nil)
	c.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)

	// Auch das alte Cookie, z. B. von einem anderen Gerät kopiert, gilt nicht mehr.
	other := e.client()
	other.http.Jar.SetCookies(mustURL(t, e.srv.URL), cookie)
	other.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)
}

func TestSitzungsCookieIstGeschuetzt(t *testing.T) {
	e := newTestEnv(t)
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")
	res := e.client().mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "geheim-geheim"})

	set := res.Header.Get("Set-Cookie")
	for _, want := range []string{"sitzung=", "HttpOnly", "SameSite=Strict", "Max-Age=7776000"} {
		if !strings.Contains(set, want) {
			t.Errorf("Set-Cookie %q enthält nicht %q", set, want)
		}
	}
	if strings.Contains(set, "Secure") {
		t.Errorf("ohne TLS darf das Cookie nicht Secure sein: %q", set)
	}
}

func TestPasswortResetBeendetAlleSitzungen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()

	if out, err := e.runCLI("ganz-neues-passwort\n", "passwort-zuruecksetzen", "--email", "alex@example.org"); err != nil {
		t.Fatalf("passwort-zuruecksetzen: %v\n%s", err, out)
	}

	c.mustStatus(http.StatusUnauthorized, "GET", "/api/ich", nil)
	neu := e.client()
	neu.mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "geheim-geheim"})
	neu.mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "ganz-neues-passwort"})
}

func TestEmailKannNurEinmalVergebenWerden(t *testing.T) {
	e := newTestEnv(t)
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")
	if _, err := e.runCLI("anderes-passwort\n", "nutzer-anlegen", "--name", "Kim", "--email", "ALEX@example.org"); err == nil {
		t.Fatal("zweiter Nutzer mit derselben E-Mail wurde angelegt")
	}
}

func TestZuKurzesPasswortWirdAbgelehnt(t *testing.T) {
	e := newTestEnv(t)
	if _, err := e.runCLI("kurz\n", "nutzer-anlegen", "--name", "Alex", "--email", "alex@example.org"); err == nil {
		t.Fatal("Nutzer mit zu kurzem Passwort wurde angelegt")
	}
	e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "kurz"})
}

func TestUnbekannterAPIEndpunktLiefertJSONFehler(t *testing.T) {
	e := newTestEnv(t)
	res := e.loggedIn().mustStatus(http.StatusNotFound, "GET", "/api/gibt-es-nicht", nil)
	if res.errorCode() != "nicht_gefunden" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}
