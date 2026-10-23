package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestPersonenAnlegenUmbenennenUndAuflisten(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()

	kind := c.mustStatus(http.StatusCreated, "POST", "/api/personen", map[string]any{"name": " Kim "})
	if kind.Body["name"] != "Kim" || kind.Body["nutzer"] != false {
		t.Fatalf("angelegte Person: %s", kind.Raw)
	}
	id := kind.Body["id"]
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/personen/%v", id), map[string]any{"name": "Kimi"})

	liste := c.mustStatus(http.StatusOK, "GET", "/api/personen", nil)
	personen := liste.Body["personen"].([]any)
	namen := map[string]bool{}
	for _, p := range personen {
		p := p.(map[string]any)
		namen[fmt.Sprintf("%v:%v", p["name"], p["nutzer"])] = true
	}
	if len(personen) != 2 || !namen["Alex:true"] || !namen["Kimi:false"] {
		t.Fatalf("Personen: %s", liste.Raw)
	}
}

func TestPersonOhneNamenWirdAbgelehnt(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.mustStatus(http.StatusBadRequest, "POST", "/api/personen", map[string]any{"name": "  "})
	if res.errorCode() != "name_fehlt" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestPersonOhneItemsIstLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	id := c.personAnlegen("Kim")

	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/personen/%d", id), nil)
	res := c.mustStatus(http.StatusNotFound, "PUT", fmt.Sprintf("/api/personen/%d", id), map[string]any{"name": "Kim"})
	if res.errorCode() != "nicht_gefunden" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestPersonMitNutzerzugangIstNichtLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/personen/%d", c.ich()), nil)
	if res.errorCode() != "hat_nutzerzugang" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
}

func TestLetzterBesitzerEinesItemsIstNichtLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.personAnlegen("Kim")
	item := c.itemAnlegen("Kuscheltier", []int64{kim}, testJPEG(t))

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/personen/%d", kim), nil)
	if res.errorCode() != "letzter_besitzer" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}

	// Mit einem zweiten Besitzer ist Kim nicht mehr letzter Besitzer; beim Löschen entfällt nur die Zuordnung.
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/items/%v/besitzer/%d", item.Body["id"], c.ich()), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/personen/%d", kim), nil)
	nachher := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", item.Body["id"]), nil)
	if got := besitzerNamen(nachher); len(got) != 1 || got[0] != "Alex" {
		t.Fatalf("Besitzer nach dem Löschen: %v", got)
	}
}
