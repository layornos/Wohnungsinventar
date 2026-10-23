package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrHatNutzerzugang  = errors.New("Person hat Nutzerzugang")
	ErrLetzterBesitzer  = errors.New("letzter Besitzer eines Items")
	ErrUnbekanntePerson = errors.New("unbekannte Person")
)

// Person ist jemand im Haushalt; Nutzer gibt an, ob sie einen Nutzerzugang hat.
type Person struct {
	ID     int64
	Name   string
	Nutzer bool
}

func (s *Store) PersonenAuflisten(ctx context.Context) ([]Person, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.name, n.person_id IS NOT NULL
		FROM person p LEFT JOIN nutzer n ON n.person_id = p.id
		ORDER BY p.name, p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	personen := []Person{}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Nutzer); err != nil {
			return nil, err
		}
		personen = append(personen, p)
	}
	return personen, rows.Err()
}

func (s *Store) PersonAnlegen(ctx context.Context, name string, jetzt time.Time) (Person, error) {
	p := Person{Name: name}
	err := s.db.QueryRowContext(ctx, `INSERT INTO person (name, erstellt_am) VALUES (?, ?) RETURNING id`,
		name, zeitText(jetzt)).Scan(&p.ID)
	return p, err
}

func (s *Store) PersonUmbenennen(ctx context.Context, id int64, name string) (Person, error) {
	p := Person{ID: id, Name: name}
	err := s.db.QueryRowContext(ctx, `
		UPDATE person SET name = ? WHERE id = ?
		RETURNING EXISTS (SELECT 1 FROM nutzer WHERE person_id = person.id)`, name, id).Scan(&p.Nutzer)
	if errors.Is(err, sql.ErrNoRows) {
		return Person{}, ErrNichtGefunden
	}
	return p, err
}

// PersonLoeschen löscht eine Person ohne Nutzerzugang, die bei keinem Item letzter Besitzer ist.
// Ihre übrigen Besitz-Zuordnungen entfallen.
func (s *Store) PersonLoeschen(ctx context.Context, id int64) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var existiert, nutzer, letzterBesitzer bool
		err := tx.QueryRowContext(ctx, `
			SELECT
				EXISTS (SELECT 1 FROM person WHERE id = ?),
				EXISTS (SELECT 1 FROM nutzer WHERE person_id = ?),
				EXISTS (SELECT 1 FROM besitz b WHERE b.person_id = ?
					AND NOT EXISTS (SELECT 1 FROM besitz a WHERE a.item_id = b.item_id AND a.person_id <> b.person_id))`,
			id, id, id).Scan(&existiert, &nutzer, &letzterBesitzer)
		switch {
		case err != nil:
			return err
		case !existiert:
			return ErrNichtGefunden
		case nutzer:
			return ErrHatNutzerzugang
		case letzterBesitzer:
			return ErrLetzterBesitzer
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM person WHERE id = ?`, id)
		return err
	})
}
