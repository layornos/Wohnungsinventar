package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"inventar/internal/server"
)

func TestClientRoutenLiefernDieAppHuelle(t *testing.T) {
	e := newTestEnvMit(t, func(d *server.Deps) {
		d.Client = fstest.MapFS{
			"index.html":                {Data: []byte("<html>app-huelle</html>")},
			"manifest.webmanifest":      {Data: []byte(`{"name":"Inventar"}`)},
			"_app/immutable/entry/a.js": {Data: []byte("console.log(1)")},
		}
	})

	for path, want := range map[string]string{
		"/":                          "app-huelle",
		"/items/42":                  "app-huelle",
		"/manifest.webmanifest":      `"Inventar"`,
		"/_app/immutable/entry/a.js": "console.log",
	} {
		res, err := http.Get(e.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("%s: Status %d, Inhalt %q", path, res.StatusCode, body)
		}
		if path == "/manifest.webmanifest" && res.Header.Get("Content-Type") != "application/manifest+json" {
			t.Errorf("%s: Content-Type %q", path, res.Header.Get("Content-Type"))
		}
		if strings.HasPrefix(path, "/_app/immutable/") && !strings.Contains(res.Header.Get("Cache-Control"), "immutable") {
			t.Errorf("%s: Cache-Control %q", path, res.Header.Get("Cache-Control"))
		}
	}
}
