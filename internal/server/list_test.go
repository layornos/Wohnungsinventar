package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"inventar/internal/server"
)

// namesInList fetches the item list with the query and returns the names in their order.
func (c *apiClient) namesInList(query string) []string {
	c.t.Helper()
	res := c.mustStatus(http.StatusOK, "GET", "/api/items"+query, nil)
	var names []string
	for _, i := range res.Body["items"].([]any) {
		names = append(names, i.(map[string]any)["name"].(string))
	}
	return names
}

func (c *apiClient) itemWithDisposition(name string, owners []int64, disposition string) {
	c.t.Helper()
	item := c.createItem(name, owners, testJPEG(c.t))
	if disposition != "" {
		c.mustStatus(http.StatusOK, "PUT", itemPath(item), map[string]any{"name": name, "disposition": disposition})
	}
}

func TestFilterItemList(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	me, kim := c.me(), c.createPerson("Kim")
	c.itemWithDisposition("Schreibtischlampe", []int64{me}, "sell")
	c.itemWithDisposition("Lampenschirm", []int64{kim}, "")
	c.itemWithDisposition("Äpfelkorb", []int64{me, kim}, "keep")
	c.itemWithDisposition("Schere 50%_rabatt", []int64{kim}, "")

	for _, tc := range []struct {
		query string
		names []string
	}{
		{"", []string{"Schere 50%_rabatt", "Äpfelkorb", "Lampenschirm", "Schreibtischlampe"}},
		{"?q=LAMPE", []string{"Lampenschirm", "Schreibtischlampe"}},
		{"?q=äpfel", []string{"Äpfelkorb"}},
		{"?q=50%25_", []string{"Schere 50%_rabatt"}},
		{"?q=%25", []string{"Schere 50%_rabatt"}},
		{fmt.Sprintf("?owner=%d", me), []string{"Äpfelkorb", "Schreibtischlampe"}},
		{"?disposition=undecided", []string{"Schere 50%_rabatt", "Lampenschirm"}},
		{"?disposition=keep", []string{"Äpfelkorb"}},
		{fmt.Sprintf("?q=lampe&owner=%d&disposition=undecided", kim), []string{"Lampenschirm"}},
		{"?disposition=give_away", nil},
	} {
		if got := c.namesInList(tc.query); !slices.Equal(got, tc.names) {
			t.Errorf("%s: %v, want %v", tc.query, got, tc.names)
		}
	}
	for _, query := range []string{"?disposition=throw_away", "?disposition=offen", "?owner=kim", "?page=0", "?perPage=1000"} {
		if res := c.do("GET", "/api/items"+query, nil); res.Status != http.StatusBadRequest {
			t.Errorf("%s: status %d", query, res.Status)
		}
	}
}

func TestMostRecentlyUpdatedItemsComeFirst(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	old := c.createItem("Alt", []int64{c.me()}, testJPEG(t))
	c.createItem("Neu", []int64{c.me()}, testJPEG(t))

	if got := c.namesInList(""); !slices.Equal(got, []string{"Neu", "Alt"}) {
		t.Fatalf("before the change: %v", got)
	}
	c.mustStatus(http.StatusOK, "PUT", itemPath(old), map[string]any{"name": "Alt", "disposition": "keep"})
	if got := c.namesInList(""); !slices.Equal(got, []string{"Alt", "Neu"}) {
		t.Fatalf("after the change: %v", got)
	}
}

func TestItemListInPages(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	for _, name := range []string{"A", "B", "C"} {
		c.createItem(name, []int64{c.me()}, testJPEG(t))
	}

	first := c.mustStatus(http.StatusOK, "GET", "/api/items?perPage=2", nil)
	second := c.mustStatus(http.StatusOK, "GET", "/api/items?perPage=2&page=2", nil)
	if first.Body["more"] != true || second.Body["more"] != false {
		t.Fatalf("more: %v, %v", first.Body["more"], second.Body["more"])
	}
	if got := c.namesInList("?perPage=2"); !slices.Equal(got, []string{"C", "B"}) {
		t.Errorf("first page: %v", got)
	}
	if got := c.namesInList("?perPage=2&page=2"); !slices.Equal(got, []string{"A"}) {
		t.Errorf("second page: %v", got)
	}
}

func TestSortingIsRightWithinOneSecond(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 5, 0, time.UTC)
	e := newTestEnvWith(t, func(d *server.Deps) { d.Now = func() time.Time { return now } })
	c := e.loggedIn()

	c.createItem("Volle Sekunde", []int64{c.me()}, testJPEG(t))
	now = now.Add(500 * time.Millisecond)
	c.createItem("Halbe Sekunde später", []int64{c.me()}, testJPEG(t))

	if got := c.namesInList(""); !slices.Equal(got, []string{"Halbe Sekunde später", "Volle Sekunde"}) {
		t.Fatalf("order %v", got)
	}
}
