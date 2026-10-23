package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func (c *apiClient) boxAnlegen(bezeichnung string) (id int64, nummer int) {
	c.t.Helper()
	res := c.mustStatus(http.StatusCreated, "POST", "/api/umzug/boxen", map[string]any{"bezeichnung": bezeichnung})
	return int64(res.Body["id"].(float64)), int(res.Body["nummer"].(float64))
}

func TestBoxnummernSindFortlaufendUndWerdenNichtWiederverwendet(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	c.umzugAnlegen()

	_, eins := c.boxAnlegen("Bücher")
	zweiID, zwei := c.boxAnlegen("")
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/boxen/%d", zweiID), nil)
	_, drei := c.boxAnlegen("Küche")
	if got := []int{eins, zwei, drei}; !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Nummern %v", got)
	}

	res := c.mustStatus(http.StatusOK, "GET", "/api/umzug", nil)
	var boxen []string
	for _, b := range res.Body["umzug"].(map[string]any)["boxen"].([]any) {
		b := b.(map[string]any)
		boxen = append(boxen, fmt.Sprintf("%v %v", b["nummer"], b["bezeichnung"]))
	}
	if !slices.Equal(boxen, []string{"1 Bücher", "3 Küche"}) {
		t.Fatalf("Boxen %v", boxen)
	}
}

func TestBoxUmbenennenUndNurLeerLoeschen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	c.umzugAnlegen()
	box, _ := c.boxAnlegen("Bücher")

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/boxen/%d", box), map[string]any{"bezeichnung": "Bücher Wohnzimmer"})
	if res.Body["bezeichnung"] != "Bücher Wohnzimmer" {
		t.Fatalf("umbenannt: %s", res.Raw)
	}

	item := c.itemAnlegen("Roman", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PATCH", itemPfad(item)+"/umzug", map[string]any{"boxId": box})
	r := c.do("DELETE", fmt.Sprintf("/api/boxen/%d", box), nil)
	if r.Status != http.StatusConflict || r.errorCode() != "box_nicht_leer" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}

	// Wird das Item gelöscht, verschwinden seine Umzugsangaben, und die Box ist leer.
	c.mustStatus(http.StatusNoContent, "DELETE", itemPfad(item), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/boxen/%d", box), nil)
}

func TestOhneLaufendenUmzugGibtEsKeineBoxen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	r := c.do("POST", "/api/umzug/boxen", map[string]any{"bezeichnung": "x"})
	if r.Status != http.StatusConflict || r.errorCode() != "kein_umzug" {
		t.Fatalf("Status %d, Fehlercode %q", r.Status, r.errorCode())
	}
}
