package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrArchived    = errors.New("item is archived")
	ErrNotArchived = errors.New("item is not archived")
	ErrNoSale      = errors.New("item is not for sale")
)

// ForSale says whether an item with this disposition can have an asking price.
func (d Disposition) ForSale() bool { return d == Sell || d == SellLater }

// offered says whether an item with this disposition is on offer, so that it counts as offered since a change to it.
func (d Disposition) offered() bool { return d == Sell || d == GiveAway }

func offeredSinceValue(d Disposition, now time.Time) any {
	if !d.offered() {
		return nil
	}
	return timeText(now)
}

func nullCents(c *int64) any {
	if c == nil {
		return nil
	}
	return *c
}

const saleSelect = `asking_price_cents, offered_since, handed_over_on, sale_proceeds_cents`

func saleSelectOf(alias string) string {
	return alias + `.asking_price_cents, ` + alias + `.offered_since, ` + alias + `.handed_over_on, ` + alias + `.sale_proceeds_cents`
}

// saleColumns receives the columns of saleSelect.
type saleColumns struct {
	askingPrice, proceeds      sql.NullInt64
	offeredSince, handedOverOn sql.NullString
}

func (c *saleColumns) targets() []any {
	return []any{&c.askingPrice, &c.offeredSince, &c.handedOverOn, &c.proceeds}
}

func (c *saleColumns) sale() (Sale, error) {
	s := Sale{HandedOverOn: c.handedOverOn.String}
	if c.askingPrice.Valid {
		s.AskingPrice = &c.askingPrice.Int64
	}
	if c.proceeds.Valid {
		s.SaleProceeds = &c.proceeds.Int64
	}
	if c.offeredSince.Valid {
		t, err := parseTime(c.offeredSince.String)
		if err != nil {
			return Sale{}, err
		}
		s.OfferedSince = &t
	}
	return s, nil
}

// touchItem sets updated_at of an item in stock; notFound is the error for an unknown item, archived items give
// ErrArchived.
func touchItem(ctx context.Context, q querier, id int64, now time.Time, notFound error) error {
	var handedOverOn sql.NullString
	err := q.QueryRowContext(ctx, `SELECT handed_over_on FROM item WHERE id = ?`, id).Scan(&handedOverOn)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	if err != nil {
		return err
	}
	if handedOverOn.Valid {
		return ErrArchived
	}
	_, err = q.ExecContext(ctx, `UPDATE item SET updated_at = ? WHERE id = ?`, timeText(now), id)
	return err
}

// setDisposition changes the disposition; every change to Sell or GiveAway starts "offered since" anew, and every
// change starts the period of the reminders anew (spec 07). The asking price stays whatever the disposition becomes.
// Saving the same disposition changes nothing.
func setDisposition(ctx context.Context, q querier, id int64, d Disposition, now time.Time) error {
	var old sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT disposition FROM item WHERE id = ?`, id).Scan(&old); err != nil {
		return err
	}
	if dispositionFrom(old) == d {
		return nil
	}
	if _, err := q.ExecContext(ctx, `UPDATE item SET disposition = ? WHERE id = ?`, dispositionValue(d), id); err != nil {
		return err
	}
	if d.offered() {
		if _, err := q.ExecContext(ctx, `UPDATE item SET offered_since = ? WHERE id = ?`, timeText(now), id); err != nil {
			return err
		}
	}
	return dispositionChanged(ctx, q, id, d, now)
}

// Handover is selling (Disposition Sell) or giving away (GiveAway) an item on a date (YYYY-MM-DD).
type Handover struct {
	Disposition Disposition
	Date        string
	// Proceeds in cents; 0 for giving away.
	Proceeds int64
}

// HandOver puts an item in stock into the archive, from any disposition: it gets the disposition of the handover,
// loses its placement and its details in the active move, and its own storage locations (furniture) go away, so
// that their contents stand in the room without storage location. For an item already in the archive it only
// corrects disposition, date and proceeds.
func (s *Store) HandOver(ctx context.Context, id int64, h Handover, now time.Time) (Item, error) {
	var item Item
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var handedOverOn sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT handed_over_on FROM item WHERE id = ?`, id).Scan(&handedOverOn)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !handedOverOn.Valid {
			if err := removeOwnStorageLocations(ctx, tx, id, now); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE item SET room_id = NULL, storage_location_id = NULL WHERE id = ?`, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM item_move WHERE item_id = ? AND move_id IN (SELECT id FROM move WHERE active = 1)`, id); err != nil {
				return err
			}
		}
		if err := setDisposition(ctx, tx, id, h.Disposition, now); err != nil {
			return err
		}
		// An item in the archive gets no reminders (spec 07).
		if err := settleOpen(ctx, tx, id, reactHandedOver, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM reminder_schedule WHERE item_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE item SET handed_over_on = ?, sale_proceeds_cents = ?, updated_at = ? WHERE id = ?`,
			h.Date, h.Proceeds, timeText(now), id); err != nil {
			return err
		}
		item, err = readItem(ctx, tx, id)
		return err
	})
	return item, err
}

// removeOwnStorageLocations removes the storage locations of an item as furniture; their contents stay in the room.
func removeOwnStorageLocations(ctx context.Context, tx *sql.Tx, id int64, now time.Time) error {
	own, err := readNamed(tx.QueryContext(ctx, `SELECT id, name FROM storage_location WHERE furniture_item_id = ?`, id))
	if err != nil {
		return err
	}
	for _, l := range own {
		// An own storage location may already be gone together with another one it was inside.
		if err := removeStorageLocation(ctx, tx, l.ID, now); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

// TakeBack brings an archived item back into stock, with its disposition and asking price but without placement;
// storage locations that went away with the handover don't come back.
func (s *Store) TakeBack(ctx context.Context, id int64, now time.Time) (Item, error) {
	var item Item
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var handedOverOn sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT handed_over_on FROM item WHERE id = ?`, id).Scan(&handedOverOn)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !handedOverOn.Valid {
			return ErrNotArchived
		}
		if _, err := tx.ExecContext(ctx, `UPDATE item SET handed_over_on = NULL, sale_proceeds_cents = NULL, updated_at = ? WHERE id = ?`,
			timeText(now), id); err != nil {
			return err
		}
		// Back in stock, the reminders of its disposition start from today.
		var d sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT disposition FROM item WHERE id = ?`, id).Scan(&d); err != nil {
			return err
		}
		if err := planFor(ctx, tx, id, dispositionFrom(d), now); err != nil {
			return err
		}
		item, err = readItem(ctx, tx, id)
		return err
	})
	return item, err
}

// Settings returns the stored household settings with these keys; missing keys are not in the result.
func (s *Store) Settings(ctx context.Context, keys ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		var v string
		err := s.db.QueryRowContext(ctx, `SELECT value FROM household_setting WHERE key = ?`, k).Scan(&v)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

// SaveSettings stores household settings in one transaction.
func (s *Store) SaveSettings(ctx context.Context, values map[string]string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for k, v := range values {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO household_setting (key, value) VALUES (?, ?)
				ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// ProceedsEntry is an archived item in the proceeds summary.
type ProceedsEntry struct {
	ItemListEntry
	// Share is the part of the proceeds that counts for the owner in the filter, otherwise all of it.
	Share int64
}

type Proceeds struct {
	// Total is the sum of the shares.
	Total          int64
	SoldCount      int
	GivenAwayCount int
	Entries        []ProceedsEntry
}

// ProceedsSummary sums up the archive in a period (YYYY-MM-DD, both included; empty means open). With an owner,
// only that owner's items count, each with an equal share of the proceeds among its owners.
func (s *Store) ProceedsSummary(ctx context.Context, from, to string, ownerID int64) (Proceeds, error) {
	entries, _, err := listItems(ctx, s.db, ItemFilter{Archived: true, HandedOverFrom: from, HandedOverTo: to, OwnerID: ownerID, Limit: allItems})
	if err != nil {
		return Proceeds{}, err
	}
	p := Proceeds{Entries: []ProceedsEntry{}}
	for _, e := range entries {
		share := *e.SaleProceeds
		if ownerID != 0 {
			owners, err := readOwners(ctx, s.db, e.ID)
			if err != nil {
				return Proceeds{}, err
			}
			share = shareOf(share, owners, ownerID)
		}
		p.Total += share
		if e.Disposition == GiveAway {
			p.GivenAwayCount++
		} else {
			p.SoldCount++
		}
		p.Entries = append(p.Entries, ProceedsEntry{ItemListEntry: e, Share: share})
	}
	return p, nil
}

// shareOf splits cents equally among the owners in their order (by name); the cents left over go one each to the
// first owners, so that the shares of all owners add up to the whole amount.
func shareOf(cents int64, owners []Person, ownerID int64) int64 {
	n := int64(len(owners))
	for i, o := range owners {
		if o.ID == ownerID {
			share := cents / n
			if int64(i) < cents%n {
				share++
			}
			return share
		}
	}
	return 0
}
