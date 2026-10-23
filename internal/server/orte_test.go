package server_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (c *apiClient) wohnungAnlegen(name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/wohnungen", map[string]any{"name": name}).Body["id"].(float64))
}

func (c *apiClient) raumAnlegen(wohnung int64, name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", fmt.Sprintf("/api/wohnungen/%d/raeume", wohnung), map[string]any{"name": name}).Body["id"].(float64))
}

func TestWohnungenUndRaeumeVerwalten(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	alt := c.wohnungAnlegen("Altbau")
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/wohnungen/%d", alt), map[string]any{"name": "Altbau Weststadt"})
	buero := c.raumAnlegen(alt, "Büro")
	c.raumAnlegen(alt, "Küche")
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/raeume/%d", buero), map[string]any{"name": "Arbeitszimmer"})
	c.wohnungAnlegen("Neue Wohnung")

	res := c.mustStatus(http.StatusOK, "GET", "/api/wohnungen", nil)
	want := `{"wohnungen":[{"id":1,"name":"Altbau Weststadt","raeume":[{"id":1,"name":"Arbeitszimmer"},{"id":2,"name":"Küche"}]},{"id":2,"name":"Neue Wohnung","raeume":[]}]}`
	if got := string(res.Raw); got != want+"\n" {
		t.Fatalf("Wohnungen:\n%s\nerwartet:\n%s", got, want)
	}

	raum := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/raeume/%d", buero), nil)
	if raum.Body["name"] != "Arbeitszimmer" || raum.Body["wohnung"].(map[string]any)["name"] != "Altbau Weststadt" {
		t.Fatalf("Raum: %s", raum.Raw)
	}

	for _, fall := range []struct{ method, path string }{
		{"POST", "/api/wohnungen"},
		{"POST", fmt.Sprintf("/api/wohnungen/%d/raeume", alt)},
	} {
		if r := c.do(fall.method, fall.path, map[string]any{"name": " "}); r.errorCode() != "name_fehlt" {
			t.Errorf("%s %s ohne Namen: Fehlercode %q", fall.method, fall.path, r.errorCode())
		}
	}
	c.mustStatus(http.StatusNotFound, "POST", "/api/wohnungen/999/raeume", map[string]any{"name": "Bad"})
}

func TestWohnungNurOhneRaeumeLoeschbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	w := c.wohnungAnlegen("Altbau")
	r := c.raumAnlegen(w, "Küche")

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/wohnungen/%d", w), nil)
	if res.errorCode() != "wohnung_nicht_leer" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/raeume/%d", r), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/wohnungen/%d", w), nil)
	c.mustStatus(http.StatusNotFound, "DELETE", fmt.Sprintf("/api/wohnungen/%d", w), nil)
}
