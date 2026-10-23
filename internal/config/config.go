// Package config lädt die Server-Konfiguration aus Umgebungsvariablen.
package config

import (
	"errors"
	"fmt"
	"strconv"
)

type Config struct {
	DataDir string
	Addr    string
	DuckDNS DuckDNS
	SMTP    SMTP
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
	User     string
	Password string
	From     string
}

func (c Config) TLSEnabled() bool  { return c.DuckDNS.Domain != "" }
func (c Config) SMTPEnabled() bool { return c.SMTP.Host != "" }

func Load(getenv func(string) string) (Config, error) {
	c := Config{
		DataDir: getenv("INVENTAR_DATA_DIR"),
		Addr:    getenv("INVENTAR_ADDR"),
		DuckDNS: DuckDNS{
			Domain: getenv("INVENTAR_DUCKDNS_DOMAIN"),
			Token:  getenv("INVENTAR_DUCKDNS_TOKEN"),
		},
		SMTP: SMTP{
			Host:     getenv("INVENTAR_SMTP_HOST"),
			User:     getenv("INVENTAR_SMTP_USER"),
			Password: getenv("INVENTAR_SMTP_PASSWORD"),
			From:     getenv("INVENTAR_SMTP_FROM"),
		},
	}
	if c.DataDir == "" {
		c.DataDir = "data"
	}

	if (c.DuckDNS.Domain == "") != (c.DuckDNS.Token == "") {
		return Config{}, errors.New("INVENTAR_DUCKDNS_DOMAIN und INVENTAR_DUCKDNS_TOKEN müssen zusammen gesetzt sein")
	}
	if c.Addr == "" {
		if c.TLSEnabled() {
			c.Addr = ":443"
		} else {
			c.Addr = "localhost:8080"
		}
	}

	if port := getenv("INVENTAR_SMTP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("INVENTAR_SMTP_PORT ist keine Zahl: %q", port)
		}
		c.SMTP.Port = p
	}
	if c.SMTPEnabled() {
		if c.SMTP.User == "" || c.SMTP.Password == "" {
			return Config{}, errors.New("INVENTAR_SMTP_HOST gesetzt, aber INVENTAR_SMTP_USER oder INVENTAR_SMTP_PASSWORD fehlt")
		}
		if c.SMTP.Port == 0 {
			c.SMTP.Port = 587
		}
		if c.SMTP.From == "" {
			c.SMTP.From = c.SMTP.User
		}
	} else if c.SMTP.User != "" || c.SMTP.Password != "" || c.SMTP.From != "" {
		return Config{}, errors.New("SMTP-Angaben gesetzt, aber INVENTAR_SMTP_HOST fehlt")
	}
	return c, nil
}
