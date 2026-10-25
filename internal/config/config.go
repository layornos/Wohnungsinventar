// Package config loads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const DefaultMaxPhotoSize = 20 << 20

type Config struct {
	DataDir string
	Addr    string
	// MaxPhotoSize is the largest allowed photo file in bytes.
	MaxPhotoSize int64
	// BehindProxy means a reverse proxy in front terminates TLS (ADR-0005): requests count as
	// HTTPS and X-Forwarded-Proto is trusted. Never set it when the server is reachable directly.
	BehindProxy bool
	// PublicURL is the address of the app as users open it, e.g. https://inventar.example.org, for links in mails.
	// Without it, mails contain no links. With DuckDNS it defaults to https://<domain>.
	PublicURL string
	DuckDNS   DuckDNS
	SMTP      SMTP
	Anthropic Anthropic
}

// Anthropic enables name and description suggestions from photos with Claude (ADR-0006) when
// the API key is set. The key must never be logged.
type Anthropic struct {
	APIKey string
	// Model is the Claude model; empty means the default of the suggest package.
	Model string
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

// WithSuggestions reports whether photos may be sent to Anthropic for suggestions.
func (c Config) WithSuggestions() bool { return c.Anthropic.APIKey != "" }

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
		Anthropic: Anthropic{
			APIKey: strings.TrimSpace(getenv("INVENTAR_ANTHROPIC_API_KEY")),
			Model:  strings.TrimSpace(getenv("INVENTAR_ANTHROPIC_MODEL")),
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

	if v := getenv("INVENTAR_BEHIND_PROXY"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("INVENTAR_BEHIND_PROXY is not a boolean (true/false): %q", v)
		}
		c.BehindProxy = b
	}
	if c.BehindProxy && (c.DuckDNS.Domain != "" || c.DuckDNS.Token != "") {
		return Config{}, errors.New("INVENTAR_BEHIND_PROXY cannot be combined with INVENTAR_DUCKDNS_DOMAIN/INVENTAR_DUCKDNS_TOKEN: " +
			"either the proxy terminates TLS or the server does it itself (ADR-0004, ADR-0005)")
	}
	if (c.DuckDNS.Domain == "") != (c.DuckDNS.Token == "") {
		return Config{}, errors.New("INVENTAR_DUCKDNS_DOMAIN and INVENTAR_DUCKDNS_TOKEN must be set together")
	}
	c.PublicURL = strings.TrimRight(strings.TrimSpace(getenv("INVENTAR_PUBLIC_URL")), "/")
	if c.PublicURL != "" && !strings.HasPrefix(c.PublicURL, "https://") && !strings.HasPrefix(c.PublicURL, "http://") {
		return Config{}, fmt.Errorf("INVENTAR_PUBLIC_URL must start with https:// or http://: %q", c.PublicURL)
	}
	if c.PublicURL == "" && c.WithTLS() {
		c.PublicURL = "https://" + c.DuckDNS.Domain
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
	if c.Anthropic.Model != "" && !c.WithSuggestions() {
		return Config{}, errors.New("INVENTAR_ANTHROPIC_MODEL is set, but INVENTAR_ANTHROPIC_API_KEY is missing")
	}
	return c, nil
}
