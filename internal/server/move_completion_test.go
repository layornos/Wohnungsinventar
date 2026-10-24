package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestCompletionPutsItemsIntoTheirTargetRooms(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box, number := c.createBox("Büro")
	punch := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(punch)+"/move", map[string]any{"boxId": box, "targetRoomId": m.office})
	pot := c.itemIn("Topf", m.desk)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"targetRoomId": m.kitchen})

	res := c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{})
	if res.Body["active"] != false || res.Body["completedAt"] == nil {
		t.Fatalf("completion: %s", res.Raw)
	}
	for item, room := range map[*response]string{&punch: "Neue Wohnung › Büro", &pot: "Neue Wohnung › Küche"} {
		got := c.get(itemPath(*item))
		if p := pathText(t, got.Body["placement"]); p != room {
			t.Errorf("%s: %q, want %q", item.Body["name"], p, room)
		}
		if got.Body["move"] != nil {
			t.Errorf("%s: expected in no box any more after completion: %s", item.Body["name"], got.Raw)
		}
	}
	if got := c.namesInList(fmt.Sprintf("?unstored=1&room=%d", m.office)); !slices.Equal(got, []string{"Locher"}) {
		t.Fatalf("unstored in the office: %v", got)
	}
	if res := c.mustStatus(http.StatusOK, "GET", "/api/move", nil); res.Body["move"] != nil {
		t.Fatalf("no move should be active any more: %s", res.Raw)
	}

	// The completed move stays visible, including the packing list of its boxes.
	list := c.mustStatus(http.StatusOK, "GET", "/api/moves", nil).Body["moves"].([]any)
	if len(list) != 1 {
		t.Fatalf("completed moves: %v", list)
	}
	pack := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/boxes/%d", box), nil)
	if got := lines(t, pack.Body["items"].([]any)); !slices.Equal(got, []string{fmt.Sprintf("Locher %d Arbeitszimmer→Büro", number)}) {
		t.Fatalf("packing list in the history: %v", got)
	}
}

func TestWithoutConfirmationItemsWithoutTargetRoomPreventCompletion(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	forgotten := c.itemIn("Tacker", m.drawer)

	res := c.mustStatus(http.StatusConflict, "POST", "/api/move/completion", map[string]any{})
	if res.errorCode() != "items_without_target_room" || !slices.Equal(namesOf(res), []string{"Tacker"}) {
		t.Fatalf("without confirmation: %s", res.Raw)
	}
	c.mustStatus(http.StatusOK, "GET", "/api/move", nil)

	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{"confirmed": true})
	if p := pathText(t, c.get(itemPath(forgotten)).Body["placement"]); p != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("item without target room should stay where it was: %q", p)
	}
}

func TestCompletedMoveCannotBeChanged(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box, _ := c.createBox("Büro")
	item := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(item)+"/move", map[string]any{"boxId": box, "targetRoomId": m.office})
	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{})

	for _, tc := range []struct {
		method, path string
		body         map[string]any
		code         string
	}{
		{"PUT", fmt.Sprintf("/api/boxes/%d", box), map[string]any{"label": "x"}, "move_completed"},
		{"DELETE", fmt.Sprintf("/api/boxes/%d", box), nil, "move_completed"},
		{"PATCH", itemPath(item) + "/move", map[string]any{"targetRoomId": m.kitchen}, "no_move"},
		{"POST", "/api/move/boxes", map[string]any{}, "no_move"},
		{"POST", "/api/move/completion", map[string]any{}, "no_move"},
	} {
		if r := c.do(tc.method, tc.path, tc.body); r.Status != http.StatusConflict || r.errorCode() != tc.code {
			t.Errorf("%s %s: status %d, error code %q", tc.method, tc.path, r.Status, r.errorCode())
		}
	}

	// A new move is possible afterwards, e.g. back again.
	c.mustStatus(http.StatusCreated, "POST", "/api/move", map[string]any{"oldApartmentId": m.newApartment, "newApartmentId": m.apartment})
}

func TestRoomInAMoveCannotBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PATCH", itemPath(item)+"/move", map[string]any{"targetRoomId": m.kitchen})

	r := c.do("DELETE", fmt.Sprintf("/api/rooms/%d", m.kitchen), nil)
	if r.Status != http.StatusConflict || r.errorCode() != "room_in_move" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
}
