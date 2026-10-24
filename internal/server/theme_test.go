package server_test

import (
	"net/http"
	"testing"
)

func TestNewUserHasNoThemeYet(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("Alex", "alex@example.org", "geheim-geheim")
	c, login := e.loginAs("alex@example.org", "geheim-geheim")

	if v, present := login.Body["theme"]; !present || v != nil {
		t.Fatalf("login: theme should be null: %s", login.Raw)
	}
	me := c.mustStatus(http.StatusOK, "GET", "/api/me", nil)
	if v, present := me.Body["theme"]; !present || v != nil {
		t.Fatalf("me: theme should be null: %s", me.Raw)
	}
}

func TestThemeAppliesOnAnotherDevice(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()

	res := alex.mustStatus(http.StatusOK, "PUT", "/api/me/theme", map[string]any{"theme": "neon"})
	if res.Body["theme"] != "neon" || res.Body["email"] != "alex@example.org" {
		t.Fatalf("response: %s", res.Raw)
	}
	if me := alex.mustStatus(http.StatusOK, "GET", "/api/me", nil); me.Body["theme"] != "neon" {
		t.Fatalf("me: %s", me.Raw)
	}

	alex.mustStatus(http.StatusNoContent, "POST", "/api/logout", nil)
	otherDevice, login := e.loginAs("alex@example.org", "geheim-geheim")
	if login.Body["theme"] != "neon" {
		t.Fatalf("login: %s", login.Raw)
	}
	otherDevice.mustStatus(http.StatusOK, "PUT", "/api/me/theme", map[string]any{"theme": "soft"})
	if me := otherDevice.mustStatus(http.StatusOK, "GET", "/api/me", nil); me.Body["theme"] != "soft" {
		t.Fatalf("me: %s", me.Raw)
	}
}

func TestThemeBelongsOnlyToOwnUser(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	e.createUser("Kim", "kim@example.org", "kims-passwort")
	kim, _ := e.loginAs("kim@example.org", "kims-passwort")

	alex.mustStatus(http.StatusOK, "PUT", "/api/me/theme", map[string]any{"theme": "minimal"})

	if me := kim.mustStatus(http.StatusOK, "GET", "/api/me", nil); me.Body["theme"] != nil {
		t.Fatalf("Kim: %s", me.Raw)
	}
	// Other users do not see the setting.
	list := kim.mustStatus(http.StatusOK, "GET", "/api/users", nil)
	for _, u := range list.Body["users"].([]any) {
		if _, present := u.(map[string]any)["theme"]; present {
			t.Fatalf("user list shows the theme: %s", list.Raw)
		}
	}
}

func TestUnknownThemeIsRejected(t *testing.T) {
	e := newTestEnv(t)
	alex := e.loggedIn()
	alex.mustStatus(http.StatusOK, "PUT", "/api/me/theme", map[string]any{"theme": "neon"})

	for _, body := range []map[string]any{
		{"theme": "pink"},
		{"theme": "weich"},
		{"theme": ""},
		{"theme": nil},
		{},
	} {
		res := alex.mustStatus(http.StatusBadRequest, "PUT", "/api/me/theme", body)
		if res.errorCode() != "unknown_theme" {
			t.Errorf("%v: error code %q", body, res.errorCode())
		}
	}
	if me := alex.mustStatus(http.StatusOK, "GET", "/api/me", nil); me.Body["theme"] != "neon" {
		t.Fatalf("me: %s", me.Raw)
	}
}

func TestSetThemeNeedsASession(t *testing.T) {
	e := newTestEnv(t)
	res := e.client().mustStatus(http.StatusUnauthorized, "PUT", "/api/me/theme", map[string]any{"theme": "neon"})
	if res.errorCode() != "not_logged_in" {
		t.Fatalf("error code %q", res.errorCode())
	}
}
