package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

// moveSetup: Altbau › Arbeitszimmer › Schreibtisch › obere Schublade, plus the new apartment
// with Büro and Küche and an active move.
type moveSetup struct {
	study
	newApartment, office, kitchen int64
}

func (c *apiClient) createMove() moveSetup {
	c.t.Helper()
	m := moveSetup{study: c.createStudy()}
	m.newApartment = c.createApartment("Neue Wohnung")
	m.office = c.createRoom(m.newApartment, "Büro")
	m.kitchen = c.createRoom(m.newApartment, "Küche")
	c.mustStatus(http.StatusCreated, "POST", "/api/move", map[string]any{"oldApartmentId": m.apartment, "newApartmentId": m.newApartment})
	return m
}

func TestCreateAndGetMove(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	if res := c.mustStatus(http.StatusOK, "GET", "/api/move", nil); res.Body["move"] != nil {
		t.Fatalf("without move: %s", res.Raw)
	}
	m := c.createMove()

	res := c.mustStatus(http.StatusOK, "GET", "/api/move", nil)
	move := res.Body["move"].(map[string]any)
	if move["oldApartment"].(map[string]any)["name"] != "Altbau" || move["newApartment"].(map[string]any)["name"] != "Neue Wohnung" {
		t.Fatalf("move: %s", res.Raw)
	}

	r := c.do("POST", "/api/move", map[string]any{"oldApartmentId": m.newApartment, "newApartmentId": m.apartment})
	if r.Status != http.StatusConflict || r.errorCode() != "move_in_progress" {
		t.Fatalf("second move: status %d, error code %q", r.Status, r.errorCode())
	}
}

func TestMoveNeedsTwoDifferentApartments(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.createApartment("Altbau")
	for _, tc := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"oldApartmentId": a, "newApartmentId": a}, "same_apartment"},
		{map[string]any{"oldApartmentId": a, "newApartmentId": 999}, "unknown_apartment"},
	} {
		if r := c.do("POST", "/api/move", tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
}

func TestApartmentInAMoveCannotBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	old := c.createApartment("Altbau")
	fresh := c.createApartment("Neue Wohnung")
	c.mustStatus(http.StatusCreated, "POST", "/api/move", map[string]any{"oldApartmentId": old, "newApartmentId": fresh})

	for _, a := range []int64{old, fresh} {
		r := c.do("DELETE", fmt.Sprintf("/api/apartments/%d", a), nil)
		if r.Status != http.StatusConflict || r.errorCode() != "apartment_in_move" {
			t.Errorf("apartment %d: status %d, error code %q", a, r.Status, r.errorCode())
		}
	}
}
