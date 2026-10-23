// Package cli enthält die Unterbefehle des Binarys.
package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"inventar/internal/auth"
	"inventar/internal/config"
	"inventar/internal/store"
)

const usage = `Verwendung: inventar <befehl> [optionen]

Befehle:
  serve                                    Server starten
  nutzer-anlegen --name NAME --email EMAIL  Nutzer anlegen (Passwort wird abgefragt)
  passwort-zuruecksetzen --email EMAIL      Passwort neu setzen und alle Sitzungen beenden

Konfiguration über Umgebungsvariablen, siehe README.
`

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return errors.New("kein Befehl angegeben")
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}
	switch args[0] {
	case "serve":
		return serve(ctx, cfg, stdout)
	case "nutzer-anlegen":
		return nutzerAnlegen(ctx, cfg, args[1:], stdin, stdout)
	case "passwort-zuruecksetzen":
		return passwortZuruecksetzen(ctx, cfg, args[1:], stdin, stdout)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		fmt.Fprint(stdout, usage)
		return fmt.Errorf("unbekannter Befehl %q", args[0])
	}
}

func nutzerAnlegen(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("nutzer-anlegen", flag.ContinueOnError)
	fs.SetOutput(stdout)
	name := fs.String("name", "", "Name der Person")
	email := fs.String("email", "", "E-Mail-Adresse für die Anmeldung")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*email) == "" {
		return errors.New("--name und --email sind Pflicht")
	}
	hash, err := passwortHashEinlesen(stdin, stdout)
	if err != nil {
		return err
	}
	st, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	n, err := st.CreateNutzer(ctx, strings.TrimSpace(*name), *email, hash, time.Now())
	if errors.Is(err, store.ErrEmailVergeben) {
		return fmt.Errorf("die E-Mail %s ist bereits vergeben", store.NormalizeEmail(*email))
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Nutzer %s <%s> angelegt.\n", n.Name, n.Email)
	return nil
}

func passwortZuruecksetzen(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("passwort-zuruecksetzen", flag.ContinueOnError)
	fs.SetOutput(stdout)
	email := fs.String("email", "", "E-Mail-Adresse des Nutzers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" {
		return errors.New("--email ist Pflicht")
	}
	hash, err := passwortHashEinlesen(stdin, stdout)
	if err != nil {
		return err
	}
	st, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	err = st.SetPasswort(ctx, *email, hash)
	if errors.Is(err, store.ErrNichtGefunden) {
		return fmt.Errorf("kein Nutzer mit der E-Mail %s", store.NormalizeEmail(*email))
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Passwort gesetzt, alle Sitzungen wurden beendet.")
	return nil
}

// passwortHashEinlesen liest das Passwort verdeckt vom Terminal oder zeilenweise von stdin.
func passwortHashEinlesen(stdin io.Reader, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, "Passwort: ")
	var pw string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(stdout)
		if err != nil {
			return "", err
		}
		pw = string(b)
	} else {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		pw = strings.TrimRight(line, "\r\n")
	}
	return auth.HashPasswort(pw)
}
