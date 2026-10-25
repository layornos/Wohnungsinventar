// Package reminder is the reminder service of the server process (spec 07, ADR-0001): it turns due schedules into
// reminders and sends the monthly mail on the 1st of every month.
package reminder

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"inventar/internal/euro"
	"inventar/internal/mail"
	"inventar/internal/store"
)

// Store is the part of data access the service needs.
type Store interface {
	TriggerDue(ctx context.Context, now time.Time) (int, error)
	PrepareMonthlyMails(ctx context.Context, month string) error
	PendingMonthlyMails(ctx context.Context) ([]store.MonthlyMail, error)
	MailOutcome(ctx context.Context, personID int64, month string, empty bool, sendErr error, now time.Time) (bool, error)
}

type Service struct {
	Store  Store
	Mailer mail.Mailer
	Now    func() time.Time
	// PublicURL is the address of the app for the link in the mail; empty means no link.
	PublicURL string
	Logger    *slog.Logger

	// Runs are serialized, so that nothing is triggered or sent twice.
	mu sync.Mutex
}

// Every is how often the server runs the service.
const Every = 15 * time.Minute

// Loop runs the service right away and then every interval until ctx ends.
func (s *Service) Loop(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if err := s.Run(ctx); err != nil && ctx.Err() == nil {
			s.Logger.Error("reminder run", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Run decides the monthly mails from 8:00 on the 1st of a month, triggers the due reminders and sends the mails,
// including those of a month the server was off for and those to try again after an error. The mail of a month is
// decided before triggering, so a reminder due on the 1st itself waits for the next month like any other.
func (s *Service) Run(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Now()
	local := now.In(store.Household)
	if local.Day() > 1 || local.Hour() >= 8 {
		if err := s.Store.PrepareMonthlyMails(ctx, local.Format("2006-01")); err != nil {
			return err
		}
	}
	if n, err := s.Store.TriggerDue(ctx, now); err != nil {
		return err
	} else if n > 0 {
		s.Logger.Info("reminders due", "count", n)
	}
	mails, err := s.Store.PendingMonthlyMails(ctx)
	if err != nil {
		return err
	}
	for _, m := range mails {
		if len(m.Reminders) == 0 {
			// Everything in it was settled in the app in the meantime.
			if _, err := s.Store.MailOutcome(ctx, m.To.PersonID, m.Month, true, nil, now); err != nil {
				return err
			}
			continue
		}
		sendErr := s.Mailer.Send(ctx, s.message(m))
		failed, err := s.Store.MailOutcome(ctx, m.To.PersonID, m.Month, false, sendErr, now)
		if err != nil {
			return err
		}
		switch {
		case failed:
			s.Logger.Error("monthly reminder mail failed for good", "to", m.To.Email, "month", m.Month, "error", sendErr)
		case sendErr != nil:
			s.Logger.Warn("monthly reminder mail failed, trying again later", "to", m.To.Email, "month", m.Month, "error", sendErr)
		default:
			s.Logger.Info("monthly reminder mail sent", "to", m.To.Email, "month", m.Month, "reminders", len(m.Reminders))
		}
	}
	return nil
}

// Line names a reminder in the mail, e.g. "Behalte-Prüfung: Schreibtisch (Sandro, Anna)" or, with the asking
// price, "Preis prüfen: Stehlampe (40 €, Sandro)".
func Line(r store.Reminder) string {
	var details []string
	if r.Kind == store.PriceReminder && r.Item.AskingPrice != nil {
		details = append(details, euro.Format(*r.Item.AskingPrice))
	}
	for _, o := range r.Item.Owners {
		details = append(details, o.Name)
	}
	return fmt.Sprintf("%s: %s (%s)", KindName(r.Kind), r.Item.Name, strings.Join(details, ", "))
}

// KindName is the German name of a kind of reminder.
func KindName(k store.ReminderKind) string {
	switch k {
	case store.KeepReview:
		return "Behalte-Prüfung"
	case store.SellLaterDue:
		return "Später verkaufen"
	case store.PriceReminder:
		return "Preis prüfen"
	case store.GiveAwayReminder:
		return "Noch zu verschenken"
	}
	return string(k)
}

func (s *Service) message(m store.MonthlyMail) mail.Message {
	subject := "Inventar: 1 offene Erinnerung"
	if n := len(m.Reminders); n != 1 {
		subject = fmt.Sprintf("Inventar: %d offene Erinnerungen", n)
	}
	var b strings.Builder
	for _, r := range m.Reminders {
		b.WriteString(Line(r) + "\n")
	}
	b.WriteString("\n")
	if s.PublicURL != "" {
		b.WriteString("Zu den Erinnerungen: " + s.PublicURL + "/erinnerungen\n")
	} else {
		b.WriteString("In der App unter Erinnerungen.\n")
	}
	return mail.Message{To: m.To.Email, Subject: subject, Text: b.String()}
}
