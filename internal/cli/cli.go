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

const hilfe = `Verwendung: inventar <befehl> [optionen]

Befehle:
  starten                                  Server starten
  nutzer-anlegen --name NAME --email EMAIL  Nutzer anlegen (Passwort wird abgefragt)
  passwort-zuruecksetzen --email EMAIL      Passwort neu setzen und alle Sitzungen beenden

Konfiguration über Umgebungsvariablen, siehe README.
`

// befehl bündelt, was jeder Unterbefehl braucht.
type befehl struct {
	cfg    config.Config
	args   []string
	stdin  io.Reader
	stdout io.Writer
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, hilfe)
		return errors.New("kein Befehl angegeben")
	}
	cfg, err := config.Laden(getenv)
	if err != nil {
		return err
	}
	b := befehl{cfg: cfg, args: args[1:], stdin: stdin, stdout: stdout}
	switch args[0] {
	case "starten":
		return starten(ctx, b)
	case "nutzer-anlegen":
		return nutzerAnlegen(ctx, b)
	case "passwort-zuruecksetzen":
		return passwortZuruecksetzen(ctx, b)
	case "help", "hilfe", "-h", "--help":
		fmt.Fprint(stdout, hilfe)
		return nil
	default:
		fmt.Fprint(stdout, hilfe)
		return fmt.Errorf("unbekannter Befehl %q", args[0])
	}
}

func nutzerAnlegen(ctx context.Context, b befehl) error {
	var name, email string
	err := b.optionenLesen("nutzer-anlegen", func(f *flag.FlagSet) {
		f.StringVar(&name, "name", "", "Name der Person")
		f.StringVar(&email, "email", "", "E-Mail-Adresse für die Anmeldung")
	}, &name, &email)
	if err != nil {
		return err
	}
	return b.mitPasswortUndStore(ctx, func(st *store.Store, hash string) error {
		n, err := st.NutzerAnlegen(ctx, strings.TrimSpace(name), email, hash, time.Now())
		if errors.Is(err, store.ErrEmailVergeben) {
			return fmt.Errorf("die E-Mail %s ist bereits vergeben", store.EmailNormalisieren(email))
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(b.stdout, "Nutzer %s <%s> angelegt.\n", n.Name, n.Email)
		return nil
	})
}

func passwortZuruecksetzen(ctx context.Context, b befehl) error {
	var email string
	err := b.optionenLesen("passwort-zuruecksetzen", func(f *flag.FlagSet) {
		f.StringVar(&email, "email", "", "E-Mail-Adresse des Nutzers")
	}, &email)
	if err != nil {
		return err
	}
	return b.mitPasswortUndStore(ctx, func(st *store.Store, hash string) error {
		err := st.PasswortSetzen(ctx, email, hash)
		if errors.Is(err, store.ErrNichtGefunden) {
			return fmt.Errorf("kein Nutzer mit der E-Mail %s", store.EmailNormalisieren(email))
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(b.stdout, "Passwort gesetzt, alle Sitzungen wurden beendet.")
		return nil
	})
}

// optionenLesen liest die Optionen eines Unterbefehls; alle übergebenen Pflichtwerte müssen gesetzt sein.
func (b befehl) optionenLesen(name string, definieren func(*flag.FlagSet), pflicht ...*string) error {
	optionen := flag.NewFlagSet(name, flag.ContinueOnError)
	optionen.SetOutput(b.stdout)
	definieren(optionen)
	if err := optionen.Parse(b.args); err != nil {
		return err
	}
	for _, p := range pflicht {
		if strings.TrimSpace(*p) == "" {
			optionen.Usage()
			return errors.New("Pflichtangabe fehlt")
		}
	}
	return nil
}

// mitPasswortUndStore liest das Passwort ein, öffnet den Store und ruft fn mit dem Passwort-Hash auf.
func (b befehl) mitPasswortUndStore(ctx context.Context, fn func(st *store.Store, hash string) error) error {
	hash, err := passwortHashEinlesen(b.stdin, b.stdout)
	if err != nil {
		return err
	}
	st, err := store.Open(ctx, b.cfg.Datenverzeichnis)
	if err != nil {
		return err
	}
	defer st.Close()
	return fn(st, hash)
}

// passwortHashEinlesen liest das Passwort verdeckt vom Terminal oder zeilenweise von stdin.
func passwortHashEinlesen(stdin io.Reader, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, "Passwort: ")
	var passwort string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		eingabe, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(stdout)
		if err != nil {
			return "", err
		}
		passwort = string(eingabe)
	} else {
		zeile, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		passwort = strings.TrimRight(zeile, "\r\n")
	}
	return auth.PasswortHashen(passwort)
}
