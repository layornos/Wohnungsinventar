package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"inventar/internal/server"
)

// clock is a controllable time for the server.
type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func newTestEnvAt(t *testing.T, now time.Time) (*testEnv, *clock) {
	t.Helper()
	c := &clock{now: now}
	return newTestEnvWith(t, func(d *server.Deps) { d.Now = c.Now }), c
}

func berlin(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

// edit changes name, disposition and, if given, the asking price of an item.
func (c *apiClient) edit(item response, disposition string, extra map[string]any) response {
	c.t.Helper()
	body := map[string]any{"name": item.Body["name"], "description": item.Body["description"], "disposition": disposition}
	for k, v := range extra {
		body[k] = v
	}
	return c.do("PUT", itemPath(item), body)
}

func (c *apiClient) handOver(item response, body map[string]any) response {
	c.t.Helper()
	return c.do("PUT", itemPath(item)+"/handover", body)
}

func TestAskingPriceOnlyForSaleAndKeptAcrossDispositions(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	lamp := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	if r := c.edit(lamp, "keep", map[string]any{"askingPrice": 4000}); r.Status != http.StatusBadRequest || r.errorCode() != "no_sale" {
		t.Fatalf("price for keep: status %d, code %q", r.Status, r.errorCode())
	}
	r := c.edit(lamp, "sell_later", map[string]any{"askingPrice": 4000})
	if r.Status != http.StatusOK || r.Body["askingPrice"] != 4000.0 {
		t.Fatalf("price for sell later: %d %s", r.Status, r.Raw)
	}
	if r.Body["offeredSince"] != nil {
		t.Errorf("sell later counts as offered: %v", r.Body["offeredSince"])
	}
	// Without the field the price stays, whatever the disposition becomes.
	for _, d := range []string{"keep", "give_away", "sell"} {
		r = c.edit(lamp, d, nil)
		if r.Status != http.StatusOK || r.Body["askingPrice"] != 4000.0 {
			t.Fatalf("%s: price %v (%d)", d, r.Body["askingPrice"], r.Status)
		}
	}
	if r.Body["offeredSince"] == nil {
		t.Error("sell has no offeredSince")
	}
	// null removes it.
	if r = c.edit(lamp, "sell", map[string]any{"askingPrice": nil}); r.Body["askingPrice"] != nil {
		t.Errorf("price not removed: %s", r.Raw)
	}
	if r = c.edit(lamp, "sell", map[string]any{"askingPrice": -1}); r.errorCode() != "invalid_asking_price" {
		t.Errorf("negative price: %s", r.Raw)
	}
}

func TestOfferedSinceStartsAnewOnEveryChangeToSellOrGiveAway(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	lamp := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	first := c.edit(lamp, "sell", nil).Body["offeredSince"]
	clk.now = clk.now.Add(48 * time.Hour)
	if again := c.edit(lamp, "sell", nil).Body["offeredSince"]; again != first {
		t.Errorf("saving with the same disposition changed offeredSince: %v → %v", first, again)
	}
	if switched := c.edit(lamp, "give_away", nil).Body["offeredSince"]; switched == first {
		t.Error("change to give away kept the old offeredSince")
	}
}

func TestListingTextFromTemplateWithDisclaimer(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	kettle := c.createItem("Wasserkocher", []int64{c.me()}, testJPEG(t))
	listing := itemPath(kettle) + "/listing"

	c.edit(kettle, "sell", nil)
	if r := c.do("GET", listing, nil); r.Status != http.StatusConflict || r.errorCode() != "no_asking_price" {
		t.Fatalf("without price: %d %s", r.Status, r.Raw)
	}
	c.mustStatus(http.StatusOK, "PUT", itemPath(kettle), map[string]any{
		"name": "Wasserkocher", "description": "Edelstahl, 1,7 l.", "disposition": "sell", "askingPrice": 1550})
	r := c.mustStatus(http.StatusOK, "GET", listing, nil)
	want := "Edelstahl, 1,7 l.\n\nPreis: 15,50 €\n\nAbholung oder Versand nach Absprache.\n\n" +
		"Privatverkauf. Der Verkauf erfolgt unter Ausschluss jeglicher Gewährleistung, Garantie und Rücknahme."
	if r.Body["title"] != "Wasserkocher" || r.Body["text"] != want {
		t.Fatalf("listing:\n%v\n%v", r.Body["title"], r.Body["text"])
	}

	// Giving away needs no price; an empty description drops its paragraph.
	c.mustStatus(http.StatusOK, "PUT", itemPath(kettle), map[string]any{"name": "Wasserkocher", "description": "", "disposition": "give_away"})
	text := c.mustStatus(http.StatusOK, "GET", listing, nil).Body["text"].(string)
	if !strings.HasPrefix(text, "Preis: zu verschenken\n\n") || !strings.HasSuffix(text, "Rücknahme.") {
		t.Errorf("give away:\n%s", text)
	}

	c.edit(kettle, "keep", nil)
	if r := c.do("GET", listing, nil); r.errorCode() != "no_sale" {
		t.Errorf("keep: %s", r.Raw)
	}
}

func TestListingTitleIsCutAndTheFullNameOpensTheText(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	name := strings.Repeat("Schreibtisch ", 6) + "aus Eiche"
	item := c.createItem(name, []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(item), map[string]any{"name": name, "disposition": "sell", "askingPrice": 120000})

	r := c.mustStatus(http.StatusOK, "GET", itemPath(item)+"/listing", nil)
	title := r.Body["title"].(string)
	if n := len([]rune(title)); n > 65 || !strings.HasSuffix(title, "…") {
		t.Errorf("title %q (%d characters)", title, n)
	}
	if text := r.Body["text"].(string); !strings.HasPrefix(text, name+"\n\nPreis: 1.200 €") {
		t.Errorf("text:\n%s", text)
	}
}

func TestListingSettingsNeedADisclaimer(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	if r := c.do("PUT", "/api/settings/listing", map[string]any{"template": "{name}: {preis}", "warrantyDisclaimer": "  "}); r.errorCode() != "warranty_disclaimer_missing" {
		t.Fatalf("empty disclaimer: %s", r.Raw)
	}
	c.mustStatus(http.StatusOK, "PUT", "/api/settings/listing", map[string]any{"template": "{name}: {preis}", "warrantyDisclaimer": "Ohne Garantie."})
	mug := c.createItem("Tasse", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(mug), map[string]any{"name": "Tasse", "disposition": "sell", "askingPrice": 200})
	if text := c.mustStatus(http.StatusOK, "GET", itemPath(mug)+"/listing", nil).Body["text"]; text != "Tasse: 2 €\n\nOhne Garantie." {
		t.Errorf("text %q", text)
	}
}

func TestSellingFurnitureArchivesItAndLeavesItsContentsUnstored(t *testing.T) {
	e, _ := newTestEnvAt(t, berlin(t, "2026-03-31 12:00"))
	c := e.loggedIn()
	s := c.createStudy()
	deskItem := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", s.desk), map[string]any{"itemId": deskItem.Body["id"]})
	scissors := c.itemIn("Schere", s.drawer)

	// From keep, without a price: the handover sets the disposition.
	sold := c.handOver(deskItem, map[string]any{"outcome": "sold", "saleProceeds": 8000})
	if sold.Status != http.StatusOK || sold.Body["disposition"] != "sell" || sold.Body["handedOverOn"] != "2026-03-31" || sold.Body["saleProceeds"] != 8000.0 {
		t.Fatalf("sold: %d %s", sold.Status, sold.Raw)
	}
	if sold.Body["placement"] != nil || len(sold.Body["furnitureStorageLocations"].([]any)) != 0 {
		t.Errorf("sold item still placed or furniture: %s", sold.Raw)
	}
	if got := pathText(t, c.get(itemPath(scissors)).Body["placement"]); got != "Altbau › Arbeitszimmer" {
		t.Errorf("scissors: %q", got)
	}
	c.mustStatus(http.StatusNotFound, "GET", fmt.Sprintf("/api/storage-locations/%d", s.drawer), nil)

	for _, path := range []string{"/api/items", "/api/items?q=Schreib", fmt.Sprintf("/api/items?room=%d", s.room), "/api/items?unstored=1"} {
		if names := namesOf(c.mustStatus(http.StatusOK, "GET", path, nil)); slices.Contains(names, "Schreibtisch") {
			t.Errorf("%s lists the sold item: %v", path, names)
		}
	}
	archive := c.mustStatus(http.StatusOK, "GET", "/api/archive", nil)
	if got := namesOf(archive); !slices.Equal(got, []string{"Schreibtisch"}) {
		t.Errorf("archive: %v", got)
	}
}

func TestArchivedItemsAreReadOnlyButCanBeCorrectedAndTakenBack(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-31 12:00"))
	c := e.loggedIn()
	lamp := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(lamp), map[string]any{"name": "Lampe", "disposition": "sell", "askingPrice": 4000})
	c.handOver(lamp, map[string]any{"outcome": "sold", "saleProceeds": 3500})

	for _, r := range []response{
		c.edit(lamp, "keep", nil),
		c.do("PUT", fmt.Sprintf("%s/owners/%d", itemPath(lamp), c.createPerson("Kim")), nil),
		c.do("POST", "/api/items/bulk", map[string]any{"itemIds": []any{lamp.Body["id"]}, "disposition": "keep"}),
	} {
		if r.Status != http.StatusConflict || r.errorCode() != "item_archived" {
			t.Errorf("change in archive: %d %s", r.Status, r.Raw)
		}
	}

	if r := c.handOver(lamp, map[string]any{"outcome": "sold", "saleProceeds": 3000, "date": "2026-04-01"}); r.errorCode() != "date_in_future" {
		t.Errorf("future date: %s", r.Raw)
	}
	fixed := c.handOver(lamp, map[string]any{"outcome": "sold", "saleProceeds": 3000, "date": "2026-03-30"})
	if fixed.Body["saleProceeds"] != 3000.0 || fixed.Body["handedOverOn"] != "2026-03-30" {
		t.Errorf("correction: %s", fixed.Raw)
	}

	clk.now = clk.now.Add(time.Hour)
	back := c.mustStatus(http.StatusOK, "DELETE", itemPath(lamp)+"/handover", nil)
	if back.Body["handedOverOn"] != nil || back.Body["saleProceeds"] != nil || back.Body["disposition"] != "sell" || back.Body["askingPrice"] != 4000.0 {
		t.Errorf("taken back: %s", back.Raw)
	}
	if got := namesOf(c.mustStatus(http.StatusOK, "GET", "/api/items", nil)); !slices.Equal(got, []string{"Lampe"}) {
		t.Errorf("stock after taking back: %v", got)
	}
	if r := c.do("DELETE", itemPath(lamp)+"/handover", nil); r.errorCode() != "not_archived" {
		t.Errorf("taking back twice: %s", r.Raw)
	}
}

func TestGivingAwayHasNoProceeds(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	mug := c.createItem("Tasse", []int64{c.me()}, testJPEG(t))
	r := c.handOver(mug, map[string]any{"outcome": "givenAway", "saleProceeds": 500})
	if r.Body["disposition"] != "give_away" || r.Body["saleProceeds"] != 0.0 {
		t.Fatalf("given away: %s", r.Raw)
	}
	if r := c.handOver(c.createItem("Teller", []int64{c.me()}, testJPEG(t)), map[string]any{"outcome": "sold"}); r.errorCode() != "sale_proceeds_missing" {
		t.Errorf("sold without proceeds: %s", r.Raw)
	}
}

func TestProceedsSummaryUsesTheHouseholdsDaysAndSplitsAmongOwners(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-31 23:30"))
	c := e.loggedIn()
	me := c.me()
	anna := c.createPerson("Anna")
	sofa := c.createItem("Sofa", []int64{me, anna}, testJPEG(t))
	chair := c.createItem("Stuhl", []int64{me}, testJPEG(t))
	mug := c.createItem("Tasse", []int64{anna}, testJPEG(t))

	// 23:30 in Berlin is 21:30 UTC, still March 31.
	c.handOver(sofa, map[string]any{"outcome": "sold", "saleProceeds": 30001})
	clk.now = berlin(t, "2026-04-01 00:30")
	c.handOver(chair, map[string]any{"outcome": "sold", "saleProceeds": 2000})
	c.handOver(mug, map[string]any{"outcome": "givenAway"})

	march := c.mustStatus(http.StatusOK, "GET", "/api/proceeds?from=2026-03-01&to=2026-03-31", nil)
	if march.Body["total"] != 30001.0 || march.Body["soldCount"] != 1.0 || march.Body["givenAwayCount"] != 0.0 {
		t.Errorf("March: %s", march.Raw)
	}
	all := c.mustStatus(http.StatusOK, "GET", "/api/proceeds", nil)
	if all.Body["total"] != 32001.0 || all.Body["soldCount"] != 2.0 || all.Body["givenAwayCount"] != 1.0 {
		t.Errorf("all: %s", all.Raw)
	}
	// Alex (the logged-in user) comes before Anna by name and gets the cent left over; both shares add up to the proceeds.
	annas := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/proceeds?owner=%d", anna), nil)
	mine := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/proceeds?owner=%d", me), nil)
	if annas.Body["total"] != 15000.0 || mine.Body["total"] != 15001.0+2000.0 {
		t.Errorf("shares: Anna %v, mine %v", annas.Body["total"], mine.Body["total"])
	}
	if got := namesOf(mine); !slices.Equal(got, []string{"Sofa", "Stuhl"}) {
		t.Errorf("my items: %v", got)
	}
}

func TestListWithoutAskingPrice(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	lamp := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	chair := c.createItem("Stuhl", []int64{c.me()}, testJPEG(t))
	c.edit(lamp, "sell", nil)
	c.edit(chair, "sell", map[string]any{"askingPrice": 1000})
	if got := namesOf(c.mustStatus(http.StatusOK, "GET", "/api/items?disposition=sell&withoutAskingPrice=1", nil)); !slices.Equal(got, []string{"Lampe"}) {
		t.Errorf("without price: %v", got)
	}
}
