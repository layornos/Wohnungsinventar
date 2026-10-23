package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
	mailer  *mail.Aufzeichner
	srv     *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvMit(t, func(*server.Deps) {})
}

// newTestEnvMit startet den Server mit angepassten Abhängigkeiten, z. B. einem eigenen Client.
func newTestEnvMit(t *testing.T, anpassen func(*server.Deps)) *testEnv {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("store öffnen: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	rec := &mail.Aufzeichner{}
	d := server.Deps{
		Config: config.Config{Datenverzeichnis: dataDir},
		Store:  st,
		Mailer: rec,
		Client: fstest.MapFS{},
	}
	anpassen(&d)
	srv := httptest.NewServer(server.New(d))
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
	return c.send(req)
}

func (c *apiClient) send(req *http.Request) response {
	c.t.Helper()
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

// formular ist der Inhalt einer multipart-Anfrage: Textfelder (auch mehrfach) und Fotos.
type formular struct {
	felder [][2]string
	fotos  [][]byte
}

func (c *apiClient) multipart(method, path string, f formular) response {
	c.t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, feld := range f.felder {
		w.WriteField(feld[0], feld[1])
	}
	for i, foto := range f.fotos {
		part, err := w.CreateFormFile("foto", fmt.Sprintf("foto%d.jpg", i))
		if err != nil {
			c.t.Fatal(err)
		}
		part.Write(foto)
	}
	w.Close()
	req, err := http.NewRequest(method, c.base+path, &b)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.send(req)
}

// itemAnlegen legt ein Item mit den Besitzern und Fotos an und liefert seine Darstellung.
func (c *apiClient) itemAnlegen(name string, besitzer []int64, fotos ...[]byte) response {
	c.t.Helper()
	f := formular{felder: [][2]string{{"name", name}}, fotos: fotos}
	for _, id := range besitzer {
		f.felder = append(f.felder, [2]string{"besitzer", fmt.Sprint(id)})
	}
	res := c.multipart("POST", "/api/items", f)
	if res.Status != http.StatusCreated {
		c.t.Fatalf("Item anlegen: Status %d: %s", res.Status, res.Raw)
	}
	return res
}

// get ruft eine Adresse ab, z. B. ein Foto, und liefert die rohe Antwort.
func (c *apiClient) get(path string) response {
	c.t.Helper()
	req, err := http.NewRequest("GET", c.base+path, nil)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.send(req)
}

func (c *apiClient) mustStatus(want int, method, path string, body any) response {
	c.t.Helper()
	res := c.do(method, path, body)
	if res.Status != want {
		c.t.Fatalf("%s %s: Status %d, erwartet %d: %s", method, path, res.Status, want, res.Raw)
	}
	return res
}

// ich liefert die Person-ID des angemeldeten Nutzers.
func (c *apiClient) ich() int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusOK, "GET", "/api/ich", nil).Body["personId"].(float64))
}

func (c *apiClient) personAnlegen(name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/personen", map[string]any{"name": name}).Body["id"].(float64))
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

// keineFotoDateien prüft, dass im Foto-Ordner keine Dateien liegen, z. B. nach dem Löschen.
func (e *testEnv) keineFotoDateien() {
	e.t.Helper()
	dateien, err := os.ReadDir(filepath.Join(e.dataDir, store.FotoVerzeichnis))
	if err != nil {
		e.t.Fatal(err)
	}
	for _, d := range dateien {
		e.t.Errorf("Datei %s liegt noch im Foto-Ordner", d.Name())
	}
}
