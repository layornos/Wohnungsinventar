package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// storageLocationQR fetches the QR code of a storage location under the given host, with the session
// cookie by hand (the cookie jar does not send Secure cookies over plain HTTP).
func storageLocationQR(t *testing.T, e *testEnv, session *http.Cookie, id int64) response {
	t.Helper()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/storage-locations/%d/qr", e.srv.URL, id), nil)
	req.Host = "inventar.example.duckdns.org"
	req.AddCookie(session)
	return e.client().send(req)
}

func TestQRCodeOfAStorageLocationLinksToItsPage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    func(*testing.T) *testEnv
		scheme string
	}{
		{"without proxy", newTestEnv, "http"},
		{"behind proxy", newTestEnvBehindProxy, "https"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.env(t)
			session := sessionCookieOf(t, loginResponse(t, e))
			c := e.client()
			c.http.Jar.SetCookies(mustURL(t, e.srv.URL), []*http.Cookie{{Name: session.Name, Value: session.Value}})
			s := c.createStudy()

			res := storageLocationQR(t, e, session, s.drawer)
			if res.Status != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "image/svg+xml") {
				t.Fatalf("status %d, type %q: %s", res.Status, res.Header.Get("Content-Type"), res.Raw)
			}
			want := fmt.Sprintf("%s://inventar.example.duckdns.org/lagerorte/%d", tc.scheme, s.drawer)
			if !strings.Contains(string(res.Raw), "<svg") || !qrContentMatches(t, res.Raw, want) {
				t.Fatalf("QR code does not encode %q:\n%s", want, res.Raw)
			}
			if res := storageLocationQR(t, e, session, 999); res.Status != http.StatusNotFound {
				t.Fatalf("unknown storage location: status %d", res.Status)
			}
		})
	}
}

func TestStorageLocationQRCodeNeedsLogin(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	if res := e.client().do("GET", fmt.Sprintf("/api/storage-locations/%d/qr", s.drawer), nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("without login: status %d", res.Status)
	}
}
