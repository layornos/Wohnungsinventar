package server_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"inventar/internal/mail"
	"inventar/internal/server"
)

// fresh logs a user in again, e.g. after the clock jumped past the end of the session.
func (e *testEnv) fresh(email string) *apiClient {
	e.t.Helper()
	c, _ := e.loginAs(email, "geheim-geheim")
	return c
}

func (c *apiClient) reminders(query string) []map[string]any {
	c.t.Helper()
	var out []map[string]any
	for _, r := range c.mustStatus(http.StatusOK, "GET", "/api/reminders"+query, nil).Body["reminders"].([]any) {
		out = append(out, r.(map[string]any))
	}
	return out
}

func (c *apiClient) react(reminder map[string]any, reaction string) response {
	c.t.Helper()
	return c.do("POST", fmt.Sprintf("/api/reminders/%v/reaction", reminder["id"]), map[string]any{"reaction": reaction})
}

func schedulesOf(item response) []string {
	var out []string
	for _, s := range item.Body["schedules"].([]any) {
		m := s.(map[string]any)
		out = append(out, fmt.Sprintf("%s %s", m["kind"], m["dueOn"]))
	}
	return out
}

func recipients(mails []mail.Message) []string {
	var to []string
	for _, m := range mails {
		to = append(to, m.To)
	}
	slices.Sort(to)
	return to
}

func TestKeepReviewComesAfterThreeYearsAndGoesOutInTheNextMonthlyMail(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	if got := schedulesOf(c.edit(desk, "keep", nil)); !slices.Equal(got, []string{"keepReview 2029-03-01"}) {
		t.Fatalf("schedules: %v", got)
	}

	clk.now = berlin(t, "2029-03-01 07:59")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if open := c.reminders(""); len(open) != 0 {
		t.Fatalf("before 8:00: %v", open)
	}
	clk.now = berlin(t, "2029-03-01 08:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	open := c.reminders("")
	if len(open) != 1 || open[0]["kind"] != "keepReview" || open[0]["dueOn"] != "2029-03-01" {
		t.Fatalf("open: %v", open)
	}
	if n := c.mustStatus(http.StatusOK, "GET", "/api/reminders/count", nil).Body["open"]; n != 1.0 {
		t.Errorf("count %v", n)
	}
	if sent := e.mailer.Sent(); len(sent) != 0 {
		t.Fatalf("mail before the next 1st: %v", sent)
	}

	clk.now = berlin(t, "2029-04-01 08:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	e.runReminders()
	sent := e.mailer.Sent()
	if len(sent) != 1 || sent[0].To != "alex@example.org" || sent[0].Subject != "Inventar: 1 offene Erinnerung" {
		t.Fatalf("mails: %+v", sent)
	}
	if !strings.Contains(sent[0].Text, "Behalte-Prüfung: Schreibtisch (Alex)") || !strings.Contains(sent[0].Text, "In der App unter Erinnerungen.") {
		t.Errorf("text:\n%s", sent[0].Text)
	}

	// The next one comes in May only if still open and not mailed before: it was in April's mail.
	clk.now = berlin(t, "2029-05-01 08:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if n := len(e.mailer.Sent()); n != 1 {
		t.Errorf("%d mails after May", n)
	}
	// Without reaction the rhythm stays.
	if got := schedulesOf(c.get(itemPath(desk))); !slices.Equal(got, []string{"keepReview 2032-03-01"}) {
		t.Errorf("next: %v", got)
	}
}

func TestKeepReviewKeepsItsRhythmWithReactionAndAfterAnOutage(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	c.edit(desk, "keep", nil)

	// The server was off from March 1 to 5.
	clk.now = berlin(t, "2029-03-05 12:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	e.runReminders()
	open := c.reminders("")
	if len(open) != 1 {
		t.Fatalf("open after outage: %v", open)
	}
	done := c.react(open[0], "keep")
	if done.Status != http.StatusOK || done.Body["reaction"] != "keep" || done.Body["doneBy"].(map[string]any)["name"] != "Alex" {
		t.Fatalf("react: %d %s", done.Status, done.Raw)
	}
	if got := schedulesOf(c.get(itemPath(desk))); !slices.Equal(got, []string{"keepReview 2032-03-01"}) {
		t.Errorf("next: %v", got)
	}
	if r := c.react(open[0], "keep"); r.errorCode() != "reminder_done" {
		t.Errorf("second reaction: %s", r.Raw)
	}
	if got := c.reminders("?state=done"); len(got) != 1 {
		t.Errorf("history: %v", got)
	}
}

func TestSellLaterRemindsOnceAndItsReactions(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	trailer := c.createItem("Fahrradanhänger", []int64{c.me()}, testJPEG(t))
	c.edit(trailer, "sell_later", nil)

	clk.now = berlin(t, "2027-03-01 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	open := c.reminders("")
	if len(open) != 1 || open[0]["kind"] != "sellLater" {
		t.Fatalf("open: %v", open)
	}
	if r := c.react(open[0], "keep"); r.errorCode() != "reaction_mismatch" {
		t.Errorf("keep on sell later: %s", r.Raw)
	}
	c.react(open[0], "oneMoreYear")
	if got := schedulesOf(c.get(itemPath(trailer))); !slices.Equal(got, []string{"sellLater 2028-03-01"}) {
		t.Fatalf("one more year: %v", got)
	}

	clk.now = berlin(t, "2028-03-01 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	c.react(c.reminders("")[0], "keepAfterAll")
	item := c.get(itemPath(trailer))
	if item.Body["disposition"] != "keep" || !slices.Equal(schedulesOf(item), []string{"keepReview 2031-03-01"}) {
		t.Errorf("keep after all: %v %v", item.Body["disposition"], schedulesOf(item))
	}
}

func TestChangingTheDispositionBeforehandMeansNoReminder(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	trailer := c.createItem("Fahrradanhänger", []int64{c.me()}, testJPEG(t))
	c.edit(trailer, "sell_later", nil)
	clk.now = berlin(t, "2026-09-01 10:00")
	c = e.fresh("alex@example.org")
	if got := schedulesOf(c.edit(trailer, "sell", nil)); len(got) != 0 {
		t.Errorf("schedules for sell: %v", got)
	}
	clk.now = berlin(t, "2027-03-02 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if open := c.reminders(""); len(open) != 0 {
		t.Errorf("reminder despite change: %v", open)
	}
}

func TestOpenRemindersAreSettledByAChangeOrAHandover(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	c.edit(desk, "keep", nil)
	clk.now = berlin(t, "2029-03-02 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()

	sold := c.handOver(desk, map[string]any{"outcome": "sold", "saleProceeds": 5000})
	if len(schedulesOf(sold)) != 0 {
		t.Errorf("schedules in the archive: %v", schedulesOf(sold))
	}
	if open := c.reminders(""); len(open) != 0 {
		t.Errorf("open after handover: %v", open)
	}
	back := c.mustStatus(http.StatusOK, "DELETE", itemPath(desk)+"/handover", nil)
	// Back as "sell", which has no reminder of its own here.
	if len(schedulesOf(back)) != 0 {
		t.Errorf("schedules after taking back: %v", schedulesOf(back))
	}
	if got := schedulesOf(c.edit(back, "keep", nil)); !slices.Equal(got, []string{"keepReview 2032-03-02"}) {
		t.Errorf("keep again: %v", got)
	}
}

func TestFebruary29BecomesFebruary28(t *testing.T) {
	e, _ := newTestEnvAt(t, berlin(t, "2028-02-29 10:00"))
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	if got := schedulesOf(c.edit(item, "keep", nil)); !slices.Equal(got, []string{"keepReview 2031-02-28"}) {
		t.Errorf("schedules: %v", got)
	}
}

func TestMonthlyMailRecipients(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	e.createUser("Anna", "anna@example.org", "geheim-geheim")
	anna, _ := e.loginAs("anna@example.org", "geheim-geheim")
	child := c.createPerson("Kind")
	toy := c.createItem("Spielzeug", []int64{child}, testJPEG(t))
	bike := c.createItem("Fahrrad", []int64{anna.me(), child}, testJPEG(t))
	c.edit(toy, "keep", nil)
	c.edit(bike, "keep", nil)

	clk.now = berlin(t, "2029-03-15 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if mine := c.reminders("?mine=1"); len(mine) != 1 || mine[0]["item"].(map[string]any)["name"] != "Spielzeug" {
		t.Errorf("Alex's reminders: %v", mine)
	}
	if all := c.reminders(""); len(all) != 2 {
		t.Errorf("all reminders: %v", all)
	}

	// A user created after the reminders became due gets them too.
	e.createUser("Ben", "ben@example.org", "geheim-geheim")
	clk.now = berlin(t, "2029-04-01 08:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	got := map[string]string{}
	for _, m := range e.mailer.Sent() {
		got[m.To] = m.Text
	}
	if !strings.Contains(got["alex@example.org"], "Spielzeug") || strings.Contains(got["alex@example.org"], "Fahrrad") {
		t.Errorf("Alex:\n%s", got["alex@example.org"])
	}
	if !strings.Contains(got["anna@example.org"], "Spielzeug") || !strings.Contains(got["anna@example.org"], "Fahrrad (Anna, Kind)") {
		t.Errorf("Anna:\n%s", got["anna@example.org"])
	}
	if !strings.Contains(got["ben@example.org"], "Spielzeug") {
		t.Errorf("Ben:\n%s", got["ben@example.org"])
	}
}

func TestSwitchedOffMailIsNotPassedOnToOthers(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	e.createUser("Anna", "anna@example.org", "geheim-geheim")
	anna, _ := e.loginAs("anna@example.org", "geheim-geheim")
	if me := anna.mustStatus(http.StatusOK, "PUT", "/api/me/reminder-mail", map[string]any{"enabled": false}); me.Body["reminderMail"] != false {
		t.Fatalf("setting: %s", me.Raw)
	}
	bike := c.createItem("Fahrrad", []int64{anna.me(), c.createPerson("Kind")}, testJPEG(t))
	c.edit(bike, "keep", nil)
	clk.now = berlin(t, "2029-03-15 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	clk.now = berlin(t, "2029-04-01 09:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if sent := e.mailer.Sent(); len(sent) != 0 {
		t.Errorf("mails: %v", recipients(sent))
	}
	if open := e.fresh("anna@example.org").reminders("?mine=1"); len(open) != 1 {
		t.Errorf("still in the app: %v", open)
	}
}

func TestOneMailForManyAndSettledOnesAreLeftOut(t *testing.T) {
	e, clk := newTestEnvAt(t, berlin(t, "2026-03-01 10:00"))
	c := e.loggedIn()
	for i := range 10 {
		item := c.createItem(fmt.Sprintf("Buch %d", i), []int64{c.me()}, testJPEG(t))
		c.edit(item, "keep", nil)
	}
	clk.now = berlin(t, "2029-03-10 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	c.react(c.reminders("")[0], "keep")

	clk.now = berlin(t, "2029-04-01 08:30")
	c = e.fresh("alex@example.org")
	e.runReminders()
	sent := e.mailer.Sent()
	if len(sent) != 1 || sent[0].Subject != "Inventar: 9 offene Erinnerungen" {
		t.Fatalf("mails: %+v", sent)
	}
}

// failingMailer fails until told otherwise.
type failingMailer struct {
	mu    sync.Mutex
	fail  bool
	tries int
	mail.Recorder
}

func (f *failingMailer) Send(ctx context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tries++
	if f.fail {
		return errors.New("smtp down")
	}
	return f.Recorder.Send(ctx, m)
}

func TestFailedMailIsTriedAgainAndTheReminderStaysInTheApp(t *testing.T) {
	failing := &failingMailer{fail: true}
	c0 := &clock{now: berlin(t, "2026-03-01 10:00")}
	e := newTestEnvWith(t, func(d *server.Deps) {
		d.Now = c0.Now
		d.Mailer = failing
		d.Config.PublicURL = "https://inventar.example.org"
	})
	c := e.loggedIn()
	desk := c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(t))
	c.edit(desk, "keep", nil)
	c0.now = berlin(t, "2029-03-15 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()

	c0.now = berlin(t, "2029-04-01 08:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	if len(c.reminders("")) != 1 || len(failing.Sent()) != 0 {
		t.Fatalf("after failure: %d sent", len(failing.Sent()))
	}
	failing.mu.Lock()
	failing.fail = false
	failing.mu.Unlock()
	e.runReminders()
	sent := failing.Sent()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Zu den Erinnerungen: https://inventar.example.org/erinnerungen") {
		t.Fatalf("retry: %+v", sent)
	}

	// Five failures end the trying.
	other := c.createItem("Regal", []int64{c.me()}, testJPEG(t))
	c.edit(other, "keep", nil)
	c0.now = berlin(t, "2032-04-15 10:00")
	c = e.fresh("alex@example.org")
	e.runReminders()
	failing.mu.Lock()
	failing.fail, failing.tries = true, 0
	failing.mu.Unlock()
	c0.now = berlin(t, "2032-05-01 08:00")
	c = e.fresh("alex@example.org")
	for range 7 {
		e.runReminders()
	}
	if failing.tries != 5 {
		t.Errorf("%d attempts, want 5", failing.tries)
	}
}
