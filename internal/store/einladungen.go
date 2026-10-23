package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrBereitsNutzer           = errors.New("Person ist bereits Nutzer")
	ErrEinladungAbgelaufen     = errors.New("Einladung abgelaufen")
	ErrEinladungVerwendet      = errors.New("Einladung verwendet")
	ErrEinladungZurueckgezogen = errors.New("Einladung zurückgezogen")
)

type EinladungsZustand string

const (
	EinladungOffen          EinladungsZustand = "offen"
	EinladungAngenommen     EinladungsZustand = "angenommen"
	EinladungZurueckgezogen EinladungsZustand = "zurueckgezogen"
	EinladungAbgelaufen     EinladungsZustand = "abgelaufen"
)

// Einladung lädt eine bestehende Person (PersonID) oder eine noch anzulegende Person (PersonID 0) ein.
// Name ist bei einer verknüpften Person ihr aktueller Name.
type Einladung struct {
	ID             int64
	PersonID       int64
	Name           string
	Ersteller      string
	ErstelltAm     time.Time
	LaeuftAbAm     time.Time
	Angenommen     bool
	Zurueckgezogen bool
}

func (e Einladung) Zustand(jetzt time.Time) EinladungsZustand {
	switch {
	case e.Zurueckgezogen:
		return EinladungZurueckgezogen
	case e.Angenommen:
		return EinladungAngenommen
	case !jetzt.Before(e.LaeuftAbAm):
		return EinladungAbgelaufen
	}
	return EinladungOffen
}

// zustandFehler liefert für eine nicht offene Einladung den passenden fachlichen Fehler.
func (e Einladung) zustandFehler(jetzt time.Time) error {
	switch e.Zustand(jetzt) {
	case EinladungZurueckgezogen:
		return ErrEinladungZurueckgezogen
	case EinladungAngenommen:
		return ErrEinladungVerwendet
	case EinladungAbgelaufen:
		return ErrEinladungAbgelaufen
	}
	return nil
}

type NeueEinladung struct {
	TokenHash   string
	PersonID    int64
	Name        string
	ErstellerID int64
}

const einladungSelect = `
	SELECT e.id, COALESCE(e.person_id, 0), COALESCE(p.name, e.name), ep.name, e.erstellt_am, e.laeuft_ab_am,
		e.angenommen_am IS NOT NULL, e.zurueckgezogen_am IS NOT NULL
	FROM einladung e
	LEFT JOIN person p ON p.id = e.person_id
	JOIN person ep ON ep.id = e.ersteller_id`

func einladungLesen(z zeile) (Einladung, error) {
	var e Einladung
	var erstellt, laeuftAb string
	err := z.Scan(&e.ID, &e.PersonID, &e.Name, &e.Ersteller, &erstellt, &laeuftAb, &e.Angenommen, &e.Zurueckgezogen)
	if errors.Is(err, sql.ErrNoRows) {
		return Einladung{}, ErrNichtGefunden
	}
	if err != nil {
		return Einladung{}, err
	}
	if e.ErstelltAm, err = zeitLesen(erstellt); err != nil {
		return Einladung{}, err
	}
	if e.LaeuftAbAm, err = zeitLesen(laeuftAb); err != nil {
		return Einladung{}, err
	}
	return e, nil
}

// EinladungAnlegen legt eine Einladung an. Eine verknüpfte Person darf noch kein Nutzer sein.
func (s *Store) EinladungAnlegen(ctx context.Context, neu NeueEinladung, jetzt, laeuftAb time.Time) (Einladung, error) {
	var e Einladung
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var personID, name any = nil, neu.Name
		if neu.PersonID != 0 {
			personID, name = neu.PersonID, nil
			var existiert, nutzer bool
			err := tx.QueryRowContext(ctx, `SELECT
				EXISTS (SELECT 1 FROM person WHERE id = ?),
				EXISTS (SELECT 1 FROM nutzer WHERE person_id = ?)`, neu.PersonID, neu.PersonID).Scan(&existiert, &nutzer)
			switch {
			case err != nil:
				return err
			case !existiert:
				return ErrUnbekanntePerson
			case nutzer:
				return ErrBereitsNutzer
			}
		}
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO einladung (token_hash, person_id, name, ersteller_id, erstellt_am, laeuft_ab_am)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
			neu.TokenHash, personID, name, neu.ErstellerID, zeitText(jetzt), zeitText(laeuftAb)).Scan(&id)
		if err != nil {
			return err
		}
		e, err = einladungLesen(tx.QueryRowContext(ctx, einladungSelect+` WHERE e.id = ?`, id))
		return err
	})
	return e, err
}

// EinladungenOffen listet die noch annehmbaren Einladungen, die zuerst ablaufenden vorn.
func (s *Store) EinladungenOffen(ctx context.Context, jetzt time.Time) ([]Einladung, error) {
	rows, err := s.db.QueryContext(ctx, einladungSelect+`
		WHERE e.angenommen_am IS NULL AND e.zurueckgezogen_am IS NULL AND e.laeuft_ab_am > ?
			AND NOT EXISTS (SELECT 1 FROM nutzer n WHERE n.person_id = e.person_id)
		ORDER BY e.laeuft_ab_am, e.id`, zeitText(jetzt))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Einladung{}
	for rows.Next() {
		e, err := einladungLesen(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EinladungZurueckziehen macht eine noch nicht angenommene Einladung unbrauchbar.
// Erneutes Zurückziehen ändert nichts.
func (s *Store) EinladungZurueckziehen(ctx context.Context, id int64, jetzt time.Time) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var angenommen bool
		err := tx.QueryRowContext(ctx, `SELECT angenommen_am IS NOT NULL FROM einladung WHERE id = ?`, id).Scan(&angenommen)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNichtGefunden
		case err != nil:
			return err
		case angenommen:
			return ErrEinladungVerwendet
		}
		_, err = tx.ExecContext(ctx, `UPDATE einladung SET zurueckgezogen_am = COALESCE(zurueckgezogen_am, ?) WHERE id = ?`,
			zeitText(jetzt), id)
		return err
	})
}

// EinladungPerToken liefert eine offene Einladung, sonst ErrNichtGefunden oder den Fehler ihres Zustands.
func (s *Store) EinladungPerToken(ctx context.Context, tokenHash string, jetzt time.Time) (Einladung, error) {
	e, err := einladungLesen(s.db.QueryRowContext(ctx, einladungSelect+` WHERE e.token_hash = ?`, tokenHash))
	if err != nil {
		return Einladung{}, err
	}
	if err := e.zustandFehler(jetzt); err != nil {
		return Einladung{}, err
	}
	return e, nil
}

// Annahme sind die Angaben der eingeladenen Person und die Sitzung, die sie danach hat.
type Annahme struct {
	TokenHash       string
	Email           string
	PasswortHash    string
	SitzungsHash    string
	SitzungLaeuftAb time.Time
}

// EinladungAnnehmen macht die eingeladene Person zum Nutzer und meldet sie an, alles oder nichts.
func (s *Store) EinladungAnnehmen(ctx context.Context, a Annahme, jetzt time.Time) (Nutzer, error) {
	n := Nutzer{Email: EmailNormalisieren(a.Email), PasswortHash: a.PasswortHash}
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		// Zuerst schreiben: so hält die Transaktion die Schreibsperre, bevor sie etwas prüft,
		// und zwei gleichzeitige Annahmen derselben Einladung können nicht beide durchgehen.
		var id int64
		var personID sql.NullInt64
		var name sql.NullString
		err := tx.QueryRowContext(ctx, `
			UPDATE einladung SET angenommen_am = ?
			WHERE token_hash = ? AND angenommen_am IS NULL AND zurueckgezogen_am IS NULL AND laeuft_ab_am > ?
			RETURNING id, person_id, name`,
			zeitText(jetzt), a.TokenHash, zeitText(jetzt)).Scan(&id, &personID, &name)
		if errors.Is(err, sql.ErrNoRows) {
			e, err := einladungLesen(tx.QueryRowContext(ctx, einladungSelect+` WHERE e.token_hash = ?`, a.TokenHash))
			if err != nil {
				return err
			}
			return e.zustandFehler(jetzt)
		}
		if err != nil {
			return err
		}

		var emailVergeben bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM nutzer WHERE email = ?)`, n.Email).Scan(&emailVergeben); err != nil {
			return err
		}
		if personID.Valid {
			n.PersonID = personID.Int64
			var bereitsNutzer bool
			err := tx.QueryRowContext(ctx, `SELECT p.name, EXISTS (SELECT 1 FROM nutzer WHERE person_id = p.id)
				FROM person p WHERE p.id = ?`, n.PersonID).Scan(&n.Name, &bereitsNutzer)
			if err != nil {
				return err
			}
			if bereitsNutzer {
				return ErrBereitsNutzer
			}
		}
		if emailVergeben {
			return ErrEmailVergeben
		}
		if !personID.Valid {
			n.Name = name.String
			err := tx.QueryRowContext(ctx, `INSERT INTO person (name, erstellt_am) VALUES (?, ?) RETURNING id`,
				n.Name, zeitText(jetzt)).Scan(&n.PersonID)
			if err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO nutzer (person_id, email, passwort_hash) VALUES (?, ?, ?)`,
			n.PersonID, n.Email, n.PasswortHash); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO sitzung (token_hash, person_id, erstellt_am, laeuft_ab_am) VALUES (?, ?, ?, ?)`,
			a.SitzungsHash, n.PersonID, zeitText(jetzt), zeitText(a.SitzungLaeuftAb))
		return err
	})
	if err != nil {
		return Nutzer{}, err
	}
	return n, nil
}

// NutzerAuflisten liefert alle Nutzer des Haushalts nach Namen.
func (s *Store) NutzerAuflisten(ctx context.Context) ([]Nutzer, error) {
	rows, err := s.db.QueryContext(ctx, nutzerSelect+` ORDER BY p.name, n.person_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Nutzer{}
	for rows.Next() {
		var n Nutzer
		if err := rows.Scan(&n.PersonID, &n.Name, &n.Email, &n.PasswortHash); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
