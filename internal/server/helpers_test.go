package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	"time"

	"inventar/internal/cli"
	"inventar/internal/config"
	"inventar/internal/mail"
	"inventar/internal/reminder"
	"inventar/internal/server"
	"inventar/internal/store"
)

// testEnv is a real server with a real SQLite file in a temporary data directory.
type testEnv struct {
	t       *testing.T
	dataDir string
	mailer  *mail.Recorder
	srv     *httptest.Server
	// reminders is the reminder service on the same store, mailer and clock; tests run it directly.
	reminders *reminder.Service
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWith(t, func(*server.Deps) {})
}

// newTestEnvWith starts the server with adjusted dependencies, e.g. a client of its own.
func newTestEnvWith(t *testing.T, adjust func(*server.Deps)) *testEnv {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	rec := &mail.Recorder{}
	d := server.Deps{
		Config: config.Config{DataDir: dataDir},
		Store:  st,
		Mailer: rec,
		Client: fstest.MapFS{},
	}
	adjust(&d)
	srv := httptest.NewServer(server.New(d))
	t.Cleanup(srv.Close)
	now := d.Now
	if now == nil {
		now = time.Now
	}
	svc := &reminder.Service{Store: st, Mailer: d.Mailer, Now: now, PublicURL: d.Config.PublicURL, Logger: slog.New(slog.DiscardHandler)}
	return &testEnv{t: t, dataDir: dataDir, mailer: rec, srv: srv, reminders: svc}
}

// runReminders runs the reminder service once, as the server does every 15 minutes.
func (e *testEnv) runReminders() {
	e.t.Helper()
	if err := e.reminders.Run(context.Background()); err != nil {
		e.t.Fatalf("reminder run: %v", err)
	}
}

// runCLI runs a subcommand against the same data directory.
func (e *testEnv) runCLI(stdin string, args ...string) (string, error) {
	e.t.Helper()
	var out bytes.Buffer
	env := map[string]string{"INVENTAR_DATA_DIR": e.dataDir}
	err := cli.Run(context.Background(), args, strings.NewReader(stdin), &out, func(k string) string { return env[k] })
	return out.String(), err
}

func (e *testEnv) createUser(name, email, password string) {
	e.t.Helper()
	if out, err := e.runCLI(password+"\n", "create-user", "--name", name, "--email", email); err != nil {
		e.t.Fatalf("create-user: %v\n%s", err, out)
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

// loginAs logs in a new client with the given credentials and returns it with the login response.
func (e *testEnv) loginAs(email, password string) (*apiClient, response) {
	e.t.Helper()
	c := e.client()
	res := c.mustStatus(http.StatusOK, "POST", "/api/login", map[string]any{"email": email, "password": password})
	return c, res
}

// loggedIn creates a user and returns a logged-in client.
func (e *testEnv) loggedIn() *apiClient {
	e.t.Helper()
	e.createUser("Alex", "alex@example.org", "geheim-geheim")
	c, _ := e.loginAs("alex@example.org", "geheim-geheim")
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

// form is the content of a multipart request: text fields (also repeated) and photos.
type form struct {
	fields [][2]string
	photos [][]byte
}

func (c *apiClient) multipart(method, path string, f form) response {
	c.t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, field := range f.fields {
		w.WriteField(field[0], field[1])
	}
	for i, p := range f.photos {
		part, err := w.CreateFormFile("photo", fmt.Sprintf("photo%d.jpg", i))
		if err != nil {
			c.t.Fatal(err)
		}
		part.Write(p)
	}
	w.Close()
	req, err := http.NewRequest(method, c.base+path, &b)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.send(req)
}

// createItem creates an item with the owners and photos and returns its representation.
func (c *apiClient) createItem(name string, owners []int64, photos ...[]byte) response {
	c.t.Helper()
	f := form{fields: [][2]string{{"name", name}}, photos: photos}
	for _, id := range owners {
		f.fields = append(f.fields, [2]string{"owner", fmt.Sprint(id)})
	}
	res := c.multipart("POST", "/api/items", f)
	if res.Status != http.StatusCreated {
		c.t.Fatalf("create item: status %d: %s", res.Status, res.Raw)
	}
	return res
}

// get fetches an address, e.g. a photo, and returns the raw response.
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
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.Status, want, res.Raw)
	}
	return res
}

// me returns the person ID of the logged-in user.
func (c *apiClient) me() int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusOK, "GET", "/api/me", nil).Body["personId"].(float64))
}

func (c *apiClient) createPerson(name string) int64 {
	c.t.Helper()
	return int64(c.mustStatus(http.StatusCreated, "POST", "/api/persons", map[string]any{"name": name}).Body["id"].(float64))
}

// errorCode returns the domain error code of an error response.
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

// noPhotoFiles checks that there are no files in the photo folder, e.g. after deleting.
func (e *testEnv) noPhotoFiles() {
	e.t.Helper()
	files, err := os.ReadDir(filepath.Join(e.dataDir, store.PhotoDir))
	if err != nil {
		e.t.Fatal(err)
	}
	for _, f := range files {
		e.t.Errorf("file %s is still in the photo folder", f.Name())
	}
}

// noOrphanedPhotos checks that the photo folder only holds the files of the created photos
// (original and thumbnail per photo).
func (e *testEnv) noOrphanedPhotos(photos int) {
	e.t.Helper()
	files, err := os.ReadDir(filepath.Join(e.dataDir, store.PhotoDir))
	if err != nil {
		e.t.Fatal(err)
	}
	if len(files) != 2*photos {
		e.t.Errorf("%d files in the photo folder, want %d", len(files), 2*photos)
	}
}
