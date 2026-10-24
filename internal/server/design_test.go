package server_test

import (
	"net/http"
	"testing"
)

func anmeldenAls(e *testEnv, email, passwort string) (*apiClient, response) {
	e.t.Helper()
	c := e.client()
	res := c.mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": email, "passwort": passwort})
	return c, res
}

func TestNeuerNutzerHatNochKeinDesign(t *testing.T) {
	e := newTestEnv(t)
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")
	c, anmeldung := anmeldenAls(e, "alex@example.org", "geheim-geheim")

	if v, da := anmeldung.Body["design"]; !da || v != nil {
		t.Fatalf("Anmeldung: design soll null sein: %s", anmeldung.Raw)
	}
	ich := c.mustStatus(http.StatusOK, "GET", "/api/ich", nil)
	if v, da := ich.Body["design"]; !da || v != nil {
		t.Fatalf("ich: design soll null sein: %s", ich.Raw)
	}
}

func TestDesignGiltAuchAufEinemAnderenGeraet(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()

	res := alex.mustStatus(http.StatusOK, "PUT", "/api/ich/design", map[string]any{"design": "neon"})
	if res.Body["design"] != "neon" || res.Body["email"] != "alex@example.org" {
		t.Fatalf("Antwort: %s", res.Raw)
	}
	if ich := alex.mustStatus(http.StatusOK, "GET", "/api/ich", nil); ich.Body["design"] != "neon" {
		t.Fatalf("ich: %s", ich.Raw)
	}

	alex.mustStatus(http.StatusNoContent, "POST", "/api/abmeldung", nil)
	anderesGeraet, anmeldung := anmeldenAls(e, "alex@example.org", "geheim-geheim")
	if anmeldung.Body["design"] != "neon" {
		t.Fatalf("Anmeldung: %s", anmeldung.Raw)
	}
	anderesGeraet.mustStatus(http.StatusOK, "PUT", "/api/ich/design", map[string]any{"design": "weich"})
	if ich := anderesGeraet.mustStatus(http.StatusOK, "GET", "/api/ich", nil); ich.Body["design"] != "weich" {
		t.Fatalf("ich: %s", ich.Raw)
	}
}

func TestDesignGehoertNurDemEigenenNutzer(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	e.createNutzer("Kim", "kim@example.org", "kims-passwort")
	kim, _ := anmeldenAls(e, "kim@example.org", "kims-passwort")

	alex.mustStatus(http.StatusOK, "PUT", "/api/ich/design", map[string]any{"design": "minimal"})

	if ich := kim.mustStatus(http.StatusOK, "GET", "/api/ich", nil); ich.Body["design"] != nil {
		t.Fatalf("Kim: %s", ich.Raw)
	}
	// Andere Nutzer sehen die Einstellung nicht.
	liste := kim.mustStatus(http.StatusOK, "GET", "/api/nutzer", nil)
	for _, n := range liste.Body["nutzer"].([]any) {
		if _, da := n.(map[string]any)["design"]; da {
			t.Fatalf("Nutzerliste zeigt das Design: %s", liste.Raw)
		}
	}
}

func TestUnbekanntesDesignWirdAbgelehnt(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	alex.mustStatus(http.StatusOK, "PUT", "/api/ich/design", map[string]any{"design": "neon"})

	for _, body := range []map[string]any{
		{"design": "pink"},
		{"design": ""},
		{"design": nil},
		{},
	} {
		res := alex.mustStatus(http.StatusBadRequest, "PUT", "/api/ich/design", body)
		if res.errorCode() != "unbekanntes_design" {
			t.Errorf("%v: Fehlercode %q", body, res.errorCode())
		}
	}
	if ich := alex.mustStatus(http.StatusOK, "GET", "/api/ich", nil); ich.Body["design"] != "neon" {
		t.Fatalf("ich: %s", ich.Raw)
	}
}

func TestDesignSetzenBrauchtEineSitzung(t *testing.T) {
	e := newTestEnv(t)
	res := e.client().mustStatus(http.StatusUnauthorized, "PUT", "/api/ich/design", map[string]any{"design": "neon"})
	if res.errorCode() != "nicht_angemeldet" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}
