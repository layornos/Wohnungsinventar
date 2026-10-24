package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func (c *apiClient) itemIn(name string, storageLocation int64) response {
	c.t.Helper()
	res := c.multipart("POST", "/api/items", form{
		fields: [][2]string{{"name", name}, {"owner", fmt.Sprint(c.me())}, {"storageLocation", fmt.Sprint(storageLocation)}},
		photos: [][]byte{testJPEG(c.t)},
	})
	if res.Status != http.StatusCreated {
		c.t.Fatalf("create item: status %d: %s", res.Status, res.Raw)
	}
	return res
}

func namesOf(r response) []string {
	var names []string
	for _, i := range r.Body["items"].([]any) {
		names = append(names, i.(map[string]any)["name"].(string))
	}
	slices.Sort(names)
	return names
}

func TestDeletingAStorageLocationPutsItsContentsIntoTheRoom(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	divider := c.createStorageLocation(map[string]any{"name": "Trenner", "parentId": s.drawer})
	shelf := c.createStorageLocation(map[string]any{"name": "Regal", "roomId": s.room})
	pen := c.itemIn("Stift", s.desk)
	scissors := c.itemIn("Schere", s.drawer)
	clip := c.itemIn("Büroklammer", divider)
	book := c.itemIn("Buch", shelf)

	preview := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/storage-locations/%d/removal", s.desk), nil)
	if got := namesOf(preview); !slices.Equal(got, []string{"Büroklammer", "Schere", "Stift"}) {
		t.Fatalf("preview: %v", got)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/storage-locations/%d", s.desk), nil)
	for _, item := range []response{pen, scissors, clip} {
		if got := pathText(t, c.get(itemPath(item)).Body["placement"]); got != "Altbau › Arbeitszimmer" {
			t.Errorf("%s: %q", item.Body["name"], got)
		}
	}
	if got := pathText(t, c.get(itemPath(book)).Body["placement"]); got != "Altbau › Arbeitszimmer › Regal" {
		t.Errorf("book: %q", got)
	}
	for _, l := range []int64{s.desk, s.drawer, divider} {
		c.mustStatus(http.StatusNotFound, "GET", fmt.Sprintf("/api/storage-locations/%d", l), nil)
	}
}

func TestDeletingFurnitureRemovesItsStorageLocations(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", s.desk), map[string]any{"itemId": desk.Body["id"]})
	scissors := c.itemIn("Schere", s.drawer)

	preview := c.mustStatus(http.StatusOK, "GET", itemPath(desk)+"/removal", nil)
	if got := namesOf(preview); !slices.Equal(got, []string{"Schere"}) {
		t.Fatalf("preview: %v", got)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", itemPath(desk), nil)
	after := c.get(itemPath(scissors))
	if got := pathText(t, after.Body["placement"]); got != "Altbau › Arbeitszimmer" {
		t.Fatalf("scissors after deleting the desk: %q", got)
	}
	c.mustStatus(http.StatusNotFound, "GET", fmt.Sprintf("/api/storage-locations/%d", s.drawer), nil)
	if got := c.namesInList("?unstored=1"); !slices.Equal(got, []string{"Schere"}) {
		t.Fatalf("unstored: %v", got)
	}
}

func TestRoomAndStorageLocationFiltersIncludeNestedStorageLocations(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	kitchen := c.createRoom(s.apartment, "Küche")
	c.itemIn("Schere", s.drawer)
	c.itemIn("Stift", s.desk)
	chair := c.createItem("Stuhl", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(chair)+"/placement", map[string]any{"roomId": s.room})
	c.itemIn("Topf", c.createStorageLocation(map[string]any{"name": "Schrank", "roomId": kitchen}))
	c.createItem("Kiste", []int64{c.me()}, testJPEG(t))

	for _, tc := range []struct {
		query string
		names []string
	}{
		{fmt.Sprintf("?room=%d", s.room), []string{"Schere", "Stift", "Stuhl"}},
		{fmt.Sprintf("?room=%d", kitchen), []string{"Topf"}},
		{fmt.Sprintf("?storageLocation=%d", s.desk), []string{"Schere", "Stift"}},
		{fmt.Sprintf("?storageLocation=%d", s.drawer), []string{"Schere"}},
		{"?unstored=1", []string{"Kiste", "Stuhl"}},
		{fmt.Sprintf("?unstored=1&room=%d", s.room), []string{"Stuhl"}},
		{fmt.Sprintf("?q=sch&room=%d", s.room), []string{"Schere"}},
	} {
		got := c.namesInList(tc.query)
		slices.Sort(got)
		if !slices.Equal(got, tc.names) {
			t.Errorf("%s: %v, want %v", tc.query, got, tc.names)
		}
	}
}

func TestListNamesThePlaceOfEachItem(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	c.itemIn("Schere", s.drawer)
	chair := c.createItem("Stuhl", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(chair)+"/placement", map[string]any{"roomId": s.room})
	c.createItem("Kiste", []int64{c.me()}, testJPEG(t))

	got := map[string]any{}
	for _, i := range c.get("/api/items").Body["items"].([]any) {
		i := i.(map[string]any)
		got[i["name"].(string)] = i["place"]
	}
	want := map[string]any{"Schere": "obere Schublade", "Stuhl": "Arbeitszimmer", "Kiste": ""}
	for name, place := range want {
		if got[name] != place {
			t.Errorf("%s: place %q, want %q", name, got[name], place)
		}
	}
}

func TestPlacesAsSearchablePathList(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	c.createRoom(c.createApartment("Neue Wohnung"), "Küche")

	res := c.mustStatus(http.StatusOK, "GET", "/api/places", nil)
	var got []string
	for _, p := range res.Body["places"].([]any) {
		p := p.(map[string]any)
		got = append(got, fmt.Sprintf("%s:%v %s", p["kind"], p["id"], p["path"]))
	}
	want := []string{
		fmt.Sprintf("room:%d Altbau › Arbeitszimmer", s.room),
		fmt.Sprintf("storageLocation:%d Altbau › Arbeitszimmer › Schreibtisch", s.desk),
		fmt.Sprintf("storageLocation:%d Altbau › Arbeitszimmer › Schreibtisch › obere Schublade", s.drawer),
		"room:2 Neue Wohnung › Küche",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("places:\n%v\nwant:\n%v", got, want)
	}
}

func TestRemovalPreviewForUnknownIsNotFound(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	for _, path := range []string{"/api/storage-locations/999/removal", "/api/items/999/removal"} {
		if r := c.do("GET", path, nil); r.Status != http.StatusNotFound || r.errorCode() != "not_found" {
			t.Errorf("%s: status %d, error code %q", path, r.Status, r.errorCode())
		}
	}
}
