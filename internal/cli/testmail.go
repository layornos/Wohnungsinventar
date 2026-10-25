package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"inventar/internal/mail"
)

// testMail sends a mail through the configured mailer, to check the SMTP settings (e.g. the Gmail app password).
func testMail(ctx context.Context, c command) error {
	var to string
	err := c.parseFlags("test-mail", func(f *flag.FlagSet) {
		f.StringVar(&to, "to", "", "address to send the test mail to")
	}, &to)
	if err != nil {
		return err
	}
	if !c.cfg.WithSMTP() {
		fmt.Fprintln(c.stdout, "SMTP is not configured (INVENTAR_SMTP_HOST); the mail only goes to the log.")
	}
	m := mail.Message{
		To:      strings.TrimSpace(to),
		Subject: "Inventar: Test-Mail",
		Text:    "Diese Mail kommt vom Inventar-Server. Wenn sie ankommt, funktioniert der Mailversand.\n",
	}
	if err := mail.New(c.cfg, slog.New(slog.NewTextHandler(c.stdout, nil))).Send(ctx, m); err != nil {
		return fmt.Errorf("sending the test mail failed: %w", err)
	}
	fmt.Fprintf(c.stdout, "Test mail sent to %s.\n", m.To)
	return nil
}
