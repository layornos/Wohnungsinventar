// Package config loads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
)

const DefaultMaxPhotoSize = 20 << 20

type Config struct {
	DataDir string
	Addr    string
	// MaxPhotoSize is the largest allowed photo file in bytes.
	MaxPhotoSize int64
	DuckDNS      DuckDNS
	SMTP         SMTP
}

// DuckDNS enables HTTPS with a certificate via DNS-01 (ADR-0004) when domain and token are set.
type DuckDNS struct {
	Domain string
	Token  string
}

// SMTP enables real mail delivery; without a host, mails only end up in the log.
type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (c Config) WithTLS() bool  { return c.DuckDNS.Domain != "" }
func (c Config) WithSMTP() bool { return c.SMTP.Host != "" }

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
	c.MaxPhotoSize = DefaultMaxPhotoSize
	if mb := getenv("INVENTAR_MAX_PHOTO_MB"); mb != "" {
		n, err := strconv.Atoi(mb)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("INVENTAR_MAX_PHOTO_MB is not a positive number: %q", mb)
		}
		c.MaxPhotoSize = int64(n) << 20
	}

	if (c.DuckDNS.Domain == "") != (c.DuckDNS.Token == "") {
		return Config{}, errors.New("INVENTAR_DUCKDNS_DOMAIN and INVENTAR_DUCKDNS_TOKEN must be set together")
	}
	if c.Addr == "" {
		if c.WithTLS() {
			c.Addr = ":443"
		} else {
			c.Addr = "localhost:8080"
		}
	}

	port := getenv("INVENTAR_SMTP_PORT")
	if port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("INVENTAR_SMTP_PORT is not a number: %q", port)
		}
		c.SMTP.Port = p
	}
	if c.WithSMTP() {
		if c.SMTP.User == "" || c.SMTP.Password == "" {
			return Config{}, errors.New("INVENTAR_SMTP_HOST is set, but INVENTAR_SMTP_USER or INVENTAR_SMTP_PASSWORD is missing")
		}
		if c.SMTP.Port == 0 {
			c.SMTP.Port = 587
		}
		if c.SMTP.From == "" {
			c.SMTP.From = c.SMTP.User
		}
	} else if c.SMTP.User != "" || c.SMTP.Password != "" || c.SMTP.From != "" || port != "" {
		return Config{}, errors.New("SMTP settings are set, but INVENTAR_SMTP_HOST is missing")
	}
	return c, nil
}
