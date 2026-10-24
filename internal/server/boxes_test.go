package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func (c *apiClient) createBox(label string) (id int64, number int) {
	c.t.Helper()
	res := c.mustStatus(http.StatusCreated, "POST", "/api/move/boxes", map[string]any{"label": label})
	return int64(res.Body["id"].(float64)), int(res.Body["number"].(float64))
}

func TestBoxNumbersAreSequentialAndNeverReused(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	c.createMove()

	_, one := c.createBox("Bücher")
	twoID, two := c.createBox("")
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/boxes/%d", twoID), nil)
	_, three := c.createBox("Küche")
	if got := []int{one, two, three}; !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("numbers %v", got)
	}

	res := c.mustStatus(http.StatusOK, "GET", "/api/move", nil)
	var boxes []string
	for _, b := range res.Body["move"].(map[string]any)["boxes"].([]any) {
		b := b.(map[string]any)
		boxes = append(boxes, fmt.Sprintf("%v %v", b["number"], b["label"]))
	}
	if !slices.Equal(boxes, []string{"1 Bücher", "3 Küche"}) {
		t.Fatalf("boxes %v", boxes)
	}
}

func TestRenameBoxAndDeleteOnlyWhenEmpty(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	c.createMove()
	box, _ := c.createBox("Bücher")

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/boxes/%d", box), map[string]any{"label": "Bücher Wohnzimmer"})
	if res.Body["label"] != "Bücher Wohnzimmer" {
		t.Fatalf("renamed: %s", res.Raw)
	}

	item := c.createItem("Roman", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PATCH", itemPath(item)+"/move", map[string]any{"boxId": box})
	r := c.do("DELETE", fmt.Sprintf("/api/boxes/%d", box), nil)
	if r.Status != http.StatusConflict || r.errorCode() != "box_not_empty" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}

	// When the item is deleted, its move details go away and the box is empty.
	c.mustStatus(http.StatusNoContent, "DELETE", itemPath(item), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/boxes/%d", box), nil)
}

func TestThereAreNoBoxesWithoutActiveMove(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	r := c.do("POST", "/api/move/boxes", map[string]any{"label": "x"})
	if r.Status != http.StatusConflict || r.errorCode() != "no_move" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
}
