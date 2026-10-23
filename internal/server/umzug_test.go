package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

// umzugsAufbau: Altbau › Arbeitszimmer › Schreibtisch › obere Schublade, dazu die neue Wohnung
// mit Büro und Küche und ein laufender Umzug.
type umzugsAufbau struct {
	arbeitszimmer
	neu, buero, kueche int64
}

func (c *apiClient) umzugAnlegen() umzugsAufbau {
	c.t.Helper()
	u := umzugsAufbau{arbeitszimmer: c.arbeitszimmerAnlegen()}
	u.neu = c.wohnungAnlegen("Neue Wohnung")
	u.buero = c.raumAnlegen(u.neu, "Büro")
	u.kueche = c.raumAnlegen(u.neu, "Küche")
	c.mustStatus(http.StatusCreated, "POST", "/api/umzug", map[string]any{"alteWohnungId": u.wohnung, "neueWohnungId": u.neu})
	return u
}

func TestUmzugAnlegenUndAbrufen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	if res := c.mustStatus(http.StatusOK, "GET", "/api/umzug", nil); res.Body["umzug"] != nil {
		t.Fatalf("ohne Umzug: %s", res.Raw)
	}
	u := c.umzugAnlegen()

	res := c.mustStatus(http.StatusOK, "GET", "/api/umzug", nil)
	umzug := res.Body["umzug"].(map[string]any)
	if umzug["alteWohnung"].(map[string]any)["name"] != "Altbau" || umzug["neueWohnung"].(map[string]any)["name"] != "Neue Wohnung" {
		t.Fatalf("Umzug: %s", res.Raw)
	}

	r := c.do("POST", "/api/umzug", map[string]any{"alteWohnungId": u.neu, "neueWohnungId": u.wohnung})
	if r.Status != http.StatusConflict || r.errorCode() != "umzug_laeuft" {
		t.Fatalf("zweiter Umzug: Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
}

func TestUmzugBrauchtZweiVerschiedeneWohnungen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	w := c.wohnungAnlegen("Altbau")
	for _, fall := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"alteWohnungId": w, "neueWohnungId": w}, "gleiche_wohnung"},
		{map[string]any{"alteWohnungId": w, "neueWohnungId": 999}, "unbekannte_wohnung"},
	} {
		if r := c.do("POST", "/api/umzug", fall.body); r.Status != http.StatusBadRequest || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
}

func TestWohnungMitUmzugIstNichtLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	alt := c.wohnungAnlegen("Altbau")
	neu := c.wohnungAnlegen("Neue Wohnung")
	c.mustStatus(http.StatusCreated, "POST", "/api/umzug", map[string]any{"alteWohnungId": alt, "neueWohnungId": neu})

	for _, w := range []int64{alt, neu} {
		r := c.do("DELETE", fmt.Sprintf("/api/wohnungen/%d", w), nil)
		if r.Status != http.StatusConflict || r.errorCode() != "wohnung_im_umzug" {
			t.Errorf("Wohnung %d: Status %d, Fehlercode %q", w, r.Status, r.errorCode())
		}
	}
}
