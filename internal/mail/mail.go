// Package mail sends mails via SMTP or writes them to the log when SMTP is not configured.
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

type Message struct {
	To      string
	Subject string
	Text    string
}

type Mailer interface {
	Send(ctx context.Context, m Message) error
}

func New(cfg config.Config, log *slog.Logger) Mailer {
	if cfg.WithSMTP() {
		return &SMTP{cfg: cfg.SMTP}
	}
	return &Log{Logger: log}
}

// Log only writes mails to the log.
type Log struct{ Logger *slog.Logger }

func (l *Log) Send(_ context.Context, m Message) error {
	l.Logger.Info("mail not sent (SMTP not configured)", "to", m.To, "subject", m.Subject, "text", m.Text)
	return nil
}

// SMTP sends via STARTTLS, e.g. through Gmail with an app password.
type SMTP struct{ cfg config.SMTP }

func (s *SMTP) Send(_ context.Context, m Message) error {
	addr := s.cfg.Host + ":" + strconv.Itoa(s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", s.cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", m.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", m.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(m.Text, "\n", "\r\n"))
	return smtp.SendMail(addr, auth, s.cfg.From, []string{m.To}, []byte(b.String()))
}

// Recorder keeps mails instead of sending them; for tests.
type Recorder struct {
	mu   sync.Mutex
	sent []Message
}

func (r *Recorder) Send(_ context.Context, m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, m)
	return nil
}

func (r *Recorder) Sent() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.sent...)
}
