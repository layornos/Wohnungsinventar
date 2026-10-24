package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

// photosOf returns the photos of an item representation.
func photosOf(r response) []map[string]any {
	var out []map[string]any
	for _, p := range r.Body["photos"].([]any) {
		out = append(out, p.(map[string]any))
	}
	return out
}

func ownerNames(r response) []string {
	var out []string
	for _, o := range r.Body["owners"].([]any) {
		out = append(out, o.(map[string]any)["name"].(string))
	}
	return out
}

func TestCreateItemWithPhotoAndOwner(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")

	res := c.multipart("POST", "/api/items", form{
		fields: [][2]string{
			{"name", " Schreibtischlampe "}, {"description", "Messing"}, {"disposition", "sell"},
			{"owner", fmt.Sprint(c.me())}, {"owner", fmt.Sprint(kim)},
		},
		photos: [][]byte{testJPEG(t), testPNG(t)},
	})
	if res.Status != http.StatusCreated {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	if res.Body["name"] != "Schreibtischlampe" || res.Body["description"] != "Messing" || res.Body["disposition"] != "sell" {
		t.Fatalf("item: %s", res.Raw)
	}
	if got := ownerNames(res); len(got) != 2 {
		t.Fatalf("owners %v", got)
	}
	photos := photosOf(res)
	if len(photos) != 2 || photos[0]["mainPhoto"] != true || photos[1]["mainPhoto"] != false {
		t.Fatalf("the first photo should be the main photo: %s", res.Raw)
	}

	fetched := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", res.Body["id"]), nil)
	if fetched.Body["name"] != "Schreibtischlampe" || len(photosOf(fetched)) != 2 {
		t.Fatalf("fetched item: %s", fetched.Raw)
	}
}

func TestItemWithoutDispositionIsUndecided(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	res := c.createItem("Schere", []int64{c.me()}, testJPEG(t))
	if v, present := res.Body["disposition"]; !present || v != nil {
		t.Fatalf("disposition should be null: %s", res.Raw)
	}
}

func TestCreateItemViolatesNoInvariant(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	me := fmt.Sprint(c.me())

	for _, tc := range []struct {
		name   string
		f      form
		status int
		code   string
	}{
		{"without photo", form{fields: [][2]string{{"name", "Lampe"}, {"owner", me}}}, http.StatusBadRequest, "photo_missing"},
		{"without owner", form{fields: [][2]string{{"name", "Lampe"}}, photos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "owner_missing"},
		{"without name", form{fields: [][2]string{{"name", " "}, {"owner", me}}, photos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "name_missing"},
		{"unknown owner", form{fields: [][2]string{{"name", "Lampe"}, {"owner", "999"}}, photos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "unknown_person"},
		{"unknown disposition", form{fields: [][2]string{{"name", "Lampe"}, {"owner", me}, {"disposition", "throw_away"}}, photos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "invalid_disposition"},
		{"German disposition", form{fields: [][2]string{{"name", "Lampe"}, {"owner", me}, {"disposition", "verkaufen"}}, photos: [][]byte{testJPEG(t)}}, http.StatusBadRequest, "invalid_disposition"},
	} {
		res := c.multipart("POST", "/api/items", tc.f)
		if res.Status != tc.status || res.errorCode() != tc.code {
			t.Errorf("%s: status %d, error code %q, want %d %q", tc.name, res.Status, res.errorCode(), tc.status, tc.code)
		}
	}

	list := c.mustStatus(http.StatusOK, "GET", "/api/items", nil)
	if n := len(list.Body["items"].([]any)); n != 0 {
		t.Fatalf("no items should have been created, found: %d", n)
	}
	e.noPhotoFiles()
}

func itemPath(r response) string { return fmt.Sprintf("/api/items/%v", r.Body["id"]) }

func TestEditItemAndResetDisposition(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	res := c.mustStatus(http.StatusOK, "PUT", itemPath(item), map[string]any{"name": "Stehlampe", "description": "mit Dimmer", "disposition": "give_away"})
	if res.Body["name"] != "Stehlampe" || res.Body["description"] != "mit Dimmer" || res.Body["disposition"] != "give_away" {
		t.Fatalf("edited: %s", res.Raw)
	}
	if res.Body["updatedAt"] == item.Body["updatedAt"] {
		t.Errorf("updatedAt should change")
	}

	res = c.mustStatus(http.StatusOK, "PUT", itemPath(item), map[string]any{"name": "Stehlampe", "description": "", "disposition": nil})
	if v, present := res.Body["disposition"]; !present || v != nil {
		t.Fatalf("disposition should be undecided again: %s", res.Raw)
	}

	for _, tc := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"name": " ", "disposition": nil}, "name_missing"},
		{map[string]any{"name": "Lampe", "disposition": "throw_away"}, "invalid_disposition"},
	} {
		if r := c.do("PUT", itemPath(item), tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNotFound, "PUT", "/api/items/999", map[string]any{"name": "x"})
}

func TestDeleteItemRemovesPhotoFiles(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t), testPNG(t))
	p := photosOf(item)[0]

	c.mustStatus(http.StatusNoContent, "DELETE", itemPath(item), nil)
	c.mustStatus(http.StatusNotFound, "GET", itemPath(item), nil)
	if res := c.get(p["url"].(string)); res.Status != http.StatusNotFound {
		t.Errorf("photo after deleting: status %d", res.Status)
	}
	e.noPhotoFiles()
}

func TestLastOwnerCannotBeRemoved(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	res := c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("%s/owners/%d", itemPath(item), kim), nil)
	if got := ownerNames(res); len(got) != 2 {
		t.Fatalf("owners after adding: %v", got)
	}
	// Adding again changes nothing.
	res = c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("%s/owners/%d", itemPath(item), kim), nil)
	if got := ownerNames(res); len(got) != 2 {
		t.Fatalf("owners after adding twice: %v", got)
	}

	res = c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("%s/owners/%d", itemPath(item), c.me()), nil)
	if got := ownerNames(res); len(got) != 1 || got[0] != "Kim" {
		t.Fatalf("owners after removing: %v", got)
	}
	res = c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("%s/owners/%d", itemPath(item), kim), nil)
	if res.errorCode() != "last_owner" {
		t.Fatalf("error code %q", res.errorCode())
	}

	res = c.mustStatus(http.StatusBadRequest, "PUT", fmt.Sprintf("%s/owners/999", itemPath(item)), nil)
	if res.errorCode() != "unknown_person" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestItemsAndPersonsAreLockedWithoutSession(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	stranger := e.client()
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/items"},
		{"POST", "/api/items"},
		{"GET", itemPath(item)},
		{"PUT", itemPath(item)},
		{"DELETE", itemPath(item)},
		{"POST", itemPath(item) + "/photos"},
		{"PUT", fmt.Sprintf("%s/owners/%d", itemPath(item), c.me())},
		{"GET", "/api/persons"},
		{"POST", "/api/persons"},
	} {
		if res := stranger.do(req.method, req.path, nil); res.Status != http.StatusUnauthorized {
			t.Errorf("%s %s: status %d", req.method, req.path, res.Status)
		}
	}
	c.mustStatus(http.StatusOK, "GET", itemPath(item), nil)
}
