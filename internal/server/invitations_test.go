package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"inventar/internal/server"
)

// testClock is a controllable clock for the server, so tests can check expiry.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestEnvWithClock(t *testing.T) (*testEnv, *testClock) {
	t.Helper()
	clock := &testClock{now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	return newTestEnvWith(t, func(d *server.Deps) { d.Now = clock.Now }), clock
}

// invite creates an invitation and returns the token from the link.
func (c *apiClient) invite(body map[string]any) (token string, res response) {
	c.t.Helper()
	res = c.mustStatus(http.StatusCreated, "POST", "/api/invitations", body)
	link, _ := res.Body["link"].(string)
	u, err := url.Parse(link)
	if err != nil || u.Fragment == "" {
		c.t.Fatalf("link %q without token", link)
	}
	return u.Fragment, res
}

func acceptance(token, email string) map[string]any {
	return map[string]any{"token": token, "email": email, "password": "kims-passwort"}
}

func TestAfterAcceptingTheNewPersonCanLogIn(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, _ := alex.invite(map[string]any{"name": "Kim"})

	kim := e.client()
	accepted := kim.mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "Kim@Example.org"))
	if accepted.Body["name"] != "Kim" || accepted.Body["email"] != "kim@example.org" {
		t.Fatalf("accepted: %s", accepted.Raw)
	}
	if v, present := accepted.Body["theme"]; !present || v != nil {
		t.Fatalf("accepted: theme should be null: %s", accepted.Raw)
	}
	// After accepting, Kim is logged in right away …
	if me := kim.mustStatus(http.StatusOK, "GET", "/api/me", nil); me.Body["name"] != "Kim" {
		t.Fatalf("me: %s", me.Raw)
	}
	// … and can later log in with email and password.
	e.client().mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": "kim@example.org", "password": "kims-passwort"})
}

func TestLinkIsBuiltFromTheRequestedAddress(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/invitations", strings.NewReader(`{"name":"Kim"}`))
	req.Host = "inventar.example.duckdns.org"
	// The cookie jar assigns cookies to the host, so pass the session cookie along ourselves.
	for _, c := range alex.http.Jar.Cookies(mustURL(t, e.srv.URL)) {
		req.AddCookie(c)
	}
	res := alex.send(req)
	if res.Status != http.StatusCreated {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	link, _ := res.Body["link"].(string)
	if !strings.HasPrefix(link, "http://inventar.example.duckdns.org/einladung#") {
		t.Fatalf("link %q", link)
	}
	// At least 128 bits, URL-safe encoded.
	token := strings.TrimPrefix(link, "http://inventar.example.duckdns.org/einladung#")
	if len(token) < 22 || strings.ContainsAny(token, "+/=") {
		t.Fatalf("token %q", token)
	}
}

func TestOpenInvitationShowsNameAndCreator(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().invite(map[string]any{"name": "Kim"})

	res := e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/check", map[string]any{"token": token})
	if res.Body["name"] != "Kim" || res.Body["createdBy"] != "Alex" {
		t.Fatalf("check: %s", res.Raw)
	}
}

func TestInvitationExpiresAfterSevenDays(t *testing.T) {
	e, clock := newTestEnvWithClock(t)
	alex := e.loggedIn()
	token, _ := alex.invite(map[string]any{"name": "Kim"})

	clock.advance(7*24*time.Hour - time.Minute)
	e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/check", map[string]any{"token": token})

	clock.advance(2 * time.Minute)
	kim := e.client()
	for _, path := range []string{"/api/invitations/check", "/api/invitations/accept"} {
		res := kim.mustStatus(http.StatusGone, "POST", path, acceptance(token, "kim@example.org"))
		if res.errorCode() != "invitation_expired" || res.Body["name"] != nil {
			t.Errorf("%s: %s", path, res.Raw)
		}
	}
	kim.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)
	if list := alex.mustStatus(http.StatusOK, "GET", "/api/invitations", nil); len(list.Body["invitations"].([]any)) != 0 {
		t.Fatalf("expired invitation is still open: %s", list.Raw)
	}
}

func TestInvitationCanOnlyBeUsedOnce(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().invite(map[string]any{"name": "Kim"})
	e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))

	other := e.client()
	for _, path := range []string{"/api/invitations/check", "/api/invitations/accept"} {
		res := other.mustStatus(http.StatusGone, "POST", path, acceptance(token, "kim2@example.org"))
		if res.errorCode() != "invitation_used" {
			t.Errorf("%s: %s", path, res.Raw)
		}
	}
	other.mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "kim2@example.org", "password": "kims-passwort"})
}

func TestRevokedInvitationNoLongerWorks(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, created := alex.invite(map[string]any{"name": "Kim"})
	id := created.Body["id"]

	list := alex.mustStatus(http.StatusOK, "GET", "/api/invitations", nil)
	open := list.Body["invitations"].([]any)
	if len(open) != 1 {
		t.Fatalf("open invitations: %s", list.Raw)
	}
	entry := open[0].(map[string]any)
	if entry["name"] != "Kim" || entry["createdBy"] != "Alex" || entry["expiresAt"] == nil || entry["link"] != nil {
		t.Fatalf("entry: %v", entry)
	}

	alex.mustStatus(http.StatusNoContent, "DELETE", fmt.Sprintf("/api/invitations/%v", id), nil)
	if list := alex.mustStatus(http.StatusOK, "GET", "/api/invitations", nil); len(list.Body["invitations"].([]any)) != 0 {
		t.Fatalf("revoked invitation is still open: %s", list.Raw)
	}
	for _, path := range []string{"/api/invitations/check", "/api/invitations/accept"} {
		res := e.client().mustStatus(http.StatusGone, "POST", path, acceptance(token, "kim@example.org"))
		if res.errorCode() != "invitation_revoked" {
			t.Errorf("%s: %s", path, res.Raw)
		}
	}
	alex.mustStatus(http.StatusNotFound, "DELETE", "/api/invitations/999", nil)
}

func TestAcceptedInvitationCannotBeRevoked(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, created := alex.invite(map[string]any{"name": "Kim"})
	e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))

	res := alex.mustStatus(http.StatusGone, "DELETE", fmt.Sprintf("/api/invitations/%v", created.Body["id"]), nil)
	if res.errorCode() != "invitation_used" {
		t.Fatalf("error code %q", res.errorCode())
	}
	e.client().mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": "kim@example.org", "password": "kims-passwort"})
}

func TestLinkedPersonBecomesUserAndStaysOwner(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	kimID := alex.createPerson("Kim")
	item := alex.createItem("Kuscheltier", []int64{kimID}, testJPEG(t))
	token, created := alex.invite(map[string]any{"personId": kimID})
	if created.Body["name"] != "Kim" || created.Body["personId"] != float64(kimID) {
		t.Fatalf("created: %s", created.Raw)
	}

	kim := e.client()
	kim.mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))
	if got := kim.me(); got != kimID {
		t.Fatalf("Kim is person %d, want %d", got, kimID)
	}
	after := kim.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/items/%v", item.Body["id"]), nil)
	if got := ownerNames(after); len(got) != 1 || got[0] != "Kim" {
		t.Fatalf("owners: %v", got)
	}
	// No second person Kim, and Kim now has user access.
	persons := alex.mustStatus(http.StatusOK, "GET", "/api/persons", nil).Body["persons"].([]any)
	if len(persons) != 2 {
		t.Fatalf("persons: %v", persons)
	}
	for _, p := range persons {
		if p := p.(map[string]any); p["name"] == "Kim" && p["isUser"] != true {
			t.Fatalf("Kim is not a user: %v", p)
		}
	}
}

func TestNewPersonIsCreatedOnAcceptance(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, created := alex.invite(map[string]any{"name": " Kim "})
	if created.Body["personId"] != nil || created.Body["name"] != "Kim" {
		t.Fatalf("created: %s", created.Raw)
	}
	// Before accepting, the person does not exist yet.
	if p := alex.mustStatus(http.StatusOK, "GET", "/api/persons", nil).Body["persons"].([]any); len(p) != 1 {
		t.Fatalf("persons before accepting: %v", p)
	}
	e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))

	users := alex.mustStatus(http.StatusOK, "GET", "/api/users", nil)
	list := users.Body["users"].([]any)
	if len(list) != 2 || list[0].(map[string]any)["name"] != "Alex" || list[1].(map[string]any)["email"] != "kim@example.org" {
		t.Fatalf("users: %s", users.Raw)
	}
	if strings.Contains(string(users.Raw), "argon2") {
		t.Fatalf("user list contains password hashes: %s", users.Raw)
	}
}

func TestInvitationWithoutNameOrWithUnknownPersonIsRejected(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	if res := alex.mustStatus(http.StatusBadRequest, "POST", "/api/invitations", map[string]any{"name": " "}); res.errorCode() != "name_missing" {
		t.Errorf("without name: %s", res.Raw)
	}
	if res := alex.mustStatus(http.StatusBadRequest, "POST", "/api/invitations", map[string]any{"personId": 999}); res.errorCode() != "unknown_person" {
		t.Errorf("unknown person: %s", res.Raw)
	}
}

func TestEmailIsUniqueOnAcceptance(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	token, _ := alex.invite(map[string]any{"name": "Kim"})

	kim := e.client()
	res := kim.mustStatus(http.StatusConflict, "POST", "/api/invitations/accept", acceptance(token, "ALEX@example.org"))
	if res.errorCode() != "email_taken" {
		t.Fatalf("error code %q", res.errorCode())
	}
	// Rejected means: nothing happened, the invitation stays open.
	kim.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)
	if p := alex.mustStatus(http.StatusOK, "GET", "/api/persons", nil).Body["persons"].([]any); len(p) != 1 {
		t.Fatalf("persons: %v", p)
	}
	kim.mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))
}

func TestAcceptanceChecksEmailAndPassword(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.loggedIn().invite(map[string]any{"name": "Kim"})
	kim := e.client()
	if res := kim.mustStatus(http.StatusBadRequest, "POST", "/api/invitations/accept", acceptance(token, "kim")); res.errorCode() != "email_invalid" {
		t.Errorf("email: %s", res.Raw)
	}
	res := kim.mustStatus(http.StatusBadRequest, "POST", "/api/invitations/accept", map[string]any{"token": token, "email": "kim@example.org", "password": "kurz"})
	if res.errorCode() != "password_too_short" {
		t.Errorf("password: %s", res.Raw)
	}
	kim.mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(token, "kim@example.org"))
}

func TestPersonWhoIsAlreadyAUserCannotBeInvited(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	res := alex.mustStatus(http.StatusConflict, "POST", "/api/invitations", map[string]any{"personId": alex.me()})
	if res.errorCode() != "already_user" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestLinkedPersonWhoBecameAUserMeanwhileCannotAccept(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	kimID := alex.createPerson("Kim")
	first, _ := alex.invite(map[string]any{"personId": kimID})
	second, _ := alex.invite(map[string]any{"personId": kimID})
	e.client().mustStatus(http.StatusOK, "POST", "/api/invitations/accept", acceptance(first, "kim@example.org"))

	// Even the check rejects, so the page explains right away what is going on instead of showing the form first.
	for _, path := range []string{"/api/invitations/check", "/api/invitations/accept"} {
		res := e.client().mustStatus(http.StatusConflict, "POST", path, acceptance(second, "kim2@example.org"))
		if res.errorCode() != "already_user" {
			t.Fatalf("%s: error code %q", path, res.errorCode())
		}
	}
	// The second invitation no longer shows up as open.
	if list := alex.mustStatus(http.StatusOK, "GET", "/api/invitations", nil); len(list.Body["invitations"].([]any)) != 0 {
		t.Fatalf("open invitations: %s", list.Raw)
	}
}

func TestPublicEndpointsRevealNothingWithoutValidToken(t *testing.T) {
	e := newTestEnv(t)
	e.loggedIn().invite(map[string]any{"name": "Kim"})
	for _, token := range []string{"", "wrong"} {
		for _, path := range []string{"/api/invitations/check", "/api/invitations/accept"} {
			res := e.client().mustStatus(http.StatusNotFound, "POST", path, acceptance(token, "kim@example.org"))
			if res.errorCode() != "invitation_invalid" || strings.Contains(string(res.Raw), "Kim") || strings.Contains(string(res.Raw), "Alex") {
				t.Errorf("%s with %q: %s", path, token, res.Raw)
			}
		}
	}
	e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "kim@example.org", "password": "kims-passwort"})
}

func TestManagingInvitationsNeedsASession(t *testing.T) {
	e := newTestEnv(t)
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/invitations"},
		{"POST", "/api/invitations"},
		{"DELETE", "/api/invitations/1"},
		{"GET", "/api/users"},
	} {
		res := e.client().do(req.method, req.path, map[string]any{"name": "Kim"})
		if res.Status != http.StatusUnauthorized || res.errorCode() != "not_logged_in" {
			t.Errorf("%s %s: status %d, error code %q", req.method, req.path, res.Status, res.errorCode())
		}
	}
}
