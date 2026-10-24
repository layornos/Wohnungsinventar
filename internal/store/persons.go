package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrHasUserAccess = errors.New("person has user access")
	ErrLastOwner     = errors.New("last owner of an item")
	ErrUnknownPerson = errors.New("unknown person")
)

// Person is someone in the household; IsUser says whether they have user access.
type Person struct {
	ID     int64
	Name   string
	IsUser bool
}

const personSelect = `SELECT p.id, p.name, u.person_id IS NOT NULL FROM person p LEFT JOIN app_user u ON u.person_id = p.id`

func (s *Store) ListPersons(ctx context.Context) ([]Person, error) {
	return readPersons(s.db.QueryContext(ctx, personSelect+` ORDER BY p.name, p.id`))
}

func readPersons(rows *sql.Rows, err error) ([]Person, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	persons := []Person{}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Name, &p.IsUser); err != nil {
			return nil, err
		}
		persons = append(persons, p)
	}
	return persons, rows.Err()
}

func (s *Store) CreatePerson(ctx context.Context, name string, now time.Time) (Person, error) {
	p := Person{Name: name}
	err := s.db.QueryRowContext(ctx, `INSERT INTO person (name, created_at) VALUES (?, ?) RETURNING id`,
		name, timeText(now)).Scan(&p.ID)
	return p, err
}

func (s *Store) RenamePerson(ctx context.Context, id int64, name string) (Person, error) {
	p := Person{ID: id, Name: name}
	err := s.db.QueryRowContext(ctx, `
		UPDATE person SET name = ? WHERE id = ?
		RETURNING EXISTS (SELECT 1 FROM app_user WHERE person_id = person.id)`, name, id).Scan(&p.IsUser)
	if errors.Is(err, sql.ErrNoRows) {
		return Person{}, ErrNotFound
	}
	return p, err
}

// DeletePerson deletes a person without user access who is not the last owner of any item.
// Their other ownerships go away.
func (s *Store) DeletePerson(ctx context.Context, id int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var exists, isUser, lastOwner bool
		err := tx.QueryRowContext(ctx, `
			SELECT
				EXISTS (SELECT 1 FROM person WHERE id = ?),
				EXISTS (SELECT 1 FROM app_user WHERE person_id = ?),
				EXISTS (SELECT 1 FROM ownership o WHERE o.person_id = ?
					AND NOT EXISTS (SELECT 1 FROM ownership a WHERE a.item_id = o.item_id AND a.person_id <> o.person_id))`,
			id, id, id).Scan(&exists, &isUser, &lastOwner)
		switch {
		case err != nil:
			return err
		case !exists:
			return ErrNotFound
		case isUser:
			return ErrHasUserAccess
		case lastOwner:
			return ErrLastOwner
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM person WHERE id = ?`, id)
		return err
	})
}
