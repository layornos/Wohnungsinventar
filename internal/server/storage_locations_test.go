package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func (c *apiClient) createStorageLocation(body map[string]any) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/storage-locations", body).Body["id"].(float64))
}

// pathText turns a placement or a storage location path into "apartment › room › …".
func pathText(t *testing.T, v any) string {
	t.Helper()
	if v == nil {
		return ""
	}
	p := v.(map[string]any)
	parts := []string{p["apartment"].(map[string]any)["name"].(string), p["room"].(map[string]any)["name"].(string)}
	for _, l := range p["storageLocations"].([]any) {
		parts = append(parts, l.(map[string]any)["name"].(string))
	}
	return strings.Join(parts, " › ")
}

// study is Altbau › Arbeitszimmer › Schreibtisch › obere Schublade.
type study struct {
	apartment, room, desk, drawer int64
}

func (c *apiClient) createStudy() study {
	c.t.Helper()
	var s study
	s.apartment = c.createApartment("Altbau")
	s.room = c.createRoom(s.apartment, "Arbeitszimmer")
	s.desk = c.createStorageLocation(map[string]any{"name": "Schreibtisch", "roomId": s.room})
	s.drawer = c.createStorageLocation(map[string]any{"name": "obere Schublade", "parentId": s.desk})
	return s
}

func TestStorageLocationsFormATreeWithPath(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	c.createStorageLocation(map[string]any{"name": "Regal", "roomId": s.room})
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d", s.drawer), map[string]any{"name": "Schublade oben"})

	room := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/rooms/%d", s.room), nil)
	tree, _ := json.Marshal(room.Body["storageLocations"])
	want := fmt.Sprintf(`[{"children":[],"furniture":null,"id":%d,"name":"Regal"},{"children":[{"children":[],"furniture":null,"id":%d,"name":"Schublade oben"}],"furniture":null,"id":%d,"name":"Schreibtisch"}]`,
		s.desk+2, s.drawer, s.desk)
	if string(tree) != want {
		t.Fatalf("tree:\n%s\nwant:\n%s", tree, want)
	}

	l := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/storage-locations/%d", s.desk), nil)
	if got := pathText(t, l.Body["path"]); got != "Altbau › Arbeitszimmer › Schreibtisch" {
		t.Errorf("path %q", got)
	}
	if children := l.Body["children"].([]any); len(children) != 1 || children[0].(map[string]any)["name"] != "Schublade oben" {
		t.Errorf("children: %s", l.Raw)
	}

	for _, tc := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"name": "X"}, "invalid_place"},
		{map[string]any{"name": "X", "roomId": s.room, "parentId": s.desk}, "invalid_place"},
		{map[string]any{"name": "X", "roomId": 999}, "unknown_place"},
		{map[string]any{"name": "X", "parentId": 999}, "unknown_place"},
		{map[string]any{"name": " ", "roomId": s.room}, "name_missing"},
	} {
		if r := c.do("POST", "/api/storage-locations", tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
}

func TestPlaceItem(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()

	scissors := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", "Schere"}, {"owner", fmt.Sprint(c.me())}, {"storageLocation", fmt.Sprint(s.drawer)}},
		photos: [][]byte{testJPEG(t)},
	})
	if got := pathText(t, scissors.Body["placement"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("placement on creation: %q", got)
	}

	res := c.mustStatus(http.StatusOK, "PUT", itemPath(scissors)+"/placement", map[string]any{"roomId": s.room})
	if got := pathText(t, res.Body["placement"]); got != "Altbau › Arbeitszimmer" {
		t.Fatalf("directly in the room: %q", got)
	}
	res = c.mustStatus(http.StatusOK, "PUT", itemPath(scissors)+"/placement", map[string]any{})
	if res.Body["placement"] != nil {
		t.Fatalf("without placement: %s", res.Raw)
	}
	if res := c.get(itemPath(scissors)); res.Body["placement"] != nil {
		t.Fatalf("fetched: %s", res.Raw)
	}

	for _, tc := range []struct {
		body   map[string]any
		status int
		code   string
	}{
		{map[string]any{"roomId": s.room, "storageLocationId": s.drawer}, http.StatusBadRequest, "invalid_place"},
		{map[string]any{"storageLocationId": 999}, http.StatusBadRequest, "unknown_place"},
		{map[string]any{"roomId": 999}, http.StatusBadRequest, "unknown_place"},
	} {
		if r := c.do("PUT", itemPath(scissors)+"/placement", tc.body); r.Status != tc.status || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
	unknown := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", "Stift"}, {"owner", fmt.Sprint(c.me())}, {"room", "999"}},
		photos: [][]byte{testJPEG(t)},
	})
	if unknown.errorCode() != "unknown_place" {
		t.Errorf("unknown room on creation: status %d, error code %q", unknown.Status, unknown.errorCode())
	}
	e.noOrphanedPhotos(1)
}

func TestFurnitureIsNotInsideItsOwnStorageLocation(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	deskID := desk.Body["id"]

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", s.desk), map[string]any{"itemId": deskID})
	res := c.get(itemPath(desk))
	if f := res.Body["furnitureStorageLocations"].([]any); len(f) != 1 || f[0].(map[string]any)["name"] != "Schreibtisch" {
		t.Fatalf("storage locations of the furniture: %s", res.Raw)
	}
	room := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/rooms/%d", s.room), nil)
	if f := room.Body["storageLocations"].([]any)[0].(map[string]any)["furniture"]; f.(map[string]any)["name"] != "Schreibtisch" {
		t.Fatalf("furniture in the tree: %s", room.Raw)
	}

	// The desk cannot be inside its own drawer.
	r := c.do("PUT", itemPath(desk)+"/placement", map[string]any{"storageLocationId": s.drawer})
	if r.Status != http.StatusConflict || r.errorCode() != "furniture_inside_itself" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
	c.mustStatus(http.StatusOK, "PUT", itemPath(desk)+"/placement", map[string]any{"roomId": s.room})

	// The other way round: if the lamp is in the drawer, the drawer cannot belong to the lamp.
	lamp := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", "Lampe"}, {"owner", fmt.Sprint(c.me())}, {"storageLocation", fmt.Sprint(s.drawer)}},
		photos: [][]byte{testJPEG(t)},
	})
	r = c.do("PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", s.desk), map[string]any{"itemId": lamp.Body["id"]})
	if r.Status != http.StatusConflict || r.errorCode() != "furniture_inside_itself" {
		t.Fatalf("assignment: status %d, error code %q", r.Status, r.errorCode())
	}

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", s.desk), map[string]any{"itemId": nil})
	if f := c.get(itemPath(desk)).Body["furnitureStorageLocations"].([]any); len(f) != 0 {
		t.Fatalf("after removing the assignment: %v", f)
	}
}

func TestMovingAShelfTakesItsContentsAlong(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	hall := c.createRoom(s.apartment, "Flur")
	shelf := c.createStorageLocation(map[string]any{"name": "Regal", "roomId": s.room})
	compartment := c.createStorageLocation(map[string]any{"name": "Fach 1", "parentId": shelf})
	book := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", "Buch"}, {"owner", fmt.Sprint(c.me())}, {"storageLocation", fmt.Sprint(compartment)}},
		photos: [][]byte{testJPEG(t)},
	})

	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/parent", shelf), map[string]any{"roomId": hall})
	if got := pathText(t, c.get(itemPath(book)).Body["placement"]); got != "Altbau › Flur › Regal › Fach 1" {
		t.Fatalf("after moving into the hall: %q", got)
	}

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/parent", shelf), map[string]any{"parentId": s.desk})
	if got := pathText(t, res.Body["path"]); got != "Altbau › Arbeitszimmer › Schreibtisch › Regal" {
		t.Fatalf("path of the shelf: %q", got)
	}
	if got := pathText(t, c.get(itemPath(book)).Body["placement"]); got != "Altbau › Arbeitszimmer › Schreibtisch › Regal › Fach 1" {
		t.Fatalf("after moving into the desk: %q", got)
	}
}

func TestStorageLocationCannotBeMovedIntoItself(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	divider := c.createStorageLocation(map[string]any{"name": "Trenner", "parentId": s.drawer})

	for _, target := range []int64{s.desk, s.drawer, divider} {
		r := c.do("PUT", fmt.Sprintf("/api/storage-locations/%d/parent", s.desk), map[string]any{"parentId": target})
		if r.Status != http.StatusConflict || r.errorCode() != "cycle" {
			t.Errorf("into %d: status %d, error code %q", target, r.Status, r.errorCode())
		}
	}
	for _, tc := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{}, "invalid_place"},
		{map[string]any{"roomId": s.room, "parentId": s.drawer}, "invalid_place"},
		{map[string]any{"roomId": 999}, "unknown_place"},
	} {
		if r := c.do("PUT", fmt.Sprintf("/api/storage-locations/%d/parent", divider), tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
	l := c.get(fmt.Sprintf("/api/storage-locations/%d", divider))
	if got := pathText(t, l.Body["path"]); got != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade › Trenner" {
		t.Fatalf("path expected unchanged: %q", got)
	}
}

func TestRoomCanOnlyBeDeletedWhenEmpty(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.createApartment("Altbau")
	kitchen := c.createRoom(a, "Küche")
	bath := c.createRoom(a, "Bad")
	shelf := c.createStorageLocation(map[string]any{"name": "Regal", "roomId": kitchen})
	sofa := c.createItem("Sofa", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(sofa)+"/placement", map[string]any{"roomId": bath})

	for _, room := range []int64{kitchen, bath} {
		r := c.do("DELETE", fmt.Sprintf("/api/rooms/%d", room), nil)
		if r.Status != http.StatusConflict || r.errorCode() != "room_not_empty" {
			t.Errorf("room %d: status %d, error code %q", room, r.Status, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/storage-locations/%d", shelf), nil)
	c.mustStatus(http.StatusOK, "PUT", itemPath(sofa)+"/placement", map[string]any{})
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/rooms/%d", kitchen), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/rooms/%d", bath), nil)
}

func TestStorageLocationNamesItsSubtree(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	divider := c.createStorageLocation(map[string]any{"name": "Trenner", "parentId": s.drawer})
	// Same name as the drawer, but not nested in the desk.
	c.createStorageLocation(map[string]any{"name": "obere Schublade", "roomId": s.room})

	l := c.get(fmt.Sprintf("/api/storage-locations/%d", s.desk))
	var ids []int64
	for _, id := range l.Body["subtree"].([]any) {
		ids = append(ids, int64(id.(float64)))
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []int64{s.desk, s.drawer, divider}) {
		t.Fatalf("subtree %v", ids)
	}
}
