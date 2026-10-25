package server

import (
	"net/http"
	"strconv"
	"time"

	"inventar/internal/store"
)

// The kinds and reactions in the API, camelCase like the other JSON values.
var (
	kindJSON     = map[store.ReminderKind]string{store.KeepReview: "keepReview", store.SellLaterDue: "sellLater"}
	reactionJSON = map[store.Reaction]string{
		store.ReactKeep: "keep", store.ReactSell: "sell", store.ReactSellNow: "sellNow",
		store.ReactOneMoreYear: "oneMoreYear", store.ReactKeepAfterAll: "keepAfterAll",
		"disposition_changed": "dispositionChanged", "handed_over": "handedOver",
	}
)

func reactionFromJSON(v string) (store.Reaction, bool) {
	for r, name := range reactionJSON {
		if name == v {
			return r, true
		}
	}
	return "", false
}

type scheduleJSON struct {
	Kind  string `json:"kind"`
	DueOn string `json:"dueOn"`
}

type reminderItemJSON struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	ThumbnailURL string       `json:"thumbnailUrl"`
	Owners       []personJSON `json:"owners"`
}

type reminderJSON struct {
	ID          int64            `json:"id"`
	Item        reminderItemJSON `json:"item"`
	Kind        string           `json:"kind"`
	DueOn       string           `json:"dueOn"`
	TriggeredAt time.Time        `json:"triggeredAt"`
	// DoneAt, DoneBy and Reaction are null while the reminder is open; DoneBy also when it was settled automatically.
	DoneAt   *time.Time `json:"doneAt"`
	DoneBy   *namedJSON `json:"doneBy"`
	Reaction *string    `json:"reaction"`
}

func toReminderJSON(r store.Reminder) reminderJSON {
	out := reminderJSON{
		ID: r.ID, Kind: kindJSON[r.Kind], DueOn: r.DueOn, TriggeredAt: r.TriggeredAt, DoneAt: r.DoneAt,
		Item: reminderItemJSON{ID: r.Item.ID, Name: r.Item.Name, ThumbnailURL: thumbnailURL(r.Item.MainPhoto), Owners: toPersonsJSON(r.Item.Owners)},
	}
	if r.DoneBy != nil {
		n := toNamedJSON(*r.DoneBy)
		out.DoneBy = &n
	}
	if r.Reaction != "" {
		name := reactionJSON[r.Reaction]
		out.Reaction = &name
	}
	return out
}

func toRemindersJSON(list []store.Reminder) []reminderJSON {
	out := make([]reminderJSON, 0, len(list))
	for _, r := range list {
		out = append(out, toReminderJSON(r))
	}
	return out
}

func toSchedulesJSON(list []store.Schedule) []scheduleJSON {
	out := make([]scheduleJSON, 0, len(list))
	for _, s := range list {
		out = append(out, scheduleJSON{Kind: kindJSON[s.Kind], DueOn: s.DueOn})
	}
	return out
}

// listReminders lists open (state=open, the default) or done (state=done) reminders, with mine=1 only those the
// logged-in user is a recipient of, in pages with page and perPage.
func (s *server) listReminders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.ReminderFilter{Limit: defaultPerPage}
	switch q.Get("state") {
	case "", "open":
		f.Open = true
	case "done":
		f.Done = true
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "state muss open oder done sein")
		return
	}
	if q.Get("mine") != "" {
		f.RecipientID = userFrom(r.Context()).PersonID
	}
	page := 1
	if p := q.Get("page"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige Seite")
			return
		}
		page = n
	}
	if p := q.Get("perPage"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > maxPerPage {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige Seitengröße")
			return
		}
		f.Limit = n
	}
	f.Offset = (page - 1) * f.Limit
	list, more, err := s.Store.ListReminders(r.Context(), f)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reminders": toRemindersJSON(list), "more": more})
}

func (s *server) countReminders(w http.ResponseWriter, r *http.Request) {
	n, err := s.Store.CountOpenReminders(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"open": n})
}

func (s *server) getReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	rem, err := s.Store.GetReminder(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toReminderJSON(rem))
}

// react answers a reminder: {"reaction": "keep"|"sell" (keep review) or "sellNow"|"oneMoreYear"|"keepAfterAll"
// (sell later)}.
func (s *server) react(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Reaction string `json:"reaction"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reaction, ok := reactionFromJSON(req.Reaction)
	if !ok {
		writeError(w, http.StatusBadRequest, "reaction_mismatch", msgReactionMismatch)
		return
	}
	rem, err := s.Store.React(r.Context(), id, reaction, userFrom(r.Context()).PersonID, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toReminderJSON(rem))
}

const msgReactionMismatch = "Diese Antwort passt nicht zur Erinnerung"

func (s *server) setReminderMail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	u, err := s.Store.SetReminderMail(r.Context(), userFrom(r.Context()).PersonID, req.Enabled)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMeJSON(u))
}
