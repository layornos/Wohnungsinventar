package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestAbschlussStelltItemsInIhreZielraeume(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	box, nummer := c.boxAnlegen("Büro")
	locher := c.itemIn("Locher", u.schublade)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(locher)+"/umzug", map[string]any{"boxId": box, "zielraumId": u.buero})
	topf := c.itemIn("Topf", u.schreibtisch)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(topf)+"/umzug", map[string]any{"zielraumId": u.kueche})

	res := c.mustStatus(http.StatusOK, "POST", "/api/umzug/abschluss", map[string]any{})
	if res.Body["laufend"] != false || res.Body["abgeschlossenAm"] == nil {
		t.Fatalf("Abschluss: %s", res.Raw)
	}
	for item, raum := range map[*response]string{&locher: "Neue Wohnung › Büro", &topf: "Neue Wohnung › Küche"} {
		got := c.get(itemPfad(*item))
		if p := pfadText(t, got.Body["platzierung"]); p != raum {
			t.Errorf("%s: %q, erwartet %q", item.Body["name"], p, raum)
		}
		if got.Body["umzug"] != nil {
			t.Errorf("%s: nach dem Abschluss in keiner Box mehr erwartet: %s", item.Body["name"], got.Raw)
		}
	}
	if got := c.namenInListe(fmt.Sprintf("?ohneLagerort=1&raum=%d", u.buero)); !slices.Equal(got, []string{"Locher"}) {
		t.Fatalf("ohne Lagerort im Büro: %v", got)
	}
	if res := c.mustStatus(http.StatusOK, "GET", "/api/umzug", nil); res.Body["umzug"] != nil {
		t.Fatalf("es soll kein Umzug mehr laufen: %s", res.Raw)
	}

	// Der abgeschlossene Umzug bleibt sichtbar, samt Packliste seiner Boxen.
	liste := c.mustStatus(http.StatusOK, "GET", "/api/umzuege", nil).Body["umzuege"].([]any)
	if len(liste) != 1 {
		t.Fatalf("abgeschlossene Umzüge: %v", liste)
	}
	pack := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/boxen/%d", box), nil)
	if got := zeilen(t, pack.Body["items"].([]any)); !slices.Equal(got, []string{fmt.Sprintf("Locher %d Arbeitszimmer→Büro", nummer)}) {
		t.Fatalf("Packliste im Verlauf: %v", got)
	}
}

func TestOhneBestaetigungVerhindernItemsOhneZielraumDenAbschluss(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	vergessen := c.itemIn("Tacker", u.schublade)

	res := c.mustStatus(http.StatusConflict, "POST", "/api/umzug/abschluss", map[string]any{})
	if res.errorCode() != "items_ohne_zielraum" || !slices.Equal(namenVon(res), []string{"Tacker"}) {
		t.Fatalf("ohne Bestätigung: %s", res.Raw)
	}
	c.mustStatus(http.StatusOK, "GET", "/api/umzug", nil)

	c.mustStatus(http.StatusOK, "POST", "/api/umzug/abschluss", map[string]any{"bestaetigt": true})
	if p := pfadText(t, c.get(itemPfad(vergessen)).Body["platzierung"]); p != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("Item ohne Zielraum soll bleiben, wo es war: %q", p)
	}
}

func TestAbgeschlossenerUmzugIstUnveraenderlich(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	box, _ := c.boxAnlegen("Büro")
	item := c.itemIn("Locher", u.schublade)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(item)+"/umzug", map[string]any{"boxId": box, "zielraumId": u.buero})
	c.mustStatus(http.StatusOK, "POST", "/api/umzug/abschluss", map[string]any{})

	for _, fall := range []struct {
		method, path string
		body         map[string]any
		code         string
	}{
		{"PUT", fmt.Sprintf("/api/boxen/%d", box), map[string]any{"bezeichnung": "x"}, "umzug_abgeschlossen"},
		{"DELETE", fmt.Sprintf("/api/boxen/%d", box), nil, "umzug_abgeschlossen"},
		{"PATCH", itemPfad(item) + "/umzug", map[string]any{"zielraumId": u.kueche}, "kein_umzug"},
		{"POST", "/api/umzug/boxen", map[string]any{}, "kein_umzug"},
		{"POST", "/api/umzug/abschluss", map[string]any{}, "kein_umzug"},
	} {
		if r := c.do(fall.method, fall.path, fall.body); r.Status != http.StatusConflict || r.errorCode() != fall.code {
			t.Errorf("%s %s: Status %d, Fehlercode %q", fall.method, fall.path, r.Status, r.errorCode())
		}
	}

	// Ein neuer Umzug ist danach möglich, z. B. zurück.
	c.mustStatus(http.StatusCreated, "POST", "/api/umzug", map[string]any{"alteWohnungId": u.neu, "neueWohnungId": u.wohnung})
}

func TestRaumImUmzugIstNichtLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(item)+"/umzug", map[string]any{"zielraumId": u.kueche})

	r := c.do("DELETE", fmt.Sprintf("/api/raeume/%d", u.kueche), nil)
	if r.Status != http.StatusConflict || r.errorCode() != "raum_im_umzug" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
}
