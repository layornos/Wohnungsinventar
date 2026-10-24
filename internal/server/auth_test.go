package server_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestUserCreatedBySubcommandCanLogIn(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("Alex", "alex@example.org", "geheim-geheim")

	c := e.client()
	c.mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": "Alex@Example.org", "password": "geheim-geheim"})

	me := c.mustStatus(http.StatusOK, "GET", "/api/me", nil)
	if me.Body["name"] != "Alex" || me.Body["email"] != "alex@example.org" {
		t.Fatalf("unexpected user: %s", me.Raw)
	}
}

func TestWrongPasswordCreatesNoSession(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("Alex", "alex@example.org", "geheim-geheim")

	c := e.client()
	res := c.mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "alex@example.org", "password": "falsch-falsch"})
	if res.errorCode() != "login_failed" {
		t.Fatalf("error code %q", res.errorCode())
	}
	c.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)
}

func TestUnknownEmailIsTreatedLikeWrongPassword(t *testing.T) {
	e := newTestEnv(t)
	res := e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "niemand@example.org", "password": "egal-egal"})
	if res.errorCode() != "login_failed" {
		t.Fatalf("error code %q", res.errorCode())
	}
}

func TestAPIIsLockedWithoutSession(t *testing.T) {
	e := newTestEnv(t)
	for _, req := range []struct{ method, path string }{
		{"GET", "/api/me"},
		{"POST", "/api/logout"},
		{"GET", "/api/does-not-exist"},
	} {
		res := e.client().do(req.method, req.path, nil)
		if res.Status != http.StatusUnauthorized || res.errorCode() != "not_logged_in" {
			t.Errorf("%s %s: status %d, error code %q", req.method, req.path, res.Status, res.errorCode())
		}
	}
}

func TestSessionIsInvalidAfterLogout(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	cookie := c.http.Jar.Cookies(mustURL(t, e.srv.URL))

	c.mustStatus(http.StatusNoContent, "POST", "/api/logout", nil)
	c.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)

	// The old cookie, e.g. copied from another device, is no longer valid either.
	other := e.client()
	other.http.Jar.SetCookies(mustURL(t, e.srv.URL), cookie)
	other.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)
}

func TestSessionCookieIsProtected(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("Alex", "alex@example.org", "geheim-geheim")
	res := e.client().mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": "alex@example.org", "password": "geheim-geheim"})

	set := res.Header.Get("Set-Cookie")
	for _, want := range []string{"session=", "HttpOnly", "SameSite=Strict", "Max-Age=7776000"} {
		if !strings.Contains(set, want) {
			t.Errorf("Set-Cookie %q does not contain %q", set, want)
		}
	}
	if strings.Contains(set, "Secure") {
		t.Errorf("without TLS the cookie must not be Secure: %q", set)
	}
}

func TestPasswordResetEndsAllSessions(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()

	if out, err := e.runCLI("ganz-neues-passwort\n", "reset-password", "--email", "alex@example.org"); err != nil {
		t.Fatalf("reset-password: %v\n%s", err, out)
	}

	c.mustStatus(http.StatusUnauthorized, "GET", "/api/me", nil)
	fresh := e.client()
	fresh.mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "alex@example.org", "password": "geheim-geheim"})
	fresh.mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": "alex@example.org", "password": "ganz-neues-passwort"})
}

func TestEmailCanOnlyBeTakenOnce(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("Alex", "alex@example.org", "geheim-geheim")
	if _, err := e.runCLI("anderes-passwort\n", "create-user", "--name", "Kim", "--email", "ALEX@example.org"); err == nil {
		t.Fatal("second user with the same email was created")
	}
}

func TestTooShortPasswordIsRejected(t *testing.T) {
	e := newTestEnv(t)
	if _, err := e.runCLI("kurz\n", "create-user", "--name", "Alex", "--email", "alex@example.org"); err == nil {
		t.Fatal("user with too short password was created")
	}
	e.client().mustStatus(http.StatusUnauthorized, "POST", "/api/login", map[string]any{"email": "alex@example.org", "password": "kurz"})
}

func TestUnknownAPIEndpointReturnsJSONError(t *testing.T) {
	e := newTestEnv(t)
	res := e.loggedIn().mustStatus(http.StatusNotFound, "GET", "/api/does-not-exist", nil)
	if res.errorCode() != "not_found" {
		t.Fatalf("error code %q", res.errorCode())
	}
}
