package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email taken")
)

type User struct {
	PersonID     int64
	Name         string
	Email        string
	PasswordHash string
	Theme        Theme
	// ReminderMail says whether the user gets the monthly reminder mail (spec 07).
	ReminderMail bool
}

// Theme is the look of the UI a user has chosen; NoTheme means none yet.
type Theme string

const (
	NoTheme      Theme = ""
	ThemeMinimal Theme = "minimal"
	ThemeNeon    Theme = "neon"
	ThemeSoft    Theme = "soft"
)

// Selectable says whether a user can choose this theme.
func (t Theme) Selectable() bool {
	switch t {
	case ThemeMinimal, ThemeNeon, ThemeSoft:
		return true
	}
	return false
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// CreateUser creates a new person together with user access.
func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash string, now time.Time) (User, error) {
	u := User{Name: name, Email: NormalizeEmail(email), PasswordHash: passwordHash}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var existing int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_user WHERE email = ?`, u.Email).Scan(&existing); err != nil {
			return err
		}
		if existing > 0 {
			return ErrEmailTaken
		}
		err := tx.QueryRowContext(ctx, `INSERT INTO person (name, created_at) VALUES (?, ?) RETURNING id`,
			name, timeText(now)).Scan(&u.PersonID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO app_user (person_id, email, password_hash) VALUES (?, ?, ?)`, u.PersonID, u.Email, passwordHash)
		return err
	})
	if err != nil {
		return User{}, err
	}
	return u, nil
}

const userSelect = `SELECT u.person_id, p.name, u.email, u.password_hash, u.theme, u.reminder_mail = 1 FROM app_user u JOIN person p ON p.id = u.person_id`

func readUser(r row) (User, error) {
	var u User
	var theme sql.NullString
	err := r.Scan(&u.PersonID, &u.Name, &u.Email, &u.PasswordHash, &theme, &u.ReminderMail)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	u.Theme = Theme(theme.String)
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return readUser(s.db.QueryRowContext(ctx, userSelect+` WHERE u.email = ?`, NormalizeEmail(email)))
}

// SetTheme stores the chosen theme on the user and returns the user with the new setting.
func (s *Store) SetTheme(ctx context.Context, personID int64, t Theme) (User, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE app_user SET theme = ? WHERE person_id = ?`, string(t), personID); err != nil {
		return User{}, err
	}
	return readUser(s.db.QueryRowContext(ctx, userSelect+` WHERE u.person_id = ?`, personID))
}

// SetPassword replaces the password and ends all sessions of the user.
func (s *Store) SetPassword(ctx context.Context, email, passwordHash string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT person_id FROM app_user WHERE email = ?`, NormalizeEmail(email)).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE app_user SET password_hash = ? WHERE person_id = ?`, passwordHash, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM session WHERE person_id = ?`, id)
		return err
	})
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, personID int64, now, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO session (token_hash, person_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, personID, timeText(now), timeText(expiresAt))
	return err
}

// SessionUser returns the user of a session that is still valid.
func (s *Store) SessionUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	return readUser(s.db.QueryRowContext(ctx,
		userSelect+` JOIN session s ON s.person_id = u.person_id WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash, timeText(now)))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session WHERE token_hash = ?`, tokenHash)
	return err
}
