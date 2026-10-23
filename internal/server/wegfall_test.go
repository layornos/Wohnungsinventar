package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func (c *apiClient) itemIn(name string, lagerort int64) response {
	c.t.Helper()
	res := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", name}, {"besitzer", fmt.Sprint(c.ich())}, {"lagerort", fmt.Sprint(lagerort)}},
		fotos:  [][]byte{testJPEG(c.t)},
	})
	if res.Status != http.StatusCreated {
		c.t.Fatalf("Item anlegen: Status %d: %s", res.Status, res.Raw)
	}
	return res
}

func namenVon(r response) []string {
	var namen []string
	for _, i := range r.Body["items"].([]any) {
		namen = append(namen, i.(map[string]any)["name"].(string))
	}
	slices.Sort(namen)
	return namen
}

func TestLagerortLoeschenStelltSeinenInhaltInDenRaum(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	trenner := c.lagerortAnlegen(map[string]any{"name": "Trenner", "elternId": a.schublade})
	regal := c.lagerortAnlegen(map[string]any{"name": "Regal", "raumId": a.raum})
	stift := c.itemIn("Stift", a.schreibtisch)
	schere := c.itemIn("Schere", a.schublade)
	klammer := c.itemIn("Büroklammer", trenner)
	buch := c.itemIn("Buch", regal)

	vorschau := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/lagerorte/%d/wegfall", a.schreibtisch), nil)
	if got := namenVon(vorschau); !slices.Equal(got, []string{"Büroklammer", "Schere", "Stift"}) {
		t.Fatalf("Vorschau: %v", got)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/lagerorte/%d", a.schreibtisch), nil)
	for _, item := range []response{stift, schere, klammer} {
		if got := pfadText(t, c.get(itemPfad(item)).Body["platzierung"]); got != "Altbau › Arbeitszimmer" {
			t.Errorf("%s: %q", item.Body["name"], got)
		}
	}
	if got := pfadText(t, c.get(itemPfad(buch)).Body["platzierung"]); got != "Altbau › Arbeitszimmer › Regal" {
		t.Errorf("Buch: %q", got)
	}
	for _, l := range []int64{a.schreibtisch, a.schublade, trenner} {
		c.mustStatus(http.StatusNotFound, "GET", fmt.Sprintf("/api/lagerorte/%d", l), nil)
	}
}

func TestMoebelstueckLoeschenLaesstSeineLagerorteWegfallen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	tisch := c.itemAnlegen("Schreibtisch", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d/moebel", a.schreibtisch), map[string]any{"itemId": tisch.Body["id"]})
	schere := c.itemIn("Schere", a.schublade)

	vorschau := c.mustStatus(http.StatusOK, "GET", itemPfad(tisch)+"/wegfall", nil)
	if got := namenVon(vorschau); !slices.Equal(got, []string{"Schere"}) {
		t.Fatalf("Vorschau: %v", got)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", itemPfad(tisch), nil)
	nachher := c.get(itemPfad(schere))
	if got := pfadText(t, nachher.Body["platzierung"]); got != "Altbau › Arbeitszimmer" {
		t.Fatalf("Schere nach dem Löschen des Schreibtischs: %q", got)
	}
	c.mustStatus(http.StatusNotFound, "GET", fmt.Sprintf("/api/lagerorte/%d", a.schublade), nil)
	if got := c.namenInListe("?ohneLagerort=1"); !slices.Equal(got, []string{"Schere"}) {
		t.Fatalf("ohne Lagerort: %v", got)
	}
}

func TestFilterNachRaumUndLagerortUmfassenVerschachtelteLagerorte(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	kueche := c.raumAnlegen(a.wohnung, "Küche")
	c.itemIn("Schere", a.schublade)
	c.itemIn("Stift", a.schreibtisch)
	stuhl := c.itemAnlegen("Stuhl", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPfad(stuhl)+"/platzierung", map[string]any{"raumId": a.raum})
	c.itemIn("Topf", c.lagerortAnlegen(map[string]any{"name": "Schrank", "raumId": kueche}))
	c.itemAnlegen("Kiste", []int64{c.ich()}, testJPEG(t))

	for _, fall := range []struct {
		abfrage string
		namen   []string
	}{
		{fmt.Sprintf("?raum=%d", a.raum), []string{"Schere", "Stift", "Stuhl"}},
		{fmt.Sprintf("?raum=%d", kueche), []string{"Topf"}},
		{fmt.Sprintf("?lagerort=%d", a.schreibtisch), []string{"Schere", "Stift"}},
		{fmt.Sprintf("?lagerort=%d", a.schublade), []string{"Schere"}},
		{"?ohneLagerort=1", []string{"Kiste", "Stuhl"}},
		{fmt.Sprintf("?ohneLagerort=1&raum=%d", a.raum), []string{"Stuhl"}},
		{fmt.Sprintf("?q=sch&raum=%d", a.raum), []string{"Schere"}},
	} {
		got := c.namenInListe(fall.abfrage)
		slices.Sort(got)
		if !slices.Equal(got, fall.namen) {
			t.Errorf("%s: %v, erwartet %v", fall.abfrage, got, fall.namen)
		}
	}
}

func TestPlaetzeAlsDurchsuchbarePfadliste(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	c.raumAnlegen(c.wohnungAnlegen("Neue Wohnung"), "Küche")

	res := c.mustStatus(http.StatusOK, "GET", "/api/plaetze", nil)
	var got []string
	for _, p := range res.Body["plaetze"].([]any) {
		p := p.(map[string]any)
		got = append(got, fmt.Sprintf("%s:%v %s", p["art"], p["id"], p["pfad"]))
	}
	want := []string{
		fmt.Sprintf("raum:%d Altbau › Arbeitszimmer", a.raum),
		fmt.Sprintf("lagerort:%d Altbau › Arbeitszimmer › Schreibtisch", a.schreibtisch),
		fmt.Sprintf("lagerort:%d Altbau › Arbeitszimmer › Schreibtisch › obere Schublade", a.schublade),
		"raum:2 Neue Wohnung › Küche",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Plätze:\n%v\nerwartet:\n%v", got, want)
	}
}
