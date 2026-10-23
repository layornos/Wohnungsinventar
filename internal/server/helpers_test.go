package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"inventar/internal/cli"
	"inventar/internal/config"
	"inventar/internal/mail"
	"inventar/internal/server"
	"inventar/internal/store"
)

// testEnv ist ein echter Server mit echter SQLite-Datei in einem temporären Datenverzeichnis.
type testEnv struct {
	t       *testing.T
	dataDir string
	mailer  *mail.Recorder
	srv     *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWith(t, fstest.MapFS{})
}

func newTestEnvWith(t *testing.T, client fs.FS) *testEnv {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("store öffnen: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	rec := &mail.Recorder{}
	h := server.New(server.Deps{
		Config: config.Config{DataDir: dataDir},
		Store:  st,
		Mailer: rec,
		Client: client,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &testEnv{t: t, dataDir: dataDir, mailer: rec, srv: srv}
}

// runCLI führt einen Unterbefehl gegen dasselbe Datenverzeichnis aus.
func (e *testEnv) runCLI(stdin string, args ...string) (string, error) {
	e.t.Helper()
	var out bytes.Buffer
	env := map[string]string{"INVENTAR_DATA_DIR": e.dataDir}
	err := cli.Run(context.Background(), args, strings.NewReader(stdin), &out, func(k string) string { return env[k] })
	return out.String(), err
}

func (e *testEnv) createNutzer(name, email, password string) {
	e.t.Helper()
	if out, err := e.runCLI(password+"\n", "nutzer-anlegen", "--name", name, "--email", email); err != nil {
		e.t.Fatalf("nutzer-anlegen: %v\n%s", err, out)
	}
}

type apiClient struct {
	t    *testing.T
	base string
	http *http.Client
}

func (e *testEnv) client() *apiClient {
	jar, _ := cookiejar.New(nil)
	return &apiClient{t: e.t, base: e.srv.URL, http: &http.Client{Jar: jar}}
}

// loggedIn legt einen Nutzer an und liefert einen angemeldeten Client.
func (e *testEnv) loggedIn() *apiClient {
	e.t.Helper()
	e.createNutzer("Alex", "alex@example.org", "geheim-geheim")
	c := e.client()
	c.mustStatus(http.StatusOK, "POST", "/api/anmeldung", map[string]any{"email": "alex@example.org", "passwort": "geheim-geheim"})
	return c
}

type response struct {
	Status int
	Body   map[string]any
	Raw    []byte
	Header http.Header
}

func (c *apiClient) do(method, path string, body any) response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := response{Status: res.StatusCode, Raw: raw, Header: res.Header}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (c *apiClient) mustStatus(want int, method, path string, body any) response {
	c.t.Helper()
	res := c.do(method, path, body)
	if res.Status != want {
		c.t.Fatalf("%s %s: Status %d, erwartet %d: %s", method, path, res.Status, want, res.Raw)
	}
	return res
}

// errorCode liefert den fachlichen Fehlercode einer Fehlerantwort.
func (r response) errorCode() string {
	s, _ := r.Body["code"].(string)
	return s
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
