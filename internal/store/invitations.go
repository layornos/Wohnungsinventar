package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrAlreadyUser       = errors.New("person is already a user")
	ErrInvitationExpired = errors.New("invitation expired")
	ErrInvitationUsed    = errors.New("invitation used")
	ErrInvitationRevoked = errors.New("invitation revoked")
)

type InvitationState string

const (
	InvitationOpen     InvitationState = "open"
	InvitationAccepted InvitationState = "accepted"
	InvitationRevoked  InvitationState = "revoked"
	InvitationExpired  InvitationState = "expired"
)

// Invitation invites an existing person (PersonID) or a person yet to be created (PersonID 0).
// For a linked person, Name is their current name.
type Invitation struct {
	ID        int64
	PersonID  int64
	Name      string
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt time.Time
	Accepted  bool
	Revoked   bool
	// PersonIsUser: the linked person has meanwhile got user access of their own.
	PersonIsUser bool
}

func (i Invitation) State(now time.Time) InvitationState {
	switch {
	case i.Revoked:
		return InvitationRevoked
	case i.Accepted:
		return InvitationAccepted
	case !now.Before(i.ExpiresAt):
		return InvitationExpired
	}
	return InvitationOpen
}

// stateError returns the matching domain error for an invitation that is not open.
func (i Invitation) stateError(now time.Time) error {
	switch i.State(now) {
	case InvitationRevoked:
		return ErrInvitationRevoked
	case InvitationAccepted:
		return ErrInvitationUsed
	case InvitationExpired:
		return ErrInvitationExpired
	}
	return nil
}

type NewInvitation struct {
	TokenHash   string
	PersonID    int64
	Name        string
	CreatedByID int64
}

const invitationSelect = `
	SELECT i.id, COALESCE(i.person_id, 0), COALESCE(p.name, i.name), cp.name, i.created_at, i.expires_at,
		i.accepted_at IS NOT NULL, i.revoked_at IS NOT NULL,
		EXISTS (SELECT 1 FROM app_user u WHERE u.person_id = i.person_id)
	FROM invitation i
	LEFT JOIN person p ON p.id = i.person_id
	JOIN person cp ON cp.id = i.created_by_id`

func readInvitation(r row) (Invitation, error) {
	var i Invitation
	var created, expires string
	err := r.Scan(&i.ID, &i.PersonID, &i.Name, &i.CreatedBy, &created, &expires, &i.Accepted, &i.Revoked, &i.PersonIsUser)
	if errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, ErrNotFound
	}
	if err != nil {
		return Invitation{}, err
	}
	if i.CreatedAt, err = parseTime(created); err != nil {
		return Invitation{}, err
	}
	if i.ExpiresAt, err = parseTime(expires); err != nil {
		return Invitation{}, err
	}
	return i, nil
}

// CreateInvitation creates an invitation. A linked person must not be a user yet.
func (s *Store) CreateInvitation(ctx context.Context, n NewInvitation, now, expiresAt time.Time) (Invitation, error) {
	var inv Invitation
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var personID, name any = nil, n.Name
		if n.PersonID != 0 {
			personID, name = n.PersonID, nil
			var exists, isUser bool
			err := tx.QueryRowContext(ctx, `SELECT
				EXISTS (SELECT 1 FROM person WHERE id = ?),
				EXISTS (SELECT 1 FROM app_user WHERE person_id = ?)`, n.PersonID, n.PersonID).Scan(&exists, &isUser)
			switch {
			case err != nil:
				return err
			case !exists:
				return ErrUnknownPerson
			case isUser:
				return ErrAlreadyUser
			}
		}
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO invitation (token_hash, person_id, name, created_by_id, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
			n.TokenHash, personID, name, n.CreatedByID, timeText(now), timeText(expiresAt)).Scan(&id)
		if err != nil {
			return err
		}
		inv, err = readInvitation(tx.QueryRowContext(ctx, invitationSelect+` WHERE i.id = ?`, id))
		return err
	})
	return inv, err
}

// OpenInvitations lists the invitations that can still be accepted, those expiring first at the front.
func (s *Store) OpenInvitations(ctx context.Context, now time.Time) ([]Invitation, error) {
	rows, err := s.db.QueryContext(ctx, invitationSelect+`
		WHERE i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > ?
			AND NOT EXISTS (SELECT 1 FROM app_user u WHERE u.person_id = i.person_id)
		ORDER BY i.expires_at, i.id`, timeText(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invitation{}
	for rows.Next() {
		i, err := readInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// RevokeInvitation makes an invitation not yet accepted unusable. Revoking again changes nothing.
func (s *Store) RevokeInvitation(ctx context.Context, id int64, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var accepted bool
		err := tx.QueryRowContext(ctx, `SELECT accepted_at IS NOT NULL FROM invitation WHERE id = ?`, id).Scan(&accepted)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNotFound
		case err != nil:
			return err
		case accepted:
			return ErrInvitationUsed
		}
		_, err = tx.ExecContext(ctx, `UPDATE invitation SET revoked_at = COALESCE(revoked_at, ?) WHERE id = ?`,
			timeText(now), id)
		return err
	})
}

// InvitationByToken returns an open invitation, otherwise ErrNotFound or the error of its state.
func (s *Store) InvitationByToken(ctx context.Context, tokenHash string, now time.Time) (Invitation, error) {
	i, err := readInvitation(s.db.QueryRowContext(ctx, invitationSelect+` WHERE i.token_hash = ?`, tokenHash))
	if err != nil {
		return Invitation{}, err
	}
	if err := i.stateError(now); err != nil {
		return Invitation{}, err
	}
	if i.PersonIsUser {
		return Invitation{}, ErrAlreadyUser
	}
	return i, nil
}

// Acceptance holds the details of the invited person and the session they have afterwards.
type Acceptance struct {
	TokenHash        string
	Email            string
	PasswordHash     string
	SessionHash      string
	SessionExpiresAt time.Time
}

// AcceptInvitation makes the invited person a user and logs them in, all or nothing.
func (s *Store) AcceptInvitation(ctx context.Context, a Acceptance, now time.Time) (User, error) {
	u := User{Email: NormalizeEmail(a.Email), PasswordHash: a.PasswordHash}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		// Write first: that way the transaction holds the write lock before it checks anything,
		// and two concurrent acceptances of the same invitation cannot both succeed.
		var id int64
		var personID sql.NullInt64
		var name sql.NullString
		err := tx.QueryRowContext(ctx, `
			UPDATE invitation SET accepted_at = ?
			WHERE token_hash = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?
			RETURNING id, person_id, name`,
			timeText(now), a.TokenHash, timeText(now)).Scan(&id, &personID, &name)
		if errors.Is(err, sql.ErrNoRows) {
			i, err := readInvitation(tx.QueryRowContext(ctx, invitationSelect+` WHERE i.token_hash = ?`, a.TokenHash))
			if err != nil {
				return err
			}
			return i.stateError(now)
		}
		if err != nil {
			return err
		}

		var emailTaken bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM app_user WHERE email = ?)`, u.Email).Scan(&emailTaken); err != nil {
			return err
		}
		if personID.Valid {
			u.PersonID = personID.Int64
			var alreadyUser bool
			err := tx.QueryRowContext(ctx, `SELECT p.name, EXISTS (SELECT 1 FROM app_user WHERE person_id = p.id)
				FROM person p WHERE p.id = ?`, u.PersonID).Scan(&u.Name, &alreadyUser)
			if err != nil {
				return err
			}
			if alreadyUser {
				return ErrAlreadyUser
			}
		}
		if emailTaken {
			return ErrEmailTaken
		}
		if !personID.Valid {
			u.Name = name.String
			err := tx.QueryRowContext(ctx, `INSERT INTO person (name, created_at) VALUES (?, ?) RETURNING id`,
				u.Name, timeText(now)).Scan(&u.PersonID)
			if err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO app_user (person_id, email, password_hash) VALUES (?, ?, ?)`,
			u.PersonID, u.Email, u.PasswordHash); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO session (token_hash, person_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
			a.SessionHash, u.PersonID, timeText(now), timeText(a.SessionExpiresAt))
		return err
	})
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// ListUsers returns all users of the household by name.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, userSelect+` ORDER BY p.name, u.person_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := readUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
