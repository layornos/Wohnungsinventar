package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNichtGefunden = errors.New("nicht gefunden")
	ErrEmailVergeben = errors.New("E-Mail vergeben")
)

type Nutzer struct {
	PersonID     int64
	Name         string
	Email        string
	PasswortHash string
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// CreateNutzer legt eine neue Person samt Nutzerzugang an.
func (s *Store) CreateNutzer(ctx context.Context, name, email, passwortHash string, now time.Time) (Nutzer, error) {
	n := Nutzer{Name: name, Email: NormalizeEmail(email), PasswortHash: passwortHash}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM nutzer WHERE email = ?`, n.Email).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			return ErrEmailVergeben
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO person (name, erstellt_am) VALUES (?, ?)`, name, fmtTime(now))
		if err != nil {
			return err
		}
		if n.PersonID, err = res.LastInsertId(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO nutzer (person_id, email, passwort_hash) VALUES (?, ?, ?)`, n.PersonID, n.Email, passwortHash)
		return err
	})
	if err != nil {
		return Nutzer{}, err
	}
	return n, nil
}

const nutzerSelect = `SELECT n.person_id, p.name, n.email, n.passwort_hash FROM nutzer n JOIN person p ON p.id = n.person_id`

func scanNutzer(row *sql.Row) (Nutzer, error) {
	var n Nutzer
	err := row.Scan(&n.PersonID, &n.Name, &n.Email, &n.PasswortHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Nutzer{}, ErrNichtGefunden
	}
	return n, err
}

func (s *Store) NutzerByEmail(ctx context.Context, email string) (Nutzer, error) {
	return scanNutzer(s.db.QueryRowContext(ctx, nutzerSelect+` WHERE n.email = ?`, NormalizeEmail(email)))
}

// SetPasswort ersetzt das Passwort und beendet alle Sitzungen des Nutzers.
func (s *Store) SetPasswort(ctx context.Context, email, passwortHash string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT person_id FROM nutzer WHERE email = ?`, NormalizeEmail(email)).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNichtGefunden
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE nutzer SET passwort_hash = ? WHERE person_id = ?`, passwortHash, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sitzung WHERE person_id = ?`, id)
		return err
	})
}

func (s *Store) CreateSitzung(ctx context.Context, tokenHash string, personID int64, now, laeuftAb time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sitzung (token_hash, person_id, erstellt_am, laeuft_ab_am) VALUES (?, ?, ?, ?)`,
		tokenHash, personID, fmtTime(now), fmtTime(laeuftAb))
	return err
}

// SitzungNutzer liefert den Nutzer einer noch gültigen Sitzung.
func (s *Store) SitzungNutzer(ctx context.Context, tokenHash string, now time.Time) (Nutzer, error) {
	return scanNutzer(s.db.QueryRowContext(ctx,
		nutzerSelect+` JOIN sitzung s ON s.person_id = n.person_id WHERE s.token_hash = ? AND s.laeuft_ab_am > ?`,
		tokenHash, fmtTime(now)))
}

func (s *Store) DeleteSitzung(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sitzung WHERE token_hash = ?`, tokenHash)
	return err
}
