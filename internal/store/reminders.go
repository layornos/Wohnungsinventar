package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	_ "time/tzdata" // the household's time zone must not depend on the installation of the Pi
)

var (
	ErrReminderDone     = errors.New("reminder is already done")
	ErrReactionMismatch = errors.New("reaction does not fit the reminder")
	ErrPriceMissing     = errors.New("new asking price missing")
)

// Household is the time zone of the household: due dates, "sold on" and the periods of the proceeds summary are
// days there.
var Household = mustLoadLocation("Europe/Berlin")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

const dayLayout = "2006-01-02"

// Day is the date of t in the household's time zone, YYYY-MM-DD.
func Day(t time.Time) string { return t.In(Household).Format(dayLayout) }

// addYears moves a day by whole years; February 29 becomes February 28 in years without it.
func addYears(day string, years int) string {
	d, err := time.Parse(dayLayout, day)
	if err != nil {
		return day
	}
	moved := d.AddDate(years, 0, 0)
	if moved.Day() != d.Day() {
		moved = moved.AddDate(0, 0, -moved.Day())
	}
	return moved.Format(dayLayout)
}

// ReminderKind is what a reminder is about.
type ReminderKind string

const (
	// KeepReview asks every three years whether a kept item is still needed.
	KeepReview ReminderKind = "keep_review"
	// SellLaterDue says, one year after "sell later", that the item is to be sold now.
	SellLaterDue ReminderKind = "sell_later"
	// PriceReminder asks to check the asking price of an item for sale after 30 days at that price (spec 08).
	PriceReminder ReminderKind = "price_reminder"
	// GiveAwayReminder says that an item has been up for giving away for 30 days (spec 08).
	GiveAwayReminder ReminderKind = "give_away_reminder"
)

// Reaction is the answer to a reminder.
type Reaction string

const (
	ReactKeep         Reaction = "keep"
	ReactSell         Reaction = "sell"
	ReactSellNow      Reaction = "sell_now"
	ReactOneMoreYear  Reaction = "one_more_year"
	ReactKeepAfterAll Reaction = "keep_after_all"
	ReactChangePrice  Reaction = "change_price"
	ReactKeepPrice    Reaction = "keep_price"
	ReactKeepOffering Reaction = "keep_offering"
	// Set by the server, not by a user.
	reactDispositionChanged Reaction = "disposition_changed"
	reactHandedOver         Reaction = "handed_over"
	reactPriceChanged       Reaction = "price_changed"
)

// Fits says whether a user can give this reaction to a reminder of the kind.
func (r Reaction) Fits(k ReminderKind) bool {
	switch k {
	case KeepReview:
		return r == ReactKeep || r == ReactSell
	case SellLaterDue:
		return r == ReactSellNow || r == ReactOneMoreYear || r == ReactKeepAfterAll
	case PriceReminder:
		return r == ReactChangePrice || r == ReactKeepPrice
	case GiveAwayReminder:
		return r == ReactKeepOffering
	}
	return false
}

// Schedule is the next planned due date of a reminder for an item.
type Schedule struct {
	Kind  ReminderKind
	DueOn string
}

type Reminder struct {
	ID int64
	// Item is filled in lists; in the reminders of an item only its ID.
	Item        ReminderItem
	Kind        ReminderKind
	DueOn       string
	TriggeredAt time.Time
	// DoneAt is nil while the reminder is open.
	DoneAt *time.Time
	// DoneBy is nil while open or when settled automatically.
	DoneBy   *Named
	Reaction Reaction
}

type ReminderItem struct {
	ID          int64
	Name        string
	MainPhoto   PhotoRef
	Owners      []Person
	AskingPrice *int64
}

// after is the due day that follows day for a kind, by its fixed period (CONTEXT.md): three years for the keep
// review, one year for sell later, 30 days for the price and the give-away reminder.
func after(k ReminderKind, day string) string {
	switch k {
	case KeepReview:
		return addYears(day, 3)
	case SellLaterDue:
		return addYears(day, 1)
	}
	d, err := time.Parse(dayLayout, day)
	if err != nil {
		return day
	}
	return d.AddDate(0, 0, 30).Format(dayLayout)
}

// kindFor is the reminder an item with this disposition gets, if any; for sale only with an asking price.
func kindFor(d Disposition, hasPrice bool) (ReminderKind, bool) {
	switch {
	case d == Keep:
		return KeepReview, true
	case d == SellLater:
		return SellLaterDue, true
	case d == Sell && hasPrice:
		return PriceReminder, true
	case d == GiveAway:
		return GiveAwayReminder, true
	}
	return "", false
}

// planFor plans the reminder of the item's disposition from today on; other schedules of the item go away.
func planFor(ctx context.Context, q querier, itemID int64, d Disposition, now time.Time) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM reminder_schedule WHERE item_id = ?`, itemID); err != nil {
		return err
	}
	var price sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT asking_price_cents FROM item WHERE id = ?`, itemID).Scan(&price); err != nil {
		return err
	}
	kind, ok := kindFor(d, price.Valid)
	if !ok {
		return nil
	}
	_, err := q.ExecContext(ctx, `INSERT INTO reminder_schedule (item_id, kind, due_on) VALUES (?, ?, ?)`,
		itemID, string(kind), after(kind, Day(now)))
	return err
}

// askingPriceChanged starts the period of the price reminder anew: open price reminders are settled, and an item
// for sale with a price gets its next one in 30 days, one without a price none (spec 08).
func askingPriceChanged(ctx context.Context, q querier, itemID int64, now time.Time) error {
	if _, err := q.ExecContext(ctx, `UPDATE item SET price_changed_at = ? WHERE id = ?`, timeText(now), itemID); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `UPDATE reminder SET done_at = ?, reaction = ? WHERE item_id = ? AND kind = ? AND done_at IS NULL`,
		timeText(now), string(reactPriceChanged), itemID, string(PriceReminder)); err != nil {
		return err
	}
	var d sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT disposition FROM item WHERE id = ?`, itemID).Scan(&d); err != nil {
		return err
	}
	return planFor(ctx, q, itemID, dispositionFrom(d), now)
}

// settleOpen settles the open reminders of an item without a user, e.g. because its disposition changed.
func settleOpen(ctx context.Context, q querier, itemID int64, r Reaction, now time.Time) error {
	_, err := q.ExecContext(ctx, `UPDATE reminder SET done_at = ?, reaction = ? WHERE item_id = ? AND done_at IS NULL`,
		timeText(now), string(r), itemID)
	return err
}

// dispositionChanged is what a new disposition means for the reminders: the period starts anew.
func dispositionChanged(ctx context.Context, q querier, itemID int64, d Disposition, now time.Time) error {
	if _, err := q.ExecContext(ctx, `UPDATE item SET disposition_since = ? WHERE id = ?`, timeText(now), itemID); err != nil {
		return err
	}
	if err := settleOpen(ctx, q, itemID, reactDispositionChanged, now); err != nil {
		return err
	}
	return planFor(ctx, q, itemID, d, now)
}

// TriggerDue turns the schedules that are due into reminders: from 8:00 in the household's time zone on their day,
// or later after an outage, each exactly once. Keep review, price and give-away reminders are planned again one
// period after their due day, so the rhythm stays the same whatever happens; an item never has two open reminders
// of a kind. Sell later reminds only once.
func (s *Store) TriggerDue(ctx context.Context, now time.Time) (int, error) {
	cutoff := Day(now)
	if now.In(Household).Hour() < 8 {
		cutoff = Day(now.AddDate(0, 0, -1))
	}
	count := 0
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT item_id, kind, due_on FROM reminder_schedule WHERE due_on <= ? ORDER BY due_on, item_id`, cutoff)
		if err != nil {
			return err
		}
		type due struct {
			itemID int64
			kind   ReminderKind
			dueOn  string
		}
		var all []due
		for rows.Next() {
			var d due
			if err := rows.Scan(&d.itemID, &d.kind, &d.dueOn); err != nil {
				rows.Close()
				return err
			}
			all = append(all, d)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, d := range all {
			var open bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM reminder WHERE item_id = ? AND kind = ? AND done_at IS NULL)`,
				d.itemID, string(d.kind)).Scan(&open); err != nil {
				return err
			}
			if !open {
				if _, err := tx.ExecContext(ctx, `INSERT INTO reminder (item_id, kind, due_on, triggered_at) VALUES (?, ?, ?, ?)`,
					d.itemID, string(d.kind), d.dueOn, timeText(now)); err != nil {
					return err
				}
				count++
			}
			if d.kind == SellLaterDue {
				if _, err := tx.ExecContext(ctx, `DELETE FROM reminder_schedule WHERE item_id = ? AND kind = ?`, d.itemID, string(d.kind)); err != nil {
					return err
				}
				continue
			}
			next := after(d.kind, d.dueOn)
			for next <= cutoff {
				next = after(d.kind, next)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE reminder_schedule SET due_on = ? WHERE item_id = ? AND kind = ?`,
				next, d.itemID, string(d.kind)); err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}

// ReminderFilter selects reminders; Open and Done can't both be set.
type ReminderFilter struct {
	Open bool
	Done bool
	// RecipientID limits them to those this user is a recipient of (see recipientCondition).
	RecipientID int64
	ItemID      int64
	Limit       int
	Offset      int
}

// recipientCondition says whether the person (a user) in the placeholder is a recipient of reminder r: as an owner
// of the item, or as any user when none of the owners is a user.
const recipientCondition = `(
	EXISTS (SELECT 1 FROM ownership o WHERE o.item_id = r.item_id AND o.person_id = ?)
	OR NOT EXISTS (SELECT 1 FROM ownership o JOIN app_user u ON u.person_id = o.person_id WHERE o.item_id = r.item_id))`

const reminderSelect = `
	SELECT r.id, r.kind, r.due_on, r.triggered_at, r.done_at, r.reaction, d.id, d.name, i.id, i.name, p.id, p.file,
		i.asking_price_cents
	FROM reminder r JOIN item i ON i.id = r.item_id
	JOIN photo p ON p.item_id = i.id AND p.main_photo = 1
	LEFT JOIN person d ON d.id = r.done_by`

// ListReminders returns the matching reminders, open ones by due date, done ones most recently settled first,
// and whether there are more.
func (s *Store) ListReminders(ctx context.Context, f ReminderFilter) ([]Reminder, bool, error) {
	var (
		conditions []string
		args       []any
	)
	if f.Open {
		conditions = append(conditions, `r.done_at IS NULL`)
	}
	if f.Done {
		conditions = append(conditions, `r.done_at IS NOT NULL`)
	}
	if f.RecipientID != 0 {
		conditions = append(conditions, recipientCondition)
		args = append(args, f.RecipientID)
	}
	if f.ItemID != 0 {
		conditions = append(conditions, `r.item_id = ?`)
		args = append(args, f.ItemID)
	}
	query := reminderSelect
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}
	if f.Done {
		query += ` ORDER BY r.done_at DESC, r.id DESC`
	} else {
		query += ` ORDER BY r.done_at IS NOT NULL, r.due_on DESC, r.id DESC`
	}
	query += ` LIMIT ? OFFSET ?`
	args = append(args, f.Limit+1, f.Offset)
	list, err := readReminders(ctx, s.db, query, args...)
	if err != nil {
		return nil, false, err
	}
	if len(list) > f.Limit {
		return list[:f.Limit], true, nil
	}
	return list, false, nil
}

func readReminders(ctx context.Context, q querier, query string, args ...any) ([]Reminder, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	list := []Reminder{}
	for rows.Next() {
		var (
			r                Reminder
			triggered        string
			doneAt, reaction sql.NullString
			doneByID         sql.NullInt64
			doneByName       sql.NullString
			price            sql.NullInt64
		)
		if err := rows.Scan(&r.ID, &r.Kind, &r.DueOn, &triggered, &doneAt, &reaction, &doneByID, &doneByName,
			&r.Item.ID, &r.Item.Name, &r.Item.MainPhoto.ID, &r.Item.MainPhoto.File, &price); err != nil {
			rows.Close()
			return nil, err
		}
		if r.TriggeredAt, err = parseTime(triggered); err != nil {
			rows.Close()
			return nil, err
		}
		if doneAt.Valid {
			t, err := parseTime(doneAt.String)
			if err != nil {
				rows.Close()
				return nil, err
			}
			r.DoneAt = &t
		}
		r.DoneBy = namedFrom(doneByID, doneByName)
		if price.Valid {
			r.Item.AskingPrice = &price.Int64
		}
		r.Reaction = Reaction(reaction.String)
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Item.Owners, err = readOwners(ctx, q, list[i].Item.ID); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// CountOpenReminders counts the open reminders of the household.
func (s *Store) CountOpenReminders(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reminder WHERE done_at IS NULL`).Scan(&n)
	return n, err
}

func (s *Store) GetReminder(ctx context.Context, id int64) (Reminder, error) {
	return getReminder(ctx, s.db, id)
}

func getReminder(ctx context.Context, q querier, id int64) (Reminder, error) {
	list, err := readReminders(ctx, q, reminderSelect+` WHERE r.id = ?`, id)
	if err != nil {
		return Reminder{}, err
	}
	if len(list) == 0 {
		return Reminder{}, ErrNotFound
	}
	return list[0], nil
}

// React settles a reminder with the reaction of a user and does what it means: a new disposition (which plans the
// next reminder), for "one more year" a new sell later reminder in a year, for "change price" the new asking price
// (which starts the price reminder anew). Keeping the price or offering on changes nothing: the next reminder is
// already planned. price is only used for "change price".
func (s *Store) React(ctx context.Context, id int64, r Reaction, price *int64, personID int64, now time.Time) (Reminder, error) {
	var out Reminder
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var (
			itemID int64
			kind   ReminderKind
			doneAt sql.NullString
		)
		err := tx.QueryRowContext(ctx, `SELECT item_id, kind, done_at FROM reminder WHERE id = ?`, id).Scan(&itemID, &kind, &doneAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if doneAt.Valid {
			return ErrReminderDone
		}
		if !r.Fits(kind) {
			return ErrReactionMismatch
		}
		if r == ReactChangePrice && price == nil {
			return ErrPriceMissing
		}
		if _, err := tx.ExecContext(ctx, `UPDATE reminder SET done_at = ?, done_by = ?, reaction = ? WHERE id = ?`,
			timeText(now), personID, string(r), id); err != nil {
			return err
		}
		var newDisposition *Disposition
		switch r {
		case ReactSell, ReactSellNow:
			d := Sell
			newDisposition = &d
		case ReactKeepAfterAll:
			d := Keep
			newDisposition = &d
		case ReactChangePrice:
			if err := touchItem(ctx, tx, itemID, now, ErrNotFound); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE item SET asking_price_cents = ? WHERE id = ?`, *price, itemID); err != nil {
				return err
			}
			if err := askingPriceChanged(ctx, tx, itemID, now); err != nil {
				return err
			}
		case ReactOneMoreYear:
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reminder_schedule (item_id, kind, due_on) VALUES (?, ?, ?)
				ON CONFLICT (item_id, kind) DO UPDATE SET due_on = excluded.due_on`,
				itemID, string(SellLaterDue), addYears(Day(now), 1)); err != nil {
				return err
			}
		}
		if newDisposition != nil {
			if err := touchItem(ctx, tx, itemID, now, ErrNotFound); err != nil {
				return err
			}
			if err := setDisposition(ctx, tx, itemID, *newDisposition, now); err != nil {
				return err
			}
		}
		out, err = getReminder(ctx, tx, id)
		return err
	})
	return out, err
}

func readSchedules(ctx context.Context, q querier, itemID int64) ([]Schedule, error) {
	rows, err := q.QueryContext(ctx, `SELECT kind, due_on FROM reminder_schedule WHERE item_id = ? ORDER BY due_on`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Schedule{}
	for rows.Next() {
		var sc Schedule
		if err := rows.Scan(&sc.Kind, &sc.DueOn); err != nil {
			return nil, err
		}
		list = append(list, sc)
	}
	return list, rows.Err()
}

// SetReminderMail switches the monthly reminder mail of a user on or off.
func (s *Store) SetReminderMail(ctx context.Context, personID int64, on bool) (User, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE app_user SET reminder_mail = ? WHERE person_id = ?`, on, personID); err != nil {
		return User{}, err
	}
	return readUser(s.db.QueryRowContext(ctx, userSelect+` WHERE u.person_id = ?`, personID))
}

// maxMailAttempts is how often a monthly mail is tried before it counts as failed.
const maxMailAttempts = 5

// PrepareMonthlyMails decides the mail of the month (YYYY-MM) for every user who wants reminder mails and has none
// for it yet: it holds the open reminders the user is a recipient of and no earlier mail to them contained. Without
// any, the month is recorded as empty, so that reminders becoming due later wait for the next month.
func (s *Store) PrepareMonthlyMails(ctx context.Context, month string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		users, err := readIDs(tx.QueryContext(ctx, `
			SELECT u.person_id FROM app_user u
			WHERE u.reminder_mail = 1 AND NOT EXISTS (SELECT 1 FROM monthly_mail m WHERE m.person_id = u.person_id AND m.month = ?)
			ORDER BY u.person_id`, month))
		if err != nil {
			return err
		}
		for _, u := range users {
			ids, err := readIDs(tx.QueryContext(ctx, `
				SELECT r.id FROM reminder r
				WHERE r.done_at IS NULL AND `+recipientCondition+`
					AND NOT EXISTS (SELECT 1 FROM monthly_mail_reminder mr WHERE mr.person_id = ? AND mr.reminder_id = r.id)
				ORDER BY r.due_on, r.id`, u, u))
			if err != nil {
				return err
			}
			state := "pending"
			if len(ids) == 0 {
				state = "empty"
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO monthly_mail (person_id, month, state) VALUES (?, ?, ?)`, u, month, state); err != nil {
				return err
			}
			for _, id := range ids {
				if _, err := tx.ExecContext(ctx, `INSERT INTO monthly_mail_reminder (person_id, month, reminder_id) VALUES (?, ?, ?)`, u, month, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// MonthlyMail is a mail still to be sent, with the reminders in it that are still open.
type MonthlyMail struct {
	To        User
	Month     string
	Reminders []Reminder
}

// PendingMonthlyMails returns the mails still to be sent, including those to try again after an error.
func (s *Store) PendingMonthlyMails(ctx context.Context) ([]MonthlyMail, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT person_id, month FROM monthly_mail WHERE state = 'pending' ORDER BY month, person_id`)
	if err != nil {
		return nil, err
	}
	type key struct {
		person int64
		month  string
	}
	var keys []key
	for rows.Next() {
		var k key
		if err := rows.Scan(&k.person, &k.month); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var mails []MonthlyMail
	for _, k := range keys {
		u, err := readUser(s.db.QueryRowContext(ctx, userSelect+` WHERE u.person_id = ?`, k.person))
		if err != nil {
			return nil, err
		}
		list, err := readReminders(ctx, s.db, reminderSelect+`
			JOIN monthly_mail_reminder mr ON mr.reminder_id = r.id AND mr.person_id = ? AND mr.month = ?
			WHERE r.done_at IS NULL ORDER BY r.due_on, r.id`, k.person, k.month)
		if err != nil {
			return nil, err
		}
		mails = append(mails, MonthlyMail{To: u, Month: k.month, Reminders: list})
	}
	return mails, nil
}

// MailOutcome records the attempt to send a monthly mail: sent, nothing left to send (empty) or failed with an
// error, which after maxMailAttempts attempts ends the trying. It returns whether the mail has finally failed.
func (s *Store) MailOutcome(ctx context.Context, personID int64, month string, empty bool, sendErr error, now time.Time) (bool, error) {
	switch {
	case empty:
		_, err := s.db.ExecContext(ctx, `UPDATE monthly_mail SET state = 'empty' WHERE person_id = ? AND month = ?`, personID, month)
		return false, err
	case sendErr == nil:
		_, err := s.db.ExecContext(ctx, `UPDATE monthly_mail SET state = 'sent', attempts = attempts + 1, sent_at = ?, last_error = NULL
			WHERE person_id = ? AND month = ?`, timeText(now), personID, month)
		return false, err
	}
	var attempts int
	err := s.db.QueryRowContext(ctx, `
		UPDATE monthly_mail SET attempts = attempts + 1, last_error = ?,
			state = CASE WHEN attempts + 1 >= ? THEN 'failed' ELSE 'pending' END
		WHERE person_id = ? AND month = ? RETURNING attempts`, sendErr.Error(), maxMailAttempts, personID, month).Scan(&attempts)
	return attempts >= maxMailAttempts, err
}
