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
	Design       Design
}

// Design ist die Gestaltung der Oberfläche, die ein Nutzer gewählt hat; KeinDesign heißt, noch keine.
type Design string

const (
	KeinDesign Design = ""
	Minimal    Design = "minimal"
	Neon       Design = "neon"
	Weich      Design = "weich"
)

// Waehlbar sagt, ob ein Nutzer dieses Design wählen kann.
func (d Design) Waehlbar() bool {
	switch d {
	case Minimal, Neon, Weich:
		return true
	}
	return false
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

const nutzerSelect = `SELECT n.person_id, p.name, n.email, n.passwort_hash, n.design FROM nutzer n JOIN person p ON p.id = n.person_id`

func nutzerLesen(z zeile) (Nutzer, error) {
	var n Nutzer
	var design sql.NullString
	err := z.Scan(&n.PersonID, &n.Name, &n.Email, &n.PasswortHash, &design)
	if errors.Is(err, sql.ErrNoRows) {
		return Nutzer{}, ErrNichtGefunden
	}
	n.Design = Design(design.String)
	return n, err
}

func (s *Store) NutzerPerEmail(ctx context.Context, email string) (Nutzer, error) {
	return nutzerLesen(s.db.QueryRowContext(ctx, nutzerSelect+` WHERE n.email = ?`, EmailNormalisieren(email)))
}

// DesignSetzen speichert das gewählte Design am Nutzer und liefert ihn mit der neuen Einstellung.
func (s *Store) DesignSetzen(ctx context.Context, personID int64, d Design) (Nutzer, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE nutzer SET design = ? WHERE person_id = ?`, string(d), personID); err != nil {
		return Nutzer{}, err
	}
	return nutzerLesen(s.db.QueryRowContext(ctx, nutzerSelect+` WHERE n.person_id = ?`, personID))
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
