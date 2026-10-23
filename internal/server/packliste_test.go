package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// zeilen beschreibt Listen-Items als „Name Box Herkunftsraum→Zielraum“ und sortiert sie.
func zeilen(t *testing.T, items []any) []string {
	t.Helper()
	var out []string
	for _, i := range items {
		item := response{Body: i.(map[string]any)}
		out = append(out, fmt.Sprintf("%s %v %v→%v", item.Body["name"], angabe(t, item, "box"), angabe(t, item, "herkunftsraum"), angabe(t, item, "zielraum")))
	}
	slices.Sort(out)
	return out
}

func TestPacklisteEinerBoxNachNummer(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	c.boxAnlegen("leer")
	box, nummer := c.boxAnlegen("Büro")
	locher := c.itemIn("Locher", u.schublade)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(locher)+"/umzug", map[string]any{"boxId": box, "zielraumId": u.buero})

	// Neu erfasst direkt in die Box, wie im Pack-Modus.
	neu := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", "Tacker"}, {"besitzer", fmt.Sprint(c.ich())}, {"raum", fmt.Sprint(u.raum)}, {"box", fmt.Sprint(box)}},
		fotos:  [][]byte{testJPEG(t)},
	})
	if neu.Status != http.StatusCreated || angabe(t, neu, "box") != nummer || angabe(t, neu, "herkunftsraum") != "Arbeitszimmer" {
		t.Fatalf("in die Box erfasst: Status %d: %s", neu.Status, neu.Raw)
	}

	res := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/umzug/boxen/%d", nummer), nil)
	if res.Body["box"].(map[string]any)["bezeichnung"] != "Büro" {
		t.Fatalf("Box: %s", res.Raw)
	}
	want := []string{"Locher 2 Arbeitszimmer→Büro", "Tacker 2 Arbeitszimmer→<nil>"}
	if got := zeilen(t, res.Body["items"].([]any)); !slices.Equal(got, want) {
		t.Fatalf("Packliste:\n%v\nerwartet:\n%v", got, want)
	}
	for _, item := range res.Body["items"].([]any) {
		if !strings.HasSuffix(item.(map[string]any)["vorschauUrl"].(string), "/vorschau") {
			t.Errorf("ohne Vorschau: %v", item)
		}
	}
	c.mustStatus(http.StatusNotFound, "GET", "/api/umzug/boxen/99", nil)
}

func TestItemListeZeigtBoxUndFiltertNachBoxUndHerkunftsraum(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	flur := c.raumAnlegen(u.wohnung, "Flur")
	box, nummer := c.boxAnlegen("")
	gepackt := c.itemIn("Locher", u.schublade)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(gepackt)+"/umzug", map[string]any{"boxId": box})
	c.itemIn("Tacker", u.schreibtisch) // noch ohne Umzugsangaben, aber im Arbeitszimmer
	schirm := c.itemAnlegen("Schirm", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPfad(schirm)+"/platzierung", map[string]any{"raumId": flur})

	if got := zeilen(t, c.mustStatus(http.StatusOK, "GET", "/api/items?q=locher", nil).Body["items"].([]any)); !slices.Equal(got, []string{fmt.Sprintf("Locher %d Arbeitszimmer→<nil>", nummer)}) {
		t.Fatalf("Suche zeigt Box nicht: %v", got)
	}
	for _, fall := range []struct {
		abfrage string
		namen   []string
	}{
		{fmt.Sprintf("?box=%d", box), []string{"Locher"}},
		{fmt.Sprintf("?herkunftsraum=%d", u.raum), []string{"Locher", "Tacker"}},
		{fmt.Sprintf("?herkunftsraum=%d", flur), []string{"Schirm"}},
	} {
		got := c.namenInListe(fall.abfrage)
		slices.Sort(got)
		if !slices.Equal(got, fall.namen) {
			t.Errorf("%s: %v, erwartet %v", fall.abfrage, got, fall.namen)
		}
	}
}

func TestUebersichtProZielraumUndOhneZielraum(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	u := c.umzugAnlegen()
	box, nummer := c.boxAnlegen("Büro")
	locher := c.itemIn("Locher", u.schublade)
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(locher)+"/umzug", map[string]any{"boxId": box, "zielraumId": u.buero})
	stuhl := c.itemAnlegen("Stuhl", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPfad(stuhl)+"/platzierung", map[string]any{"raumId": u.raum})
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(stuhl)+"/umzug", map[string]any{"zielraumId": u.buero})
	c.itemIn("Tacker", u.schreibtisch)
	c.itemAnlegen("Ohne Platz", []int64{c.ich()}, testJPEG(t)) // gehört zu keiner Wohnung

	res := c.mustStatus(http.StatusOK, "GET", "/api/umzug/zielraeume", nil)
	var got []string
	for _, z := range res.Body["zielraeume"].([]any) {
		z := z.(map[string]any)
		var boxen []string
		for _, b := range z["boxen"].([]any) {
			boxen = append(boxen, fmt.Sprint(b.(map[string]any)["nummer"]))
		}
		got = append(got, fmt.Sprintf("%s: Boxen %v, Items %v", z["raum"].(map[string]any)["name"], boxen, namenVon(response{Body: z})))
	}
	want := []string{fmt.Sprintf("Büro: Boxen [%d], Items [Locher Stuhl]", nummer), "Küche: Boxen [], Items []"}
	if !slices.Equal(got, want) {
		t.Fatalf("Übersicht:\n%v\nerwartet:\n%v", got, want)
	}

	if got := c.namenInListe("?ohneZielraum=1"); !slices.Equal(got, []string{"Tacker"}) {
		t.Fatalf("ohne Zielraum: %v", got)
	}
}
