package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"inventar/internal/cli"
)

func run(env map[string]string, args ...string) (string, error) {
	var out bytes.Buffer
	err := cli.Run(context.Background(), args, strings.NewReader(""), &out, func(k string) string { return env[k] })
	return out.String(), err
}

func TestUnvollstaendigeKonfigurationVerhindertDenStart(t *testing.T) {
	cases := map[string]map[string]string{
		"DuckDNS-Domain ohne Token": {"INVENTAR_DUCKDNS_DOMAIN": "inventar.duckdns.org"},
		"DuckDNS-Token ohne Domain": {"INVENTAR_DUCKDNS_TOKEN": "abc"},
		"SMTP-Host ohne Passwort":   {"INVENTAR_SMTP_HOST": "smtp.gmail.com", "INVENTAR_SMTP_USER": "a@gmail.com"},
		"SMTP-Nutzer ohne Host":     {"INVENTAR_SMTP_USER": "a@gmail.com", "INVENTAR_SMTP_PASSWORD": "x"},
		"SMTP-Port keine Zahl":      {"INVENTAR_SMTP_PORT": "abc"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			env["INVENTAR_DATA_DIR"] = t.TempDir()
			if _, err := run(env, "serve"); err == nil {
				t.Fatal("Server startete trotz unvollständiger Konfiguration")
			}
		})
	}
}

func TestUnbekannterBefehlZeigtHilfe(t *testing.T) {
	out, err := run(map[string]string{"INVENTAR_DATA_DIR": t.TempDir()}, "gibt-es-nicht")
	if err == nil || !strings.Contains(out, "nutzer-anlegen") {
		t.Fatalf("err=%v, Ausgabe:\n%s", err, out)
	}
}
