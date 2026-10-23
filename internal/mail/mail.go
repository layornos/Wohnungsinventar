// Package mail verschickt Mails per SMTP oder schreibt sie ins Log, wenn SMTP nicht konfiguriert ist.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"inventar/internal/config"
)

type Nachricht struct {
	An      string
	Betreff string
	Text    string
}

type Mailer interface {
	Senden(ctx context.Context, n Nachricht) error
}

func Neu(cfg config.Config, log *slog.Logger) Mailer {
	if cfg.MitSMTP() {
		return &SMTP{cfg: cfg.SMTP}
	}
	return &Log{Logger: log}
}

// Log schreibt Mails nur ins Log.
type Log struct{ Logger *slog.Logger }

func (l *Log) Senden(_ context.Context, n Nachricht) error {
	l.Logger.Info("Mail nicht versendet (SMTP nicht konfiguriert)", "an", n.An, "betreff", n.Betreff, "text", n.Text)
	return nil
}

// SMTP versendet per STARTTLS, z. B. über Gmail mit App-Passwort.
type SMTP struct{ cfg config.SMTP }

func (s *SMTP) Senden(_ context.Context, n Nachricht) error {
	adresse := s.cfg.Host + ":" + strconv.Itoa(s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Nutzer, s.cfg.Passwort, s.cfg.Host)
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", s.cfg.Absender)
	fmt.Fprintf(&b, "To: %s\r\n", n.An)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", n.Betreff))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(n.Text, "\n", "\r\n"))
	return smtp.SendMail(adresse, auth, s.cfg.Absender, []string{n.An}, []byte(b.String()))
}

// Aufzeichner merkt sich Mails, statt sie zu senden; für Tests.
type Aufzeichner struct {
	mu        sync.Mutex
	gesendete []Nachricht
}

func (a *Aufzeichner) Senden(_ context.Context, n Nachricht) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.gesendete = append(a.gesendete, n)
	return nil
}

func (a *Aufzeichner) Gesendete() []Nachricht {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Nachricht(nil), a.gesendete...)
}
