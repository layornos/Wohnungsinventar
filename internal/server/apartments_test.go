package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (c *apiClient) createApartment(name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/apartments", map[string]any{"name": name}).Body["id"].(float64))
}

func (c *apiClient) createRoom(apartment int64, name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", fmt.Sprintf("/api/apartments/%d/rooms", apartment), map[string]any{"name": name}).Body["id"].(float64))
}

func TestManageApartmentsAndRooms(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	old := c.createApartment("Altbau")
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/apartments/%d", old), map[string]any{"name": "Altbau Weststadt"})
	office := c.createRoom(old, "Büro")
	c.createRoom(old, "Küche")
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/rooms/%d", office), map[string]any{"name": "Arbeitszimmer"})
	c.createApartment("Neue Wohnung")

	res := c.mustStatus(http.StatusOK, "GET", "/api/apartments", nil)
	want := `{"apartments":[{"id":1,"name":"Altbau Weststadt","rooms":[{"id":1,"name":"Arbeitszimmer"},{"id":2,"name":"Küche"}]},{"id":2,"name":"Neue Wohnung","rooms":[]}]}`
	if got := string(res.Raw); got != want+"\n" {
		t.Fatalf("apartments:\n%s\nwant:\n%s", got, want)
	}

	room := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/rooms/%d", office), nil)
	if room.Body["name"] != "Arbeitszimmer" || room.Body["apartment"].(map[string]any)["name"] != "Altbau Weststadt" {
		t.Fatalf("room: %s", room.Raw)
	}

	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/apartments"},
		{"POST", fmt.Sprintf("/api/apartments/%d/rooms", old)},
	} {
		if r := c.do(tc.method, tc.path, map[string]any{"name": " "}); r.errorCode() != "name_missing" {
			t.Errorf("%s %s without name: error code %q", tc.method, tc.path, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNotFound, "POST", "/api/apartments/999/rooms", map[string]any{"name": "Bad"})
}

func TestApartmentCanOnlyBeDeletedWithoutRooms(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	a := c.createApartment("Altbau")
	r := c.createRoom(a, "Küche")

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/apartments/%d", a), nil)
	if res.errorCode() != "apartment_not_empty" {
		t.Fatalf("error code %q", res.errorCode())
	}
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/rooms/%d", r), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/apartments/%d", a), nil)
	c.mustStatus(http.StatusNotFound, "DELETE", fmt.Sprintf("/api/apartments/%d", a), nil)
}
