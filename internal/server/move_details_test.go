package server_test

import (
	"net/http"
	"testing"
)

// detail returns the name of a room or the number of the box from the move details of an item.
func detail(t *testing.T, item response, field string) any {
	t.Helper()
	m, _ := item.Body["move"].(map[string]any)
	if m == nil || m[field] == nil {
		return nil
	}
	v := m[field].(map[string]any)
	if field == "box" {
		return int(v["number"].(float64))
	}
	return v["name"]
}

func TestOriginRoomIsPrefilledFromPlacement(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	scissors := c.itemIn("Schere", m.drawer)

	res := c.mustStatus(http.StatusOK, "PATCH", itemPath(scissors)+"/move", map[string]any{"targetRoomId": m.office})
	if detail(t, res, "originRoom") != "Arbeitszimmer" || detail(t, res, "targetRoom") != "Büro" {
		t.Fatalf("move details: %s", res.Raw)
	}
	// The placement stays unchanged during the move.
	if got := pathText(t, res.Body["placement"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("placement %q", got)
	}

	// The origin room can be changed and reset; details not mentioned stay.
	hall := c.createRoom(m.apartment, "Flur")
	res = c.mustStatus(http.StatusOK, "PATCH", itemPath(scissors)+"/move", map[string]any{"originRoomId": hall})
	if detail(t, res, "originRoom") != "Flur" || detail(t, res, "targetRoom") != "Büro" {
		t.Fatalf("changed: %s", res.Raw)
	}
	res = c.mustStatus(http.StatusOK, "PATCH", itemPath(scissors)+"/move", map[string]any{"originRoomId": nil, "targetRoomId": nil})
	if detail(t, res, "originRoom") != nil || detail(t, res, "targetRoom") != nil {
		t.Fatalf("reset: %s", res.Raw)
	}

	unplaced := c.createItem("Kiste", []int64{c.me()}, testJPEG(t))
	res = c.mustStatus(http.StatusOK, "PATCH", itemPath(unplaced)+"/move", map[string]any{"targetRoomId": m.kitchen})
	if detail(t, res, "originRoom") != nil {
		t.Fatalf("no origin room without placement: %s", res.Raw)
	}
}

func TestMoveDetailsAreChecked(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	for _, tc := range []struct {
		body   map[string]any
		status int
		code   string
	}{
		{map[string]any{"targetRoomId": m.room}, http.StatusBadRequest, "target_room_not_in_new_apartment"},
		{map[string]any{"targetRoomId": 999}, http.StatusBadRequest, "unknown_place"},
		{map[string]any{"originRoomId": 999}, http.StatusBadRequest, "unknown_place"},
		{map[string]any{"boxId": 999}, http.StatusBadRequest, "unknown_box"},
		{map[string]any{"boxId": "one"}, http.StatusBadRequest, "invalid_request"},
	} {
		if r := c.do("PATCH", itemPath(item)+"/move", tc.body); r.Status != tc.status || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
	if res := c.get(itemPath(item)); res.Body["move"] != nil {
		t.Fatalf("rejected changes should not create details: %s", res.Raw)
	}
}

func TestBulkOperationAllOrNothing(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box, number := c.createBox("Büro")
	a := c.itemIn("Locher", m.desk)
	b := c.itemIn("Tacker", m.drawer)

	c.mustStatus(http.StatusNoContent, "POST", "/api/move/bulk", map[string]any{
		"itemIds": []any{a.Body["id"], b.Body["id"]}, "boxId": box, "targetRoomId": m.office,
	})
	for _, item := range []response{a, b} {
		got := c.get(itemPath(item))
		if detail(t, got, "box") != number || detail(t, got, "targetRoom") != "Büro" || detail(t, got, "originRoom") != "Arbeitszimmer" {
			t.Errorf("%s: %s", item.Body["name"], got.Raw)
		}
	}

	// An unknown item prevents the whole change.
	r := c.do("POST", "/api/move/bulk", map[string]any{"itemIds": []any{a.Body["id"], 999}, "targetRoomId": m.kitchen})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unknown_item" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
	if got := detail(t, c.get(itemPath(a)), "targetRoom"); got != "Büro" {
		t.Fatalf("target room after rejected bulk operation: %v", got)
	}
	if r := c.do("POST", "/api/move/bulk", map[string]any{"itemIds": []any{}, "targetRoomId": m.kitchen}); r.Status != http.StatusBadRequest {
		t.Fatalf("without items: status %d", r.Status)
	}
}

func TestThereAreNoMoveDetailsWithoutMove(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	r := c.do("PATCH", itemPath(item)+"/move", map[string]any{"targetRoomId": 1})
	if r.Status != http.StatusConflict || r.errorCode() != "no_move" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
}
