package server_test

import (
	"net/http"
	"slices"
	"testing"
)

// books creates n items owned by the logged-in user.
func (c *apiClient) books(n int) ([]response, []any) {
	c.t.Helper()
	var items []response
	var ids []any
	for range n {
		item := c.createItem("Buch", []int64{c.me()}, testJPEG(c.t))
		items = append(items, item)
		ids = append(ids, item.Body["id"])
	}
	return items, ids
}

func TestBulkSetsDisposition(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	items, ids := c.books(10)

	res := c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids, "disposition": "give_away"})
	if res.Body["count"] != float64(10) {
		t.Fatalf("count: %s", res.Raw)
	}
	for _, item := range items {
		if got := c.get(itemPath(item)).Body["disposition"]; got != "give_away" {
			t.Fatalf("disposition %v", got)
		}
	}

	// null sets the disposition back to undecided.
	c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids[:1], "disposition": nil})
	if got := c.get(itemPath(items[0])).Body["disposition"]; got != nil {
		t.Fatalf("disposition after null: %v", got)
	}
	if got := c.get(itemPath(items[1])).Body["disposition"]; got != "give_away" {
		t.Fatalf("untouched item: %v", got)
	}
}

func TestBulkAddsOwner(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")
	items, ids := c.books(2)

	res := c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids, "addOwnerId": kim})
	if res.Body["count"] != float64(2) {
		t.Fatalf("count: %s", res.Raw)
	}
	for _, item := range items {
		got := c.get(itemPath(item))
		if names := ownerNames(got); !slices.Contains(names, "Kim") || !slices.Contains(names, "Alex") {
			t.Fatalf("owners %v", names)
		}
	}
	// Adding an owner twice changes nothing.
	c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids, "addOwnerId": kim})
	if names := ownerNames(c.get(itemPath(items[0]))); len(names) != 2 {
		t.Fatalf("owners after adding twice: %v", names)
	}

	r := c.do("POST", "/api/items/bulk", map[string]any{"itemIds": ids, "addOwnerId": 999})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unknown_person" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
}

func TestBulkSetsPlacement(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	apartment := c.createApartment("Altbau")
	office := c.createRoom(apartment, "Büro")
	shelf := c.createStorageLocation(map[string]any{"name": "Regal", "roomId": office})
	items, ids := c.books(2)

	c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids, "storageLocationId": shelf})
	for _, item := range items {
		if got := pathText(t, c.get(itemPath(item)).Body["placement"]); got != "Altbau › Büro › Regal" {
			t.Fatalf("placement %q", got)
		}
	}
	c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{"itemIds": ids, "roomId": office})
	for _, item := range items {
		if got := pathText(t, c.get(itemPath(item)).Body["placement"]); got != "Altbau › Büro" {
			t.Fatalf("placement %q", got)
		}
	}

	r := c.do("POST", "/api/items/bulk", map[string]any{"itemIds": ids, "roomId": office, "storageLocationId": shelf})
	if r.Status != http.StatusBadRequest || r.errorCode() != "invalid_place" {
		t.Fatalf("both: status %d, error code %q", r.Status, r.errorCode())
	}
	r = c.do("POST", "/api/items/bulk", map[string]any{"itemIds": ids, "storageLocationId": 999})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unknown_place" {
		t.Fatalf("unknown: status %d, error code %q", r.Status, r.errorCode())
	}
}

func TestBulkAppliesSeveralChangesAtOnce(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")
	office := c.createRoom(c.createApartment("Altbau"), "Büro")
	items, ids := c.books(1)

	c.mustStatus(http.StatusOK, "POST", "/api/items/bulk", map[string]any{
		"itemIds": ids, "disposition": "sell", "addOwnerId": kim, "roomId": office,
	})
	got := c.get(itemPath(items[0]))
	if got.Body["disposition"] != "sell" || !slices.Contains(ownerNames(got), "Kim") || pathText(t, got.Body["placement"]) != "Altbau › Büro" {
		t.Fatalf("item: %s", got.Raw)
	}
}

func TestBulkIsAllOrNothing(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kim := c.createPerson("Kim")
	items, ids := c.books(2)

	r := c.do("POST", "/api/items/bulk", map[string]any{"itemIds": append(ids, 999), "disposition": "sell", "addOwnerId": kim})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unknown_item" {
		t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
	}
	for _, item := range items {
		got := c.get(itemPath(item))
		if got.Body["disposition"] != nil || len(ownerNames(got)) != 1 {
			t.Fatalf("item changed despite rejected bulk operation: %s", got.Raw)
		}
	}
}

func TestBulkRejectsInvalidRequests(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	_, ids := c.books(1)

	for _, tc := range []struct {
		name string
		body map[string]any
		code string
	}{
		{"invalid disposition", map[string]any{"itemIds": ids, "disposition": "verbrennen"}, "invalid_disposition"},
		{"no items", map[string]any{"itemIds": []any{}, "disposition": "sell"}, "invalid_request"},
		{"no change", map[string]any{"itemIds": ids}, "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := c.do("POST", "/api/items/bulk", tc.body)
			if r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
				t.Fatalf("status %d, error code %q", r.Status, r.errorCode())
			}
		})
	}
}

func TestBulkNeedsSession(t *testing.T) {
	e := newTestEnv(t)
	e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/items/bulk", map[string]any{"itemIds": []any{1}, "disposition": "sell"})
}
