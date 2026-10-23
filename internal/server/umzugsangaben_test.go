package server_test

import (
	"net/http"
	"testing"
)

// angabe liefert den Namen eines Raums oder die Nummer der Box aus den Umzugsangaben eines Items.
func angabe(t *testing.T, item response, feld string) any {
	t.Helper()
	u, _ := item.Body["umzug"].(map[string]any)
	if u == nil || u[feld] == nil {
		return nil
	}
	v := u[feld].(map[string]any)
	if feld == "box" {
		return int(v["nummer"].(float64))
	}
	return v["name"]
}

func TestHerkunftsraumWirdAusDerPlatzierungVorbelegt(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	schere := c.itemIn("Schere", u.schublade)

	res := c.mustStatus(http.StatusOK, "PATCH", itemPfad(schere)+"/umzug", map[string]any{"zielraumId": u.buero})
	if angabe(t, res, "herkunftsraum") != "Arbeitszimmer" || angabe(t, res, "zielraum") != "Büro" {
		t.Fatalf("Umzugsangaben: %s", res.Raw)
	}
	// Die Platzierung bleibt während des Umzugs unverändert.
	if got := pfadText(t, res.Body["platzierung"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("Platzierung %q", got)
	}

	// Herkunftsraum lässt sich ändern und zurücksetzen; nicht genannte Angaben bleiben.
	flur := c.raumAnlegen(u.wohnung, "Flur")
	res = c.mustStatus(http.StatusOK, "PATCH", itemPfad(schere)+"/umzug", map[string]any{"herkunftsraumId": flur})
	if angabe(t, res, "herkunftsraum") != "Flur" || angabe(t, res, "zielraum") != "Büro" {
		t.Fatalf("geändert: %s", res.Raw)
	}
	res = c.mustStatus(http.StatusOK, "PATCH", itemPfad(schere)+"/umzug", map[string]any{"herkunftsraumId": nil, "zielraumId": nil})
	if angabe(t, res, "herkunftsraum") != nil || angabe(t, res, "zielraum") != nil {
		t.Fatalf("zurückgesetzt: %s", res.Raw)
	}

	ohnePlatz := c.itemAnlegen("Kiste", []int64{c.ich()}, testJPEG(t))
	res = c.mustStatus(http.StatusOK, "PATCH", itemPfad(ohnePlatz)+"/umzug", map[string]any{"zielraumId": u.kueche})
	if angabe(t, res, "herkunftsraum") != nil {
		t.Fatalf("ohne Platzierung kein Herkunftsraum: %s", res.Raw)
	}
}

func TestUmzugsangabenWerdenGeprueft(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))

	for _, fall := range []struct {
		body   map[string]any
		status int
		code   string
	}{
		{map[string]any{"zielraumId": u.raum}, http.StatusBadRequest, "zielraum_nicht_in_neuer_wohnung"},
		{map[string]any{"zielraumId": 999}, http.StatusBadRequest, "unbekannter_platz"},
		{map[string]any{"herkunftsraumId": 999}, http.StatusBadRequest, "unbekannter_platz"},
		{map[string]any{"boxId": 999}, http.StatusBadRequest, "unbekannte_box"},
		{map[string]any{"boxId": "eins"}, http.StatusBadRequest, "ungueltige_anfrage"},
	} {
		if r := c.do("PATCH", itemPfad(item)+"/umzug", fall.body); r.Status != fall.status || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
	if res := c.get(itemPfad(item)); res.Body["umzug"] != nil {
		t.Fatalf("abgelehnte Änderungen sollen keine Angaben anlegen: %s", res.Raw)
	}
}

func TestSammeloperationGanzOderGarNicht(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	box, nummer := c.boxAnlegen("Büro")
	a := c.itemIn("Locher", u.schreibtisch)
	b := c.itemIn("Tacker", u.schublade)

	c.mustStatus(http.StatusNoContent, "POST", "/api/umzug/sammeln", map[string]any{
		"itemIds": []any{a.Body["id"], b.Body["id"]}, "boxId": box, "zielraumId": u.buero,
	})
	for _, item := range []response{a, b} {
		got := c.get(itemPfad(item))
		if angabe(t, got, "box") != nummer || angabe(t, got, "zielraum") != "Büro" || angabe(t, got, "herkunftsraum") != "Arbeitszimmer" {
			t.Errorf("%s: %s", item.Body["name"], got.Raw)
		}
	}

	// Ein unbekanntes Item verhindert die ganze Änderung.
	r := c.do("POST", "/api/umzug/sammeln", map[string]any{"itemIds": []any{a.Body["id"], 999}, "zielraumId": u.kueche})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unbekanntes_item" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
	if got := angabe(t, c.get(itemPfad(a)), "zielraum"); got != "Büro" {
		t.Fatalf("Zielraum nach abgelehnter Sammeloperation: %v", got)
	}
	if r := c.do("POST", "/api/umzug/sammeln", map[string]any{"itemIds": []any{}, "zielraumId": u.kueche}); r.Status != http.StatusBadRequest {
		t.Fatalf("ohne Items: Status %d", r.Status)
	}
}

func TestOhneUmzugGibtEsKeineUmzugsangaben(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))
	r := c.do("PATCH", itemPfad(item)+"/umzug", map[string]any{"zielraumId": 1})
	if r.Status != http.StatusConflict || r.errorCode() != "kein_umzug" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
}
