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

func EmailNormalisieren(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// NutzerAnlegen legt eine neue Person samt Nutzerzugang an.
func (s *Store) NutzerAnlegen(ctx context.Context, name, email, passwortHash string, jetzt time.Time) (Nutzer, error) {
	n := Nutzer{Name: name, Email: EmailNormalisieren(email), PasswortHash: passwortHash}
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var vorhanden int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM nutzer WHERE email = ?`, n.Email).Scan(&vorhanden); err != nil {
			return err
		}
		if vorhanden > 0 {
			return ErrEmailVergeben
		}
		err := tx.QueryRowContext(ctx, `INSERT INTO person (name, erstellt_am) VALUES (?, ?) RETURNING id`,
			name, zeitText(jetzt)).Scan(&n.PersonID)
		if err != nil {
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

func nutzerLesen(row *sql.Row) (Nutzer, error) {
	var n Nutzer
	err := row.Scan(&n.PersonID, &n.Name, &n.Email, &n.PasswortHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Nutzer{}, ErrNichtGefunden
	}
	return n, err
}

func (s *Store) NutzerPerEmail(ctx context.Context, email string) (Nutzer, error) {
	return nutzerLesen(s.db.QueryRowContext(ctx, nutzerSelect+` WHERE n.email = ?`, EmailNormalisieren(email)))
}

// PasswortSetzen ersetzt das Passwort und beendet alle Sitzungen des Nutzers.
func (s *Store) PasswortSetzen(ctx context.Context, email, passwortHash string) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT person_id FROM nutzer WHERE email = ?`, EmailNormalisieren(email)).Scan(&id)
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

func (s *Store) SitzungAnlegen(ctx context.Context, tokenHash string, personID int64, jetzt, laeuftAb time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sitzung (token_hash, person_id, erstellt_am, laeuft_ab_am) VALUES (?, ?, ?, ?)`,
		tokenHash, personID, zeitText(jetzt), zeitText(laeuftAb))
	return err
}

// SitzungsNutzer liefert den Nutzer einer noch gültigen Sitzung.
func (s *Store) SitzungsNutzer(ctx context.Context, tokenHash string, jetzt time.Time) (Nutzer, error) {
	return nutzerLesen(s.db.QueryRowContext(ctx,
		nutzerSelect+` JOIN sitzung s ON s.person_id = n.person_id WHERE s.token_hash = ? AND s.laeuft_ab_am > ?`,
		tokenHash, zeitText(jetzt)))
}

func (s *Store) SitzungLoeschen(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sitzung WHERE token_hash = ?`, tokenHash)
	return err
}
