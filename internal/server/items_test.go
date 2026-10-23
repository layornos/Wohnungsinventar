package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

// fotosVon liefert die Fotos einer Item-Darstellung.
func fotosVon(r response) []map[string]any {
	var out []map[string]any
	for _, f := range r.Body["fotos"].([]any) {
		out = append(out, f.(map[string]any))
	}
	return out
}

func besitzerNamen(r response) []string {
	var out []string
	for _, b := range r.Body["besitzer"].([]any) {
		out = append(out, b.(map[string]any)["name"].(string))
	}
	return out
}

func TestItemMitFotoUndBesitzerAnlegen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.personAnlegen("Kim")

	res := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{
			{"name", " Schreibtischlampe "}, {"beschreibung", "Messing"}, {"verbleib", "verkaufen"},
			{"besitzer", fmt.Sprint(c.ich())}, {"besitzer", fmt.Sprint(kim)},
		},
		fotos: [][]byte{testJPEG(t), testPNG(t)},
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("Status %d: %s", res.Status, res.Raw)
	}
	if res.Body["name"] != "Schreibtischlampe" || res.Body["beschreibung"] != "Messing" || res.Body["verbleib"] != "verkaufen" {
		t.Fatalf("Item: %s", res.Raw)
	}
	if got := besitzerNamen(res); len(got) != 2 {
		t.Fatalf("Besitzer %v", got)
	}
	fotos := fotosVon(res)
	if len(fotos) != 2 || fotos[0]["hauptfoto"] != true || fotos[1]["hauptfoto"] != false {
		t.Fatalf("das erste Foto soll Hauptfoto sein: %s", res.Raw)
	}

	abgerufen := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", res.Body["id"]), nil)
	if abgerufen.Body["name"] != "Schreibtischlampe" || len(fotosVon(abgerufen)) != 2 {
		t.Fatalf("abgerufenes Item: %s", abgerufen.Raw)
	}
}

func TestItemOhneVerbleibIstOffen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.itemAnlegen("Schere", []int64{c.ich()}, testJPEG(t))
	if v, da := res.Body["verbleib"]; !da || v != nil {
		t.Fatalf("Verbleib soll null sein: %s", res.Raw)
	}
}

func TestItemAnlegenVerletztKeineInvariante(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	ich := fmt.Sprint(c.ich())

	for _, fall := range []struct {
		name   string
		f      formular
		status int
		code   string
	}{
		{"ohne Foto", formular{felder: [][2]string{{"name", "Lampe"}, {"besitzer", ich}}}, http.StatusBadRequest, "foto_fehlt"},
		{"ohne Besitzer", formular{felder: [][2]string{{"name", "Lampe"}}, fotos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "besitzer_fehlt"},
		{"ohne Namen", formular{felder: [][2]string{{"name", " "}, {"besitzer", ich}}, fotos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "name_fehlt"},
		{"unbekannter Besitzer", formular{felder: [][2]string{{"name", "Lampe"}, {"besitzer", "999"}}, fotos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "unbekannte_person"},
		{"unbekannter Verbleib", formular{felder: [][2]string{{"name", "Lampe"}, {"besitzer", ich}, {"verbleib", "wegwerfen"}}, fotos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "ungueltiger_verbleib"},
	} {
		res := c.multipart("POST", "/api/items", fall.f)
		if res.Status != fall.status || res.errorCode() != fall.code {
			t.Errorf("%s: Status %d, Fehlercode %q, erwartet %d %q", fall.name, res.Status, res.errorCode(), fall.status, fall.code)
		}
	}

	liste := c.mustStatus(http.StatusOK, "GET", "/api/items", nil)
	if n := len(liste.Body["items"].([]any)); n != 0 {
		t.Fatalf("es sollen keine Items entstanden sein, gefunden: %d", n)
	}
	e.keineFotoDateien()
}

func itemPfad(r response) string { return fmt.Sprintf("/api/items/%v", r.Body["id"]) }

func TestItemBearbeitenUndVerbleibZuruecksetzen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))

	res := c.mustStatus(http.StatusOK, "PUT", itemPfad(item), map[string]any{"name": "Stehlampe", "beschreibung": "mit Dimmer", "verbleib": "verschenken"})
	if res.Body["name"] != "Stehlampe" || res.Body["beschreibung"] != "mit Dimmer" || res.Body["verbleib"] != "verschenken" {
		t.Fatalf("bearbeitet: %s", res.Raw)
	}
	if res.Body["geaendertAm"] == item.Body["geaendertAm"] {
		t.Errorf("geaendertAm soll sich ändern")
	}

	res = c.mustStatus(http.StatusOK, "PUT", itemPfad(item), map[string]any{"name": "Stehlampe", "beschreibung": "", "verbleib": nil})
	if v, da := res.Body["verbleib"]; !da || v != nil {
		t.Fatalf("Verbleib soll wieder offen sein: %s", res.Raw)
	}

	for _, fall := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"name": " ", "verbleib": nil}, "name_fehlt"},
		{map[string]any{"name": "Lampe", "verbleib": "wegwerfen"}, "ungueltiger_verbleib"},
	} {
		if r := c.do("PUT", itemPfad(item), fall.body); r.Status != http.StatusBadRequest || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNotFound, "PUT", "/api/items/999", map[string]any{"name": "x"})
}

func TestItemLoeschenEntferntFotoDateien(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t), testPNG(t))
	foto := fotosVon(item)[0]

	c.mustStatus(http.StatusNoContent, "DELETE", itemPfad(item), nil)
	c.mustStatus(http.StatusNotFound, "GET", itemPfad(item), nil)
	if res := c.get(foto["url"].(string)); res.Status != http.StatusNotFound {
		t.Errorf("Foto nach dem Löschen: Status %d", res.Status)
	}
	e.keineFotoDateien()
}

func TestLetzterBesitzerLaesstSichNichtEntfernen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.personAnlegen("Kim")
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("%s/besitzer/%d", itemPfad(item), kim), nil)
	if got := besitzerNamen(res); len(got) != 2 {
		t.Fatalf("Besitzer nach dem Hinzufügen: %v", got)
	}
	// Nochmal hinzufügen ändert nichts.
	res = c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("%s/besitzer/%d", itemPfad(item), kim), nil)
	if got := besitzerNamen(res); len(got) != 2 {
		t.Fatalf("Besitzer nach doppeltem Hinzufügen: %v", got)
	}

	res = c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("%s/besitzer/%d", itemPfad(item), c.ich()), nil)
	if got := besitzerNamen(res); len(got) != 1 || got[0] != "Kim" {
		t.Fatalf("Besitzer nach dem Entfernen: %v", got)
	}
	res = c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("%s/besitzer/%d", itemPfad(item), kim), nil)
	if res.errorCode() != "letzter_besitzer" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}

	res = c.mustStatus(http.StatusBadRequest, "PUT", fmt.Sprintf("%s/besitzer/999", itemPfad(item)), nil)
	if res.errorCode() != "unbekannte_person" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestOhneSitzungSindItemsUndPersonenGesperrt(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))

	fremd := e.client()
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/items"},
		{"POST", "/api/items"},
		{"GET", itemPfad(item)},
		{"PUT", itemPfad(item)},
		{"DELETE", itemPfad(item)},
		{"POST", itemPfad(item) + "/fotos"},
		{"PUT", fmt.Sprintf("%s/besitzer/%d", itemPfad(item), c.ich())},
		{"GET", "/api/personen"},
		{"POST", "/api/personen"},
	} {
		if res := fremd.do(req.method, req.path, nil); res.Status != http.StatusUnauthorized {
			t.Errorf("%s %s: Status %d", req.method, req.path, res.Status)
		}
	}
	c.mustStatus(http.StatusOK, "GET", itemPfad(item), nil)
}
