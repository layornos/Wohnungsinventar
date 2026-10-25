package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func (c *apiClient) reactWith(reminder map[string]any, body map[string]any) response {
	c.t.Helper()
	return c.do("POST", fmt.Sprintf("/api/reminders/%v/reaction", reminder["id"]), body)
}

func TestPriceReminderAfter30DaysAtTheSamePrice(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	lamp := c.createItem("Stehlampe", []int64{c.me()}, testJPEG(t))

	// Without a price there is nothing to check.
	if got := schedulesOf(c.edit(lamp, "sell", nil)); len(got) != 0 {
		t.Fatalf("without price: %v", got)
	}
	if got := schedulesOf(c.edit(lamp, "sell", map[string]any{"askingPrice": 4000})); !slices.Equal(got, []string{"priceReminder 2026-03-31"}) {
		t.Fatalf("with price: %v", got)
	}
	// A new price on March 20 starts the 30 days anew; saving the same price doesn't.
	clk.now = berlin(t, "2026-03-20 10:00")
	c = e.fresh("alex@example.org")
	c.edit(lamp, "sell", map[string]any{"askingPrice": 3500})
	clk.now = berlin(t, "2026-03-25 10:00")
	c = e.fresh("alex@example.org")
	if got := schedulesOf(c.edit(lamp, "sell", map[string]any{"askingPrice": 3500})); !slices.Equal(got, []string{"priceReminder 2026-04-19"}) {
		t.Fatalf("after price change: %v", got)
	}

	clk.now = berlin(t, "2026-03-31 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if open := c.reminders(""); len(open) != 0 {
		t.Fatalf("reminder despite new price: %v", open)
	}
	clk.now = berlin(t, "2026-04-19 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	open := c.reminders("")
	if len(open) != 1 || open[0]["kind"] != "priceReminder" || open[0]["item"].(map[string]any)["askingPrice"] != 3500.0 {
		t.Fatalf("open: %v", open)
	}
	// The run never changes the price.
	if p := c.get(itemPath(lamp)).Body["askingPrice"]; p != 3500.0 {
		t.Errorf("price %v", p)
	}

	// Keeping the price: the next one 30 days after the due day.
	c.react(open[0], "keepPrice")
	if got := schedulesOf(c.get(itemPath(lamp))); !slices.Equal(got, []string{"priceReminder 2026-05-19"}) {
		t.Errorf("after keeping the price: %v", got)
	}
}

func TestChangingThePriceInTheReminderStartsAnew(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	lamp := c.createItem("Stehlampe", []int64{c.me()}, testJPEG(t))
	c.edit(lamp, "sell", map[string]any{"askingPrice": 4000})
	clk.now = berlin(t, "2026-04-05 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	open := c.reminders("")[0]

	if r := c.react(open, "changePrice"); r.errorCode() != "invalid_asking_price" {
		t.Errorf("without price: %s", r.Raw)
	}
	if r := c.react(open, "keepOffering"); r.errorCode() != "reaction_mismatch" {
		t.Errorf("give-away answer: %s", r.Raw)
	}
	r := c.reactWith(open, map[string]any{"reaction": "changePrice", "askingPrice": 3000})
	if r.Status != http.StatusOK || r.Body["reaction"] != "changePrice" {
		t.Fatalf("change price: %d %s", r.Status, r.Raw)
	}
	item := c.get(itemPath(lamp))
	if item.Body["askingPrice"] != 3000.0 || !slices.Equal(schedulesOf(item), []string{"priceReminder 2026-05-05"}) {
		t.Errorf("after change: %v %v", item.Body["askingPrice"], schedulesOf(item))
	}
}

func TestGiveAwayReminderAfter30DaysUntilGivenAway(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	kettle := c.createItem("Wasserkocher", []int64{c.me()}, testJPEG(t))
	if got := schedulesOf(c.edit(kettle, "give_away", nil)); !slices.Equal(got, []string{"giveAwayReminder 2026-03-31"}) {
		t.Fatalf("schedules: %v", got)
	}

	clk.now = berlin(t, "2026-03-31 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	open := c.reminders("")
	if len(open) != 1 || open[0]["kind"] != "giveAwayReminder" {
		t.Fatalf("open: %v", open)
	}
	c.react(open[0], "keepOffering")
	if got := schedulesOf(c.get(itemPath(kettle))); !slices.Equal(got, []string{"giveAwayReminder 2026-04-30"}) {
		t.Errorf("after keep offering: %v", got)
	}

	clk.now = berlin(t, "2026-04-30 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	c.handOver(kettle, map[string]any{"outcome": "givenAway"})
	if open := c.reminders(""); len(open) != 0 {
		t.Errorf("open after giving away: %v", open)
	}
	if got := schedulesOf(c.get(itemPath(kettle))); len(got) != 0 {
		t.Errorf("schedules after giving away: %v", got)
	}
}

func TestUnansweredPriceReminderStaysOneAndGoesIntoTheMailWithPrice(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	lamp := c.createItem("Stehlampe", []int64{c.me()}, testJPEG(t))
	c.edit(lamp, "sell", map[string]any{"askingPrice": 4000})

	// Two periods without an answer: still one open reminder, the rhythm goes on.
	clk.now = berlin(t, "2026-03-31 09:00")
	e.runReminders()
	clk.now = berlin(t, "2026-04-30 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if open := c.reminders(""); len(open) != 1 {
		t.Fatalf("open: %v", open)
	}
	if got := schedulesOf(c.get(itemPath(lamp))); !slices.Equal(got, []string{"priceReminder 2026-05-30"}) {
		t.Errorf("next: %v", got)
	}

	clk.now = berlin(t, "2026-05-01 08:00")
	e.runReminders()
	sent := e.mailer.Sent()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Preis prüfen: Stehlampe (40 €, Alex)") {
		t.Fatalf("mails: %+v", sent)
	}
}

func TestChangingTheDispositionEndsThePriceReminder(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	lamp := c.createItem("Stehlampe", []int64{c.me()}, testJPEG(t))
	c.edit(lamp, "sell", map[string]any{"askingPrice": 4000})
	clk.now = berlin(t, "2026-04-01 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()

	// Removing the price ends it too.
	item := c.edit(lamp, "sell", map[string]any{"askingPrice": nil})
	if len(schedulesOf(item)) != 0 || len(c.reminders("")) != 0 {
		t.Errorf("without price: %v, open %v", schedulesOf(item), c.reminders(""))
	}
	c.edit(lamp, "sell", map[string]any{"askingPrice": 4000})
	if got := schedulesOf(c.edit(lamp, "keep", nil)); !slices.Equal(got, []string{"keepReview 2029-04-01"}) {
		t.Errorf("keep: %v", got)
	}
}
