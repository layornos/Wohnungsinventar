package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"inventar/internal/cli"
	"inventar/internal/store"
)

// dataDirWithUser creates a data directory with one user and one photo file.
func dataDirWithUser(t *testing.T) (dataDir string, env map[string]string) {
	t.Helper()
	dataDir = filepath.Join(t.TempDir(), "data")
	env = map[string]string{"INVENTAR_DATA_DIR": dataDir}
	var out bytes.Buffer
	err := cli.Run(context.Background(), []string{"create-user", "--name", "Alex", "--email", "alex@example.org"},
		strings.NewReader("geheim-geheim\n"), &out, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("create-user: %v\n%s", err, out.String())
	}
	writeFile(t, filepath.Join(dataDir, store.PhotoDir, "a.jpg"), "photo a")
	return dataDir, env
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBackupContainsDatabaseAndPhotos(t *testing.T) {
	_, env := dataDirWithUser(t)
	to := filepath.Join(t.TempDir(), "backup")

	out, err := run(env, "backup", "--to", to)
	if err != nil {
		t.Fatalf("backup: %v\n%s", err, out)
	}
	if !strings.Contains(out, "1 photo") {
		t.Errorf("summary without photo count:\n%s", out)
	}
	if got := readFile(t, filepath.Join(to, store.PhotoDir, "a.jpg")); got != "photo a" {
		t.Errorf("photo in backup: %q", got)
	}

	// The backup folder is laid out like a data directory and can be opened as one.
	st, err := store.Open(context.Background(), to)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer st.Close()
	if u, err := st.UserByEmail(context.Background(), "alex@example.org"); err != nil || u.Name != "Alex" {
		t.Fatalf("user in snapshot: %+v, %v", u, err)
	}
}

func TestRepeatedBackupOnlyCopiesNewAndChangedPhotos(t *testing.T) {
	dataDir, env := dataDirWithUser(t)
	to := filepath.Join(t.TempDir(), "backup")
	if out, err := run(env, "backup", "--to", to); err != nil {
		t.Fatalf("first backup: %v\n%s", err, out)
	}

	photos := filepath.Join(dataDir, store.PhotoDir)
	writeFile(t, filepath.Join(photos, "b.jpg"), "photo b")
	writeFile(t, filepath.Join(photos, "a.jpg"), "photo a, replaced")
	// Make sure the change is visible even on file systems with coarse timestamps.
	later := time.Now().Add(time.Minute)
	os.Chtimes(filepath.Join(photos, "a.jpg"), later, later)
	writeFile(t, filepath.Join(photos, "c.jpg"), "photo c")
	if out, err := run(env, "backup", "--to", to); err != nil {
		t.Fatalf("second backup: %v\n%s", err, out)
	}
	os.Remove(filepath.Join(photos, "c.jpg"))

	out, err := run(env, "backup", "--to", to)
	if err != nil {
		t.Fatalf("third backup: %v\n%s", err, out)
	}
	if !strings.Contains(out, "0 photos copied") {
		t.Errorf("unchanged photos were copied again:\n%s", out)
	}
	backup := filepath.Join(to, store.PhotoDir)
	if got := readFile(t, filepath.Join(backup, "a.jpg")); got != "photo a, replaced" {
		t.Errorf("changed photo: %q", got)
	}
	if got := readFile(t, filepath.Join(backup, "b.jpg")); got != "photo b" {
		t.Errorf("new photo: %q", got)
	}
	// Deleted photos may stay in the backup.
	if got := readFile(t, filepath.Join(backup, "c.jpg")); got != "photo c" {
		t.Errorf("deleted photo: %q", got)
	}
}

func TestBackupNeedsATargetOutsideTheDataDir(t *testing.T) {
	dataDir, env := dataDirWithUser(t)
	if _, err := run(env, "backup"); err == nil {
		t.Error("backup without --to succeeded")
	}
	if _, err := run(env, "backup", "--to", dataDir); err == nil {
		t.Error("backup into the data directory succeeded")
	}
}
