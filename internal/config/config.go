// Package config lädt die Server-Konfiguration aus Umgebungsvariablen.
package config

import (
	"errors"
	"fmt"
	"strconv"
)

const StandardMaxFotoGroesse = 20 << 20

type Config struct {
	Datenverzeichnis string
	Adresse          string
	// MaxFotoGroesse ist die größte erlaubte Fotodatei in Bytes.
	MaxFotoGroesse int64
	DuckDNS        DuckDNS
	SMTP           SMTP
}

// DuckDNS aktiviert HTTPS mit Zertifikat per DNS-01 (ADR-0004), wenn Domain und Token gesetzt sind.
type DuckDNS struct {
	Domain string
	Token  string
}

// SMTP aktiviert echten Mailversand; ohne Host landen Mails nur im Log.
type SMTP struct {
	Host     string
	Port     int
	Nutzer   string
	Passwort string
	Absender string
}

func (c Config) MitTLS() bool  { return c.DuckDNS.Domain != "" }
func (c Config) MitSMTP() bool { return c.SMTP.Host != "" }

func Laden(getenv func(string) string) (Config, error) {
	c := Config{
		Datenverzeichnis: getenv("INVENTAR_DATA_DIR"),
		Adresse:          getenv("INVENTAR_ADDR"),
		DuckDNS: DuckDNS{
			Domain: getenv("INVENTAR_DUCKDNS_DOMAIN"),
			Token:  getenv("INVENTAR_DUCKDNS_TOKEN"),
		},
		SMTP: SMTP{
			Host:     getenv("INVENTAR_SMTP_HOST"),
			Nutzer:   getenv("INVENTAR_SMTP_USER"),
			Passwort: getenv("INVENTAR_SMTP_PASSWORD"),
			Absender: getenv("INVENTAR_SMTP_FROM"),
		},
	}
	if c.Datenverzeichnis == "" {
		c.Datenverzeichnis = "data"
	}
	c.MaxFotoGroesse = StandardMaxFotoGroesse
	if mb := getenv("INVENTAR_MAX_FOTO_MB"); mb != "" {
		n, err := strconv.Atoi(mb)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("INVENTAR_MAX_FOTO_MB ist keine positive Zahl: %q", mb)
		}
		c.MaxFotoGroesse = int64(n) << 20
	}

	if (c.DuckDNS.Domain == "") != (c.DuckDNS.Token == "") {
		return Config{}, errors.New("INVENTAR_DUCKDNS_DOMAIN und INVENTAR_DUCKDNS_TOKEN müssen zusammen gesetzt sein")
	}
	if c.Adresse == "" {
		if c.MitTLS() {
			c.Adresse = ":443"
		} else {
			c.Adresse = "localhost:8080"
		}
	}

	port := getenv("INVENTAR_SMTP_PORT")
	if port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("INVENTAR_SMTP_PORT ist keine Zahl: %q", port)
		}
		c.SMTP.Port = p
	}
	if c.MitSMTP() {
		if c.SMTP.Nutzer == "" || c.SMTP.Passwort == "" {
			return Config{}, errors.New("INVENTAR_SMTP_HOST gesetzt, aber INVENTAR_SMTP_USER oder INVENTAR_SMTP_PASSWORD fehlt")
		}
		if c.SMTP.Port == 0 {
			c.SMTP.Port = 587
		}
		if c.SMTP.Absender == "" {
			c.SMTP.Absender = c.SMTP.Nutzer
		}
	} else if c.SMTP.Nutzer != "" || c.SMTP.Passwort != "" || c.SMTP.Absender != "" || port != "" {
		return Config{}, errors.New("SMTP-Angaben gesetzt, aber INVENTAR_SMTP_HOST fehlt")
	}
	return c, nil
}
