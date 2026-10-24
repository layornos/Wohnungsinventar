package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"inventar/internal/auth"
	"inventar/internal/store"
)

const invitationDuration = 7 * 24 * time.Hour

type invitationJSON struct {
	ID int64 `json:"id"`
	// PersonID is null if the person is only created on acceptance.
	PersonID  *int64    `json:"personId"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func toInvitationJSON(i store.Invitation) invitationJSON {
	j := invitationJSON{ID: i.ID, Name: i.Name, CreatedBy: i.CreatedBy, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt}
	if i.PersonID != 0 {
		j.PersonID = &i.PersonID
	}
	return j
}

// invitationError translates the domain errors around invitations, everything else like storeError.
func (s *server) invitationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrInvitationExpired):
		writeError(w, http.StatusGone, "invitation_expired", "Diese Einladung ist abgelaufen. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrInvitationUsed):
		writeError(w, http.StatusGone, "invitation_used", "Diese Einladung wurde schon verwendet. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrInvitationRevoked):
		writeError(w, http.StatusGone, "invitation_revoked", "Diese Einladung wurde zurückgezogen. Bitte um einen neuen Link.")
	case errors.Is(err, store.ErrAlreadyUser):
		writeError(w, http.StatusConflict, "already_user", "Diese Person ist bereits Nutzer")
	case errors.Is(err, store.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "Diese E-Mail-Adresse ist bereits vergeben")
	default:
		s.storeError(w, r, err)
	}
}

func invitationInvalid(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "invitation_invalid", "Diese Einladung gibt es nicht. Prüfe den Link oder bitte um einen neuen.")
}

// invitationLink builds the link from the address the app is currently opened under, so that
// it is the same at home and via the VPN. The token is in the fragment and so, when the link
// is opened, reaches neither the server nor its logs. The UI route stays German.
func invitationLink(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/einladung#" + token
}

func (s *server) createInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PersonID int64  `json:"personId"`
		Name     string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	n := store.NewInvitation{PersonID: req.PersonID, CreatedByID: userFrom(r.Context()).PersonID}
	if n.PersonID == 0 {
		n.Name = strings.TrimSpace(req.Name)
		if n.Name == "" {
			nameMissing(w)
			return
		}
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	n.TokenHash = hash
	now := s.Now()
	i, err := s.Store.CreateInvitation(r.Context(), n, now, now.Add(invitationDuration))
	if err != nil {
		s.invitationError(w, r, err)
		return
	}
	// The link is only returned here, once; only the hash is stored.
	writeJSON(w, http.StatusCreated, struct {
		invitationJSON
		Link string `json:"link"`
	}{toInvitationJSON(i), invitationLink(r, token)})
}

func (s *server) listInvitations(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.OpenInvitations(r.Context(), s.Now())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]invitationJSON, 0, len(list))
	for _, i := range list {
		out = append(out, toInvitationJSON(i))
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": out})
}

func (s *server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.RevokeInvitation(r.Context(), id, s.Now()); err != nil {
		s.invitationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListUsers(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]userJSON, 0, len(list))
	for _, u := range list {
		out = append(out, toUserJSON(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

// The token comes in the body instead of the path so it does not end up in logs of request paths.
type tokenRequest struct {
	Token string `json:"token"`
}

// checkInvitation is public and only reveals name and creator for an open invitation.
func (s *server) checkInvitation(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		invitationInvalid(w)
		return
	}
	i, err := s.Store.InvitationByToken(r.Context(), auth.TokenHash(req.Token), s.Now())
	if errors.Is(err, store.ErrNotFound) {
		invitationInvalid(w)
		return
	}
	if err != nil {
		s.invitationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": i.Name, "createdBy": i.CreatedBy, "expiresAt": i.ExpiresAt})
}

// acceptInvitation is public: the invited person becomes a user and is logged in afterwards.
func (s *server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		tokenRequest
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		invitationInvalid(w)
		return
	}
	email := store.NormalizeEmail(req.Email)
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "email_invalid", "Bitte eine gültige E-Mail-Adresse angeben")
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if errors.Is(err, auth.ErrPasswordTooShort) {
		writeError(w, http.StatusBadRequest, "password_too_short", auth.ErrPasswordTooShort.Error())
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	session, sessionHash, err := auth.NewToken()
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	now := s.Now()
	u, err := s.Store.AcceptInvitation(r.Context(), store.Acceptance{
		TokenHash:        auth.TokenHash(req.Token),
		Email:            email,
		PasswordHash:     passwordHash,
		SessionHash:      sessionHash,
		SessionExpiresAt: now.Add(sessionDuration),
	}, now)
	if errors.Is(err, store.ErrNotFound) {
		invitationInvalid(w)
		return
	}
	if err != nil {
		s.invitationError(w, r, err)
		return
	}
	s.setSessionCookie(w, session, int(sessionDuration.Seconds()))
	writeJSON(w, http.StatusOK, toMeJSON(u))
}
