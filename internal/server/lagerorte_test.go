package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func (c *apiClient) lagerortAnlegen(body map[string]any) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/lagerorte", body).Body["id"].(float64))
}

// pfadText macht aus einer Platzierung oder einem Lagerort-Pfad „Wohnung › Raum › …“.
func pfadText(t *testing.T, v any) string {
	t.Helper()
	if v == nil {
		return ""
	}
	p := v.(map[string]any)
	teile := []string{p["wohnung"].(map[string]any)["name"].(string), p["raum"].(map[string]any)["name"].(string)}
	for _, l := range p["lagerorte"].([]any) {
		teile = append(teile, l.(map[string]any)["name"].(string))
	}
	return strings.Join(teile, " › ")
}

// arbeitszimmer legt Altbau › Arbeitszimmer › Schreibtisch › obere Schublade an.
type arbeitszimmer struct {
	wohnung, raum, schreibtisch, schublade int64
}

func (c *apiClient) arbeitszimmerAnlegen() arbeitszimmer {
	c.t.Helper()
	var a arbeitszimmer
	a.wohnung = c.wohnungAnlegen("Altbau")
	a.raum = c.raumAnlegen(a.wohnung, "Arbeitszimmer")
	a.schreibtisch = c.lagerortAnlegen(map[string]any{"name": "Schreibtisch", "raumId": a.raum})
	a.schublade = c.lagerortAnlegen(map[string]any{"name": "obere Schublade", "elternId": a.schreibtisch})
	return a
}

func TestLagerorteBildenEinenBaumMitPfad(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	c.lagerortAnlegen(map[string]any{"name": "Regal", "raumId": a.raum})
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d", a.schublade), map[string]any{"name": "Schublade oben"})

	raum := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/raeume/%d", a.raum), nil)
	baum, _ := json.Marshal(raum.Body["lagerorte"])
	want := fmt.Sprintf(`[{"id":%d,"moebel":null,"name":"Regal","unter":[]},{"id":%d,"moebel":null,"name":"Schreibtisch","unter":[{"id":%d,"moebel":null,"name":"Schublade oben","unter":[]}]}]`,
		a.schreibtisch+2, a.schreibtisch, a.schublade)
	if string(baum) != want {
		t.Fatalf("Baum:\n%s\nerwartet:\n%s", baum, want)
	}

	l := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/lagerorte/%d", a.schreibtisch), nil)
	if got := pfadText(t, l.Body["pfad"]); got != "Altbau › Arbeitszimmer › Schreibtisch" {
		t.Errorf("Pfad %q", got)
	}
	if unter := l.Body["unter"].([]any); len(unter) != 1 || unter[0].(map[string]any)["name"] != "Schublade oben" {
		t.Errorf("Unterlagerorte: %s", l.Raw)
	}

	for _, fall := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"name": "X"}, "ungueltiger_platz"},
		{map[string]any{"name": "X", "raumId": a.raum, "elternId": a.schreibtisch}, "ungueltiger_platz"},
		{map[string]any{"name": "X", "raumId": 999}, "unbekannter_platz"},
		{map[string]any{"name": "X", "elternId": 999}, "unbekannter_platz"},
		{map[string]any{"name": " ", "raumId": a.raum}, "name_fehlt"},
	} {
		if r := c.do("POST", "/api/lagerorte", fall.body); r.Status != http.StatusBadRequest || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
}

func TestItemPlatzieren(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()

	schere := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", "Schere"}, {"besitzer", fmt.Sprint(c.ich())}, {"lagerort", fmt.Sprint(a.schublade)}},
		fotos:  [][]byte{testJPEG(t)},
	})
	if got := pfadText(t, schere.Body["platzierung"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("Platzierung beim Anlegen: %q", got)
	}

	res := c.mustStatus(http.StatusOK, "PUT", itemPfad(schere)+"/platzierung", map[string]any{"raumId": a.raum})
	if got := pfadText(t, res.Body["platzierung"]); got != "Altbau › Arbeitszimmer" {
		t.Fatalf("direkt im Raum: %q", got)
	}
	res = c.mustStatus(http.StatusOK, "PUT", itemPfad(schere)+"/platzierung", map[string]any{})
	if res.Body["platzierung"] != nil {
		t.Fatalf("ohne Platzierung: %s", res.Raw)
	}
	if res := c.get(itemPfad(schere)); res.Body["platzierung"] != nil {
		t.Fatalf("abgerufen: %s", res.Raw)
	}

	for _, fall := range []struct {
		body   map[string]any
		status int
		code   string
	}{
		{map[string]any{"raumId": a.raum, "lagerortId": a.schublade}, http.StatusBadRequest, "ungueltiger_platz"},
		{map[string]any{"lagerortId": 999}, http.StatusBadRequest, "unbekannter_platz"},
		{map[string]any{"raumId": 999}, http.StatusBadRequest, "unbekannter_platz"},
	} {
		if r := c.do("PUT", itemPfad(schere)+"/platzierung", fall.body); r.Status != fall.status || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
	unbekannt := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", "Stift"}, {"besitzer", fmt.Sprint(c.ich())}, {"raum", "999"}},
		fotos:  [][]byte{testJPEG(t)},
	})
	if unbekannt.errorCode() != "unbekannter_platz" {
		t.Errorf("unbekannter Raum beim Anlegen: Status %d, Fehlercode %q", unbekannt.Status, unbekannt.errorCode())
	}
	e.keineVerwaistenFotos(1)
}

func TestMoebelstueckLiegtNichtInEigenemLagerort(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	tisch := c.itemAnlegen("Schreibtisch", []int64{c.ich()}, testJPEG(t))
	tischID := tisch.Body["id"]

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d/moebel", a.schreibtisch), map[string]any{"itemId": tischID})
	res := c.get(itemPfad(tisch))
	if m := res.Body["moebelLagerorte"].([]any); len(m) != 1 || m[0].(map[string]any)["name"] != "Schreibtisch" {
		t.Fatalf("Lagerorte des Möbelstücks: %s", res.Raw)
	}
	raum := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/raeume/%d", a.raum), nil)
	if m := raum.Body["lagerorte"].([]any)[0].(map[string]any)["moebel"]; m.(map[string]any)["name"] != "Schreibtisch" {
		t.Fatalf("Möbelstück im Baum: %s", raum.Raw)
	}

	// Der Schreibtisch kann nicht in seiner eigenen Schublade liegen.
	r := c.do("PUT", itemPfad(tisch)+"/platzierung", map[string]any{"lagerortId": a.schublade})
	if r.Status != http.StatusConflict || r.errorCode() != "moebel_in_sich_selbst" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
	c.mustStatus(http.StatusOK, "PUT", itemPfad(tisch)+"/platzierung", map[string]any{"raumId": a.raum})

	// Umgekehrt: Liegt die Lampe in der Schublade, kann die Schublade nicht der Lampe gehören.
	lampe := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", "Lampe"}, {"besitzer", fmt.Sprint(c.ich())}, {"lagerort", fmt.Sprint(a.schublade)}},
		fotos:  [][]byte{testJPEG(t)},
	})
	r = c.do("PUT", fmt.Sprintf("/api/lagerorte/%d/moebel", a.schreibtisch), map[string]any{"itemId": lampe.Body["id"]})
	if r.Status != http.StatusConflict || r.errorCode() != "moebel_in_sich_selbst" {
		t.Fatalf("Zuordnung: Status %d, Fehlercode %q", r.Status, r.errorCode())
	}

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d/moebel", a.schreibtisch), map[string]any{"itemId": nil})
	if m := c.get(itemPfad(tisch)).Body["moebelLagerorte"].([]any); len(m) != 0 {
		t.Fatalf("nach dem Lösen: %v", m)
	}
}

func TestRegalVerschiebenNimmtSeinenInhaltMit(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	flur := c.raumAnlegen(a.wohnung, "Flur")
	regal := c.lagerortAnlegen(map[string]any{"name": "Regal", "raumId": a.raum})
	fach := c.lagerortAnlegen(map[string]any{"name": "Fach 1", "elternId": regal})
	buch := c.multipart("POST", "/api/items", formular{
		felder: [][2]string{{"name", "Buch"}, {"besitzer", fmt.Sprint(c.ich())}, {"lagerort", fmt.Sprint(fach)}},
		fotos:  [][]byte{testJPEG(t)},
	})

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d/eltern", regal), map[string]any{"raumId": flur})
	if got := pfadText(t, c.get(itemPfad(buch)).Body["platzierung"]); got != "Altbau › Flur › Regal › Fach 1" {
		t.Fatalf("nach dem Verschieben in den Flur: %q", got)
	}

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/lagerorte/%d/eltern", regal), map[string]any{"elternId": a.schreibtisch})
	if got := pfadText(t, res.Body["pfad"]); got != "Altbau › Arbeitszimmer › Schreibtisch › Regal" {
		t.Fatalf("Pfad des Regals: %q", got)
	}
	if got := pfadText(t, c.get(itemPfad(buch)).Body["platzierung"]); got != "Altbau › Arbeitszimmer › Schreibtisch › Regal › Fach 1" {
		t.Fatalf("nach dem Verschieben in den Schreibtisch: %q", got)
	}
}

func TestLagerortLaesstSichNichtInSichSelbstVerschieben(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	trenner := c.lagerortAnlegen(map[string]any{"name": "Trenner", "elternId": a.schublade})

	for _, ziel := range []int64{a.schreibtisch, a.schublade, trenner} {
		r := c.do("PUT", fmt.Sprintf("/api/lagerorte/%d/eltern", a.schreibtisch), map[string]any{"elternId": ziel})
		if r.Status != http.StatusConflict || r.errorCode() != "zyklus" {
			t.Errorf("in %d: Status %d, Fehlercode %q", ziel, r.Status, r.errorCode())
		}
	}
	for _, fall := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{}, "ungueltiger_platz"},
		{map[string]any{"raumId": a.raum, "elternId": a.schublade}, "ungueltiger_platz"},
		{map[string]any{"raumId": 999}, "unbekannter_platz"},
	} {
		if r := c.do("PUT", fmt.Sprintf("/api/lagerorte/%d/eltern", trenner), fall.body); r.Status != http.StatusBadRequest || r.errorCode() != fall.code {
			t.Errorf("%v: Status %d, Fehlercode %q", fall.body, r.Status, r.errorCode())
		}
	}
	l := c.get(fmt.Sprintf("/api/lagerorte/%d", trenner))
	if got := pfadText(t, l.Body["pfad"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade › Trenner" {
		t.Fatalf("Pfad unverändert erwartet: %q", got)
	}
}

func TestRaumNurLeerLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	w := c.wohnungAnlegen("Altbau")
	kueche := c.raumAnlegen(w, "Küche")
	bad := c.raumAnlegen(w, "Bad")
	regal := c.lagerortAnlegen(map[string]any{"name": "Regal", "raumId": kueche})
	sofa := c.itemAnlegen("Sofa", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPfad(sofa)+"/platzierung", map[string]any{"raumId": bad})

	for _, raum := range []int64{kueche, bad} {
		r := c.do("DELETE", fmt.Sprintf("/api/raeume/%d", raum), nil)
		if r.Status != http.StatusConflict || r.errorCode() != "raum_nicht_leer" {
			t.Errorf("Raum %d: Status %d, Fehlercode %q", raum, r.Status, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/lagerorte/%d", regal), nil)
	c.mustStatus(http.StatusOK, "PUT", itemPfad(sofa)+"/platzierung", map[string]any{})
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/raeume/%d", kueche), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/raeume/%d", bad), nil)
}

func TestLagerortNenntSeinenUnterbaum(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.arbeitszimmerAnlegen()
	trenner := c.lagerortAnlegen(map[string]any{"name": "Trenner", "elternId": a.schublade})
	// Gleicher Name wie die Schublade, aber kein Unterlagerort des Schreibtischs.
	c.lagerortAnlegen(map[string]any{"name": "obere Schublade", "raumId": a.raum})

	l := c.get(fmt.Sprintf("/api/lagerorte/%d", a.schreibtisch))
	var ids []int64
	for _, id := range l.Body["unterbaum"].([]any) {
		ids = append(ids, int64(id.(float64)))
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []int64{a.schreibtisch, a.schublade, trenner}) {
		t.Fatalf("Unterbaum %v", ids)
	}
}
