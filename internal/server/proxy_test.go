package server_test

import (
	"net/http"
	"strings"
	"testing"

	"inventar/internal/server"
)

// newTestEnvBehindProxy starts the server as it runs behind the shared reverse proxy (ADR-0005).
func newTestEnvBehindProxy(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWith(t, func(d *server.Deps) { d.Config.BehindProxy = true })
}

// sessionCookieOf returns the session cookie set by a response.
func sessionCookieOf(t *testing.T, res response) *http.Cookie {
	t.Helper()
	for _, c := range (&http.Response{Header: res.Header}).Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatalf("no session cookie in %v", res.Header)
	return nil
}

// loginResponse creates a user, logs in and returns the login response.
func loginResponse(t *testing.T, e *testEnv) response {
	t.Helper()
	e.createUser("Alex", "alex@example.org", "geheim-geheim")
	_, res := e.loginAs("alex@example.org", "geheim-geheim")
	return res
}

// inviteVia creates an invitation with the given forwarding header and returns the link. The
// session cookie is passed along by hand, because the cookie jar does not send Secure cookies
// over plain HTTP.
func inviteVia(t *testing.T, e *testEnv, session *http.Cookie, forwardedProto string) string {
	t.Helper()
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/invitations", strings.NewReader(`{"name":"Kim"}`))
	req.Host = "inventar.example.duckdns.org"
	if forwardedProto != "" {
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	req.AddCookie(session)
	res := e.client().send(req)
	if res.Status != http.StatusCreated {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	link, _ := res.Body["link"].(string)
	return link
}

func TestBehindProxyTheSessionCookieIsSecure(t *testing.T) {
	e := newTestEnvBehindProxy(t)
	if c := sessionCookieOf(t, loginResponse(t, e)); !c.Secure {
		t.Fatal("session cookie is not Secure behind the proxy")
	}
}

func TestWithoutProxyTheSessionCookieIsNotSecureOverHTTP(t *testing.T) {
	e := newTestEnv(t)
	if c := sessionCookieOf(t, loginResponse(t, e)); c.Secure {
		t.Fatal("session cookie is Secure over plain HTTP without proxy")
	}
}

func TestBehindProxyTheInvitationLinkUsesHTTPS(t *testing.T) {
	e := newTestEnvBehindProxy(t)
	session := sessionCookieOf(t, loginResponse(t, e))
	for _, proto := range []string{"", "https"} {
		link := inviteVia(t, e, session, proto)
		if !strings.HasPrefix(link, "https://inventar.example.duckdns.org/einladung#") {
			t.Errorf("X-Forwarded-Proto %q: link %q", proto, link)
		}
	}
	// Only behind the proxy is the header trusted, also when it reports plain HTTP.
	if link := inviteVia(t, e, session, "http"); !strings.HasPrefix(link, "http://") {
		t.Errorf("X-Forwarded-Proto http: link %q", link)
	}
}

func TestWithoutProxyXForwardedProtoIsIgnored(t *testing.T) {
	e := newTestEnv(t)
	session := sessionCookieOf(t, loginResponse(t, e))
	link := inviteVia(t, e, session, "https")
	if !strings.HasPrefix(link, "http://inventar.example.duckdns.org/einladung#") {
		t.Fatalf("link %q", link)
	}
}
