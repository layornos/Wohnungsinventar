package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// lines describes list items as "name box originRoom→targetRoom" and sorts them.
func lines(t *testing.T, items []any) []string {
	t.Helper()
	var out []string
	for _, i := range items {
		item := response{Body: i.(map[string]any)}
		out = append(out, fmt.Sprintf("%s %v %v→%v", item.Body["name"], detail(t, item, "box"), detail(t, item, "originRoom"), detail(t, item, "targetRoom")))
	}
	slices.Sort(out)
	return out
}

func TestPackingListOfABoxByNumber(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	c.createBox("leer")
	box, number := c.createBox("Büro")
	punch := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(punch)+"/move", map[string]any{"boxId": box, "targetRoomId": m.office})

	// Newly captured straight into the box, as in packing mode.
	added := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", "Tacker"}, {"owner", fmt.Sprint(c.me())}, {"room", fmt.Sprint(m.room)}, {"box", fmt.Sprint(box)}},
		photos: [][]byte{testJPEG(t)},
	})
	if added.Status != http.StatusCreated || detail(t, added, "box") != number || detail(t, added, "originRoom") != "Arbeitszimmer" {
		t.Fatalf("captured into the box: status %d: %s", added.Status, added.Raw)
	}

	res := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/move/boxes/%d", number), nil)
	if res.Body["box"].(map[string]any)["label"] != "Büro" {
		t.Fatalf("box: %s", res.Raw)
	}
	want := []string{"Locher 2 Arbeitszimmer→Büro", "Tacker 2 Arbeitszimmer→<nil>"}
	if got := lines(t, res.Body["items"].([]any)); !slices.Equal(got, want) {
		t.Fatalf("packing list:\n%v\nwant:\n%v", got, want)
	}
	for _, item := range res.Body["items"].([]any) {
		if !strings.HasSuffix(item.(map[string]any)["thumbnailUrl"].(string), "/thumbnail") {
			t.Errorf("without thumbnail: %v", item)
		}
	}
	c.mustStatus(http.StatusNotFound, "GET", "/api/move/boxes/99", nil)
}

func TestItemListShowsBoxAndFiltersByBoxAndOriginRoom(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	hall := c.createRoom(m.apartment, "Flur")
	box, number := c.createBox("")
	packed := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(packed)+"/move", map[string]any{"boxId": box})
	c.itemIn("Tacker", m.desk) // no move details yet, but in the study
	umbrella := c.createItem("Schirm", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(umbrella)+"/placement", map[string]any{"roomId": hall})

	if got := lines(t, c.mustStatus(http.StatusOK, "GET", "/api/items?q=locher", nil).Body["items"].([]any)); !slices.Equal(got, []string{fmt.Sprintf("Locher %d Arbeitszimmer→<nil>", number)}) {
		t.Fatalf("search does not show the box: %v", got)
	}
	for _, tc := range []struct {
		query string
		names []string
	}{
		{fmt.Sprintf("?box=%d", box), []string{"Locher"}},
		{fmt.Sprintf("?originRoom=%d", m.room), []string{"Locher", "Tacker"}},
		{fmt.Sprintf("?originRoom=%d", hall), []string{"Schirm"}},
	} {
		got := c.namesInList(tc.query)
		slices.Sort(got)
		if !slices.Equal(got, tc.names) {
			t.Errorf("%s: %v, want %v", tc.query, got, tc.names)
		}
	}
}

func TestOverviewPerTargetRoomAndWithoutTargetRoom(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box, number := c.createBox("Büro")
	punch := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(punch)+"/move", map[string]any{"boxId": box, "targetRoomId": m.office})
	chair := c.createItem("Stuhl", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(chair)+"/placement", map[string]any{"roomId": m.room})
	c.mustStatus(http.StatusOK, "PATCH", itemPath(chair)+"/move", map[string]any{"targetRoomId": m.office})
	c.itemIn("Tacker", m.desk)
	c.createItem("Ohne Platz", []int64{c.me()}, testJPEG(t)) // belongs to no apartment

	res := c.mustStatus(http.StatusOK, "GET", "/api/move/target-rooms", nil)
	var got []string
	for _, tr := range res.Body["targetRooms"].([]any) {
		tr := tr.(map[string]any)
		var boxes []string
		for _, b := range tr["boxes"].([]any) {
			boxes = append(boxes, fmt.Sprint(b.(map[string]any)["number"]))
		}
		got = append(got, fmt.Sprintf("%s: boxes %v, items %v", tr["room"].(map[string]any)["name"], boxes, namesOf(response{Body: tr})))
	}
	want := []string{fmt.Sprintf("Büro: boxes [%d], items [Locher Stuhl]", number), "Küche: boxes [], items []"}
	if !slices.Equal(got, want) {
		t.Fatalf("overview:\n%v\nwant:\n%v", got, want)
	}

	if got := c.namesInList("?withoutTargetRoom=1"); !slices.Equal(got, []string{"Tacker"}) {
		t.Fatalf("without target room: %v", got)
	}
}
