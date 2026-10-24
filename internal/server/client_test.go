package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"inventar/internal/server"
)

func TestClientRoutesReturnTheAppShell(t *testing.T) {
	e := newTestEnvWith(t, func(d *server.Deps) {
		d.Client = fstest.MapFS{
			"index.html":                {Data: []byte("<html>app-shell</html>")},
			"manifest.webmanifest":      {Data: []byte(`{"name":"Inventar"}`)},
			"_app/immutable/entry/a.js": {Data: []byte("console.log(1)")},
			"service-worker.js":         {Data: []byte("self.skipWaiting")},
		}
	})

	for path, want := range map[string]string{
		"/":                          "app-shell",
		"/items/42":                  "app-shell",
		"/manifest.webmanifest":      `"Inventar"`,
		"/_app/immutable/entry/a.js": "console.log",
		"/service-worker.js":         "skipWaiting",
	} {
		res, err := http.Get(e.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("%s: status %d, content %q", path, res.StatusCode, body)
		}
		if path == "/manifest.webmanifest" && res.Header.Get("Content-Type") != "application/manifest+json" {
			t.Errorf("%s: Content-Type %q", path, res.Header.Get("Content-Type"))
		}
		// Only hashed files may be cached for long; the app shell and the service worker must be revalidated on
		// every request, or an installed app keeps the old version after a deploy.
		cacheControl := res.Header.Get("Cache-Control")
		if strings.HasPrefix(path, "/_app/immutable/") {
			if !strings.Contains(cacheControl, "immutable") {
				t.Errorf("%s: Cache-Control %q", path, cacheControl)
			}
		} else if cacheControl != "no-cache" {
			t.Errorf("%s: Cache-Control %q, want no-cache", path, cacheControl)
		}
	}
}
