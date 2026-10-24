package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"inventar/internal/cli"
)

func run(env map[string]string, args ...string) (string, error) {
	var out bytes.Buffer
	err := cli.Run(context.Background(), args, strings.NewReader(""), &out, func(k string) string { return env[k] })
	return out.String(), err
}

func TestIncompleteConfigurationPreventsStart(t *testing.T) {
	cases := map[string]map[string]string{
		"DuckDNS domain without token": {"INVENTAR_DUCKDNS_DOMAIN": "inventar.duckdns.org"},
		"DuckDNS token without domain": {"INVENTAR_DUCKDNS_TOKEN": "abc"},
		"SMTP host without password":   {"INVENTAR_SMTP_HOST": "smtp.gmail.com", "INVENTAR_SMTP_USER": "a@gmail.com"},
		"SMTP user without host":       {"INVENTAR_SMTP_USER": "a@gmail.com", "INVENTAR_SMTP_PASSWORD": "x"},
		"SMTP port not a number":       {"INVENTAR_SMTP_PORT": "abc"},
		"SMTP port without host":       {"INVENTAR_SMTP_PORT": "587"},
		"photo size not a number":      {"INVENTAR_MAX_PHOTO_MB": "lots"},
		"photo size zero":              {"INVENTAR_MAX_PHOTO_MB": "0"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			env["INVENTAR_DATA_DIR"] = t.TempDir()
			if _, err := run(env, "help"); err == nil {
				t.Fatal("server started despite incomplete configuration")
			}
		})
	}
}

func TestUnknownCommandShowsUsage(t *testing.T) {
	out, err := run(map[string]string{"INVENTAR_DATA_DIR": t.TempDir()}, "does-not-exist")
	if err == nil || !strings.Contains(out, "create-user") {
		t.Fatalf("err=%v, output:\n%s", err, out)
	}
}

func TestDataDirContainsPhotoFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	env := map[string]string{"INVENTAR_DATA_DIR": dir}
	var out bytes.Buffer
	err := cli.Run(context.Background(), []string{"create-user", "--name", "Alex", "--email", "alex@example.org"},
		strings.NewReader("geheim-geheim\n"), &out, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("create-user: %v\n%s", err, out.String())
	}
	if st, err := os.Stat(filepath.Join(dir, "fotos")); err != nil || !st.IsDir() {
		t.Fatalf("photo folder missing: %v", err)
	}
}
