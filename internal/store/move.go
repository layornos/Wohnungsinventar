package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNoMove                      = errors.New("no active move")
	ErrMoveInProgress              = errors.New("a move is already active")
	ErrMoveCompleted               = errors.New("move is completed")
	ErrSameApartment               = errors.New("old and new apartment are the same")
	ErrUnknownApartment            = errors.New("unknown apartment")
	ErrApartmentInMove             = errors.New("apartment belongs to a move")
	ErrRoomInMove                  = errors.New("room occurs in a move")
	ErrBoxNotEmpty                 = errors.New("box is not empty")
	ErrUnknownBox                  = errors.New("box does not belong to the active move")
	ErrTargetRoomNotInNewApartment = errors.New("target room is not in the new apartment")
	ErrItemsWithoutTargetRoom      = errors.New("items of the old apartment without target room")
	ErrUnknownItem                 = errors.New("unknown item")
	ErrPlaceNotInNewApartment      = errors.New("place is not in the new apartment")
)

type Move struct {
	ID           int64
	OldApartment Named
	NewApartment Named
	Active       bool
	CreatedAt    time.Time
	CompletedAt  *time.Time
	Boxes        []Box
}

type Box struct {
	ID     int64
	MoveID int64
	Number int
	Label  string
	// TargetRoom is where the items in the box go unless they have a target room of their own; nil if none.
	TargetRoom *Named
	// Count is the number of items in the box.
	Count int
}

const moveSelect = `
	SELECT m.id, o.id, o.name, n.id, n.name, m.active = 1, m.created_at, m.completed_at
	FROM move m JOIN apartment o ON o.id = m.old_apartment_id JOIN apartment n ON n.id = m.new_apartment_id`

func scanMove(r row) (Move, error) {
	var (
		m         Move
		created   string
		completed sql.NullString
	)
	if err := r.Scan(&m.ID, &m.OldApartment.ID, &m.OldApartment.Name, &m.NewApartment.ID, &m.NewApartment.Name,
		&m.Active, &created, &completed); err != nil {
		return Move{}, err
	}
	var err error
	if m.CreatedAt, err = parseTime(created); err != nil {
		return Move{}, err
	}
	if completed.Valid {
		t, err := parseTime(completed.String)
		if err != nil {
			return Move{}, err
		}
		m.CompletedAt = &t
	}
	return m, nil
}

// readMove reads a move including its boxes; without ID the active one.
func readMove(ctx context.Context, q querier, id int64) (Move, error) {
	query, args := moveSelect+` WHERE m.active = 1`, []any{}
	if id != 0 {
		query, args = moveSelect+` WHERE m.id = ?`, []any{id}
	}
	m, err := scanMove(q.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		if id == 0 {
			return Move{}, ErrNoMove
		}
		return Move{}, ErrNotFound
	}
	if err != nil {
		return Move{}, err
	}
	m.Boxes, err = readBoxes(q.QueryContext(ctx, boxSelect+` WHERE b.move_id = ? ORDER BY b.number`, m.ID))
	return m, err
}

func (s *Store) ActiveMove(ctx context.Context) (Move, error) { return readMove(ctx, s.db, 0) }

func (s *Store) GetMove(ctx context.Context, id int64) (Move, error) {
	return readMove(ctx, s.db, id)
}

// ListMoves returns the completed moves, most recent first.
func (s *Store) ListMoves(ctx context.Context) ([]Move, error) {
	rows, err := s.db.QueryContext(ctx, moveSelect+` WHERE m.active = 0 ORDER BY m.completed_at DESC, m.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	moves := []Move{}
	for rows.Next() {
		m, err := scanMove(rows)
		if err != nil {
			return nil, err
		}
		moves = append(moves, m)
	}
	return moves, rows.Err()
}

func (s *Store) CreateMove(ctx context.Context, oldApartmentID, newApartmentID int64, now time.Time) (Move, error) {
	var m Move
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if oldApartmentID == newApartmentID {
			return ErrSameApartment
		}
		var both, active bool
		err := tx.QueryRowContext(ctx, `
			SELECT (SELECT COUNT(*) FROM apartment WHERE id IN (?, ?)) = 2, EXISTS (SELECT 1 FROM move WHERE active = 1)`,
			oldApartmentID, newApartmentID).Scan(&both, &active)
		switch {
		case err != nil:
			return err
		case !both:
			return ErrUnknownApartment
		case active:
			return ErrMoveInProgress
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO move (old_apartment_id, new_apartment_id, created_at) VALUES (?, ?, ?) RETURNING id`,
			oldApartmentID, newApartmentID, timeText(now)).Scan(&id); err != nil {
			return err
		}
		m, err = readMove(ctx, tx, id)
		return err
	})
	return m, err
}

const boxSelect = `
	SELECT b.id, b.move_id, b.number, b.label, tr.id, tr.name, (SELECT COUNT(*) FROM item_move im WHERE im.box_id = b.id)
	FROM box b LEFT JOIN room tr ON tr.id = b.target_room_id`

func readBoxes(rows *sql.Rows, err error) ([]Box, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	boxes := []Box{}
	for rows.Next() {
		var (
			b          Box
			targetID   sql.NullInt64
			targetName sql.NullString
		)
		if err := rows.Scan(&b.ID, &b.MoveID, &b.Number, &b.Label, &targetID, &targetName, &b.Count); err != nil {
			return nil, err
		}
		b.TargetRoom = namedFrom(targetID, targetName)
		boxes = append(boxes, b)
	}
	return boxes, rows.Err()
}

func readBox(ctx context.Context, q querier, condition string, args ...any) (Box, error) {
	boxes, err := readBoxes(q.QueryContext(ctx, boxSelect+` WHERE `+condition, args...))
	if err != nil {
		return Box{}, err
	}
	if len(boxes) == 0 {
		return Box{}, ErrNotFound
	}
	return boxes[0], nil
}

func (s *Store) GetBox(ctx context.Context, id int64) (Box, error) {
	return readBox(ctx, s.db, `b.id = ?`, id)
}

// BoxByNumber returns the box with the number in the active move.
func (s *Store) BoxByNumber(ctx context.Context, number int) (Box, error) {
	if _, err := readMove(ctx, s.db, 0); err != nil {
		return Box{}, err
	}
	return readBox(ctx, s.db, `b.number = ? AND b.move_id = (SELECT id FROM move WHERE active = 1)`, number)
}

// CreateBox creates a box in the active move with the next number, never given out before.
// targetRoomID 0 means without target room.
func (s *Store) CreateBox(ctx context.Context, label string, targetRoomID int64, now time.Time) (Box, error) {
	var b Box
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var moveID, newApartmentID int64
		var number int
		err := tx.QueryRowContext(ctx, `
			UPDATE move SET last_box_number = last_box_number + 1 WHERE active = 1 RETURNING id, new_apartment_id, last_box_number`).
			Scan(&moveID, &newApartmentID, &number)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoMove
		}
		if err != nil {
			return err
		}
		if err := checkTargetRoom(ctx, tx, targetRoomID, newApartmentID); err != nil {
			return err
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO box (move_id, number, label, target_room_id, created_at) VALUES (?, ?, ?, ?, ?) RETURNING id`,
			moveID, number, label, nullID(targetRoomID), timeText(now)).Scan(&id); err != nil {
			return err
		}
		b, err = readBox(ctx, tx, `b.id = ?`, id)
		return err
	})
	return b, err
}

// updateBox runs a change to a box of the active move.
func (s *Store) updateBox(ctx context.Context, id int64, change func(tx *sql.Tx, b Box) error) (Box, error) {
	var b Box
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := moveIsActive(ctx, tx, `SELECT move_id FROM box WHERE id = ?`, id); err != nil {
			return err
		}
		before, err := readBox(ctx, tx, `b.id = ?`, id)
		if err != nil {
			return err
		}
		if err := change(tx, before); err != nil {
			return err
		}
		b, err = readBox(ctx, tx, `b.id = ?`, id)
		if errors.Is(err, ErrNotFound) {
			err = nil
		}
		return err
	})
	return b, err
}

// moveIsActive checks that the move returned by the query is still active.
func moveIsActive(ctx context.Context, q querier, moveQuery string, args ...any) error {
	var active bool
	err := q.QueryRowContext(ctx, `SELECT active = 1 FROM move WHERE id = (`+moveQuery+`)`, args...).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !active {
		return ErrMoveCompleted
	}
	return nil
}

// EditBox sets the label of a box and, unless targetRoomID is nil, its target room (a pointer to 0 removes it).
func (s *Store) EditBox(ctx context.Context, id int64, label string, targetRoomID *int64) (Box, error) {
	return s.updateBox(ctx, id, func(tx *sql.Tx, b Box) error {
		if targetRoomID != nil {
			var newApartmentID int64
			if err := tx.QueryRowContext(ctx, `SELECT new_apartment_id FROM move WHERE id = ?`, b.MoveID).Scan(&newApartmentID); err != nil {
				return err
			}
			if err := checkTargetRoom(ctx, tx, *targetRoomID, newApartmentID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE box SET target_room_id = ? WHERE id = ?`, nullID(*targetRoomID), id); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE box SET label = ? WHERE id = ?`, label, id)
		return err
	})
}

func (s *Store) DeleteBox(ctx context.Context, id int64) error {
	_, err := s.updateBox(ctx, id, func(tx *sql.Tx, b Box) error {
		if b.Count > 0 {
			return ErrBoxNotEmpty
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM box WHERE id = ?`, id)
		return err
	})
	return err
}

// MoveDetails are the origin room, target room and box of an item in the active move.
type MoveDetails struct {
	OriginRoom *Named
	// TargetRoom is the effective target room: the item's own one, otherwise the one of its box.
	TargetRoom *Named
	// TargetRoomFromBox tells that TargetRoom comes from the box.
	TargetRoomFromBox bool
	Box               *Box
}

// useBoxTargetRoom makes the target room of the box the effective one if the item has none of its own.
func (d *MoveDetails) useBoxTargetRoom() {
	if d.TargetRoom == nil && d.Box != nil && d.Box.TargetRoom != nil {
		d.TargetRoom = d.Box.TargetRoom
		d.TargetRoomFromBox = true
	}
}

// MoveDetailsChange changes move details: nil leaves a detail as it is, a pointer to 0 resets it.
type MoveDetailsChange struct {
	OriginRoomID *int64
	TargetRoomID *int64
	BoxID        *int64
}

func (s *Store) SetMoveDetails(ctx context.Context, itemID int64, c MoveDetailsChange, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		return setMoveDetails(ctx, tx, itemID, c)
	})
}

// SetMoveDetailsBulk changes the details of several items, all or nothing.
func (s *Store) SetMoveDetailsBulk(ctx context.Context, itemIDs []int64, c MoveDetailsChange, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for _, id := range itemIDs {
			res, err := tx.ExecContext(ctx, `UPDATE item SET updated_at = ? WHERE id = ?`, timeText(now), id)
			if err != nil {
				return err
			}
			if n, err := res.RowsAffected(); err != nil || n == 0 {
				return errors.Join(err, ErrUnknownItem)
			}
			if err := setMoveDetails(ctx, tx, id, c); err != nil {
				return err
			}
		}
		return nil
	})
}

// setMoveDetails creates the move details if needed, with the current room of the item as
// origin room, and then changes the given fields.
func setMoveDetails(ctx context.Context, q querier, itemID int64, c MoveDetailsChange) error {
	var moveID, newApartmentID int64
	err := q.QueryRowContext(ctx, `SELECT id, new_apartment_id FROM move WHERE active = 1`).Scan(&moveID, &newApartmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoMove
	}
	if err != nil {
		return err
	}
	if c.TargetRoomID != nil {
		if err := checkTargetRoom(ctx, q, *c.TargetRoomID, newApartmentID); err != nil {
			return err
		}
	}
	if c.OriginRoomID != nil && *c.OriginRoomID != 0 {
		if err := checkPlace(ctx, q, *c.OriginRoomID, 0); err != nil {
			return err
		}
	}
	if c.BoxID != nil && *c.BoxID != 0 {
		var fits bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM box WHERE id = ? AND move_id = ?)`, *c.BoxID, moveID).Scan(&fits); err != nil {
			return err
		}
		if !fits {
			return ErrUnknownBox
		}
	}

	room, err := currentRoom(ctx, q, itemID)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `
		INSERT INTO item_move (item_id, move_id, origin_room_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		itemID, moveID, nullID(room)); err != nil {
		return err
	}
	for column, value := range map[string]*int64{"origin_room_id": c.OriginRoomID, "target_room_id": c.TargetRoomID, "box_id": c.BoxID} {
		if value == nil {
			continue
		}
		if _, err := q.ExecContext(ctx, `UPDATE item_move SET `+column+` = ? WHERE item_id = ? AND move_id = ?`,
			nullID(*value), itemID, moveID); err != nil {
			return err
		}
	}
	return nil
}

// checkTargetRoom checks that a target room is in the new apartment; 0 (no target room) is fine.
func checkTargetRoom(ctx context.Context, q querier, roomID, newApartmentID int64) error {
	if roomID == 0 {
		return nil
	}
	var apartmentID int64
	err := q.QueryRowContext(ctx, `SELECT apartment_id FROM room WHERE id = ?`, roomID).Scan(&apartmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownPlace
	}
	if err != nil {
		return err
	}
	if apartmentID != newApartmentID {
		return ErrTargetRoomNotInNewApartment
	}
	return nil
}

// currentRoom is the room an item is placed in right now, via its storage location or directly; 0 without placement.
func currentRoom(ctx context.Context, q querier, itemID int64) (int64, error) {
	var storageLocationID, roomID sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT storage_location_id, room_id FROM item WHERE id = ?`, itemID).Scan(&storageLocationID, &roomID); err != nil {
		return 0, err
	}
	p, err := readPlacement(ctx, q, storageLocationID.Int64, roomID.Int64)
	if err != nil || p == nil {
		return 0, err
	}
	return p.Room.ID, nil
}

// readMoveDetails reads the move details of an item in the active move; nil if there are none.
func readMoveDetails(ctx context.Context, q querier, itemID int64) (*MoveDetails, error) {
	var (
		originID, targetID, boxID sql.NullInt64
		originName, targetName    sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT o.id, o.name, t.id, t.name, im.box_id
		FROM item_move im JOIN move m ON m.id = im.move_id AND m.active = 1
		LEFT JOIN room o ON o.id = im.origin_room_id LEFT JOIN room t ON t.id = im.target_room_id
		WHERE im.item_id = ?`, itemID).Scan(&originID, &originName, &targetID, &targetName, &boxID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := &MoveDetails{OriginRoom: namedFrom(originID, originName), TargetRoom: namedFrom(targetID, targetName)}
	if boxID.Valid {
		b, err := readBox(ctx, q, `b.id = ?`, boxID.Int64)
		if err != nil {
			return nil, err
		}
		d.Box = &b
	}
	d.useBoxTargetRoom()
	return d, nil
}

func namedFrom(id sql.NullInt64, name sql.NullString) *Named {
	if !id.Valid {
		return nil
	}
	return &Named{ID: id.Int64, Name: name.String}
}

// TargetRoomEntry is a room of the new apartment with the boxes and items that go there.
type TargetRoomEntry struct {
	Room  Named
	Boxes []Box
	Items []ItemListEntry
}

func (s *Store) TargetRoomOverview(ctx context.Context) ([]TargetRoomEntry, error) {
	m, err := readMove(ctx, s.db, 0)
	if err != nil {
		return nil, err
	}
	rooms, err := readNamed(s.db.QueryContext(ctx, `SELECT id, name FROM room WHERE apartment_id = ? ORDER BY name, id`, m.NewApartment.ID))
	if err != nil {
		return nil, err
	}
	overview := []TargetRoomEntry{}
	for _, r := range rooms {
		boxes, err := readBoxes(s.db.QueryContext(ctx, boxSelect+`
			WHERE b.move_id = ? AND (b.target_room_id = ?
				OR b.id IN (SELECT box_id FROM item_move WHERE move_id = ? AND target_room_id = ?))
			ORDER BY b.number`, m.ID, r.ID, m.ID, r.ID))
		if err != nil {
			return nil, err
		}
		items, _, err := s.ListItems(ctx, ItemFilter{TargetRoomID: r.ID, Limit: allItems})
		if err != nil {
			return nil, err
		}
		overview = append(overview, TargetRoomEntry{Room: r, Boxes: boxes, Items: items})
	}
	return overview, nil
}

// activeBox returns the box if it belongs to the active move, together with that move.
func activeBox(ctx context.Context, q querier, boxID int64) (Move, Box, error) {
	m, err := readMove(ctx, q, 0)
	if err != nil {
		return Move{}, Box{}, err
	}
	b, err := readBox(ctx, q, `b.id = ?`, boxID)
	if err != nil {
		return Move{}, Box{}, err
	}
	if b.MoveID != m.ID {
		return Move{}, Box{}, ErrMoveCompleted
	}
	return m, b, nil
}

// PackStorageLocation packs all items placed in the storage location or one of its nested ones
// into the box and returns how many were not in it before. The piece of furniture the storage
// location belongs to is not inside it and so stays out.
func (s *Store) PackStorageLocation(ctx context.Context, boxID, storageLocationID int64, now time.Time) (int, Box, error) {
	var (
		count int
		b     Box
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		m, _, err := activeBox(ctx, tx, boxID)
		if err != nil {
			return err
		}
		if err := checkPlace(ctx, tx, 0, storageLocationID); err != nil {
			return err
		}
		ids, err := readIDs(tx.QueryContext(ctx, withSubtree+`
			SELECT i.id FROM item i WHERE i.storage_location_id IN (SELECT id FROM subtree)
				AND NOT EXISTS (SELECT 1 FROM item_move im WHERE im.item_id = i.id AND im.move_id = ? AND im.box_id = ?)
			ORDER BY i.id`, storageLocationID, m.ID, boxID))
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx, `UPDATE item SET updated_at = ? WHERE id = ?`, timeText(now), id); err != nil {
				return err
			}
			if err := setMoveDetails(ctx, tx, id, MoveDetailsChange{BoxID: &boxID}); err != nil {
				return err
			}
		}
		count = len(ids)
		b, err = readBox(ctx, tx, `b.id = ?`, boxID)
		return err
	})
	return count, b, err
}

// UnpackBox places items of the box for good at a place of the new apartment, takes them out of
// the box and makes the room of the place their target room. itemIDs nil means all items of the
// box. Furniture takes its storage locations along into that room.
func (s *Store) UnpackBox(ctx context.Context, boxID int64, itemIDs []int64, p Place, now time.Time) (int, Box, error) {
	var (
		count int
		b     Box
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		m, _, err := activeBox(ctx, tx, boxID)
		if err != nil {
			return err
		}
		if err := checkPlace(ctx, tx, p.RoomID, p.StorageLocationID); err != nil {
			return err
		}
		path, err := readPlacement(ctx, tx, p.StorageLocationID, p.RoomID)
		if err != nil {
			return err
		}
		if path.Apartment.ID != m.NewApartment.ID {
			return ErrPlaceNotInNewApartment
		}
		ids := itemIDs
		if ids == nil {
			if ids, err = readIDs(tx.QueryContext(ctx, `SELECT item_id FROM item_move WHERE box_id = ? ORDER BY item_id`, boxID)); err != nil {
				return err
			}
		}
		seen := map[int64]bool{}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			var inBox bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM item_move WHERE item_id = ? AND box_id = ?)`, id, boxID).Scan(&inBox); err != nil {
				return err
			}
			if !inBox {
				return ErrUnknownItem
			}
			if _, err := tx.ExecContext(ctx, `UPDATE item SET updated_at = ? WHERE id = ?`, timeText(now), id); err != nil {
				return err
			}
			if err := setPlace(ctx, tx, id, p); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE item_move SET box_id = NULL, target_room_id = ? WHERE item_id = ? AND move_id = ?`,
				path.Room.ID, id, m.ID); err != nil {
				return err
			}
			if err := carryStorageLocations(ctx, tx, id, path.Room.ID); err != nil {
				return err
			}
		}
		count = len(seen)
		b, err = readBox(ctx, tx, `b.id = ?`, boxID)
		return err
	})
	return count, b, err
}

// carryStorageLocations puts the storage locations of a piece of furniture, with everything in and
// below them, into the room. Storage locations of the furniture nested in one another stay so.
func carryStorageLocations(ctx context.Context, q querier, furnitureID, roomID int64) error {
	_, err := q.ExecContext(ctx, `WITH RECURSIVE `+subtree("own", "furniture_item_id = ?")+`
		UPDATE storage_location SET room_id = ?, parent_id = NULL
		WHERE furniture_item_id = ? AND (parent_id IS NULL OR parent_id NOT IN (SELECT id FROM own))`,
		furnitureID, roomID, furnitureID)
	return err
}

// CompleteMove completes the active move. Items already placed in the new apartment stay as they
// are. All other items with an effective target room are put directly into it, without storage
// location; furniture among them takes its storage locations along, and items in there without a
// target room of their own keep their placement. Items without target room keep their placement;
// if such items are still in the old apartment, confirmation is needed, otherwise
// ErrItemsWithoutTargetRoom and these items are returned. The move details stay as history.
func (s *Store) CompleteMove(ctx context.Context, confirmed bool, now time.Time) (Move, []ItemListEntry, error) {
	var (
		m       Move
		without []ItemListEntry
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		active, err := readMove(ctx, tx, 0)
		if err != nil {
			return err
		}
		if !confirmed {
			if without, _, err = listItems(ctx, tx, ItemFilter{MoveID: active.ID, WithoutTargetRoom: true, Limit: allItems}); err != nil {
				return err
			}
			if len(without) > 0 {
				return ErrItemsWithoutTargetRoom
			}
		}
		rows, err := tx.QueryContext(ctx, `WITH RECURSIVE `+subtree("in_new", "room_id IN (SELECT id FROM room WHERE apartment_id = ?)")+`
			SELECT i.id, COALESCE(im.target_room_id, b.target_room_id),
				EXISTS (SELECT 1 FROM storage_location WHERE furniture_item_id = i.id)
			FROM item i JOIN item_move im ON im.item_id = i.id AND im.move_id = ?
			LEFT JOIN box b ON b.id = im.box_id
			WHERE COALESCE(im.target_room_id, b.target_room_id) IS NOT NULL
				AND NOT COALESCE(i.room_id IN (SELECT id FROM room WHERE apartment_id = ?), FALSE)
				AND NOT COALESCE(i.storage_location_id IN (SELECT id FROM in_new), FALSE)
			ORDER BY i.id`, active.NewApartment.ID, active.ID, active.NewApartment.ID)
		if err != nil {
			return err
		}
		type placing struct {
			itemID, roomID int64
			furniture      bool
		}
		var all []placing
		for rows.Next() {
			var p placing
			if err := rows.Scan(&p.itemID, &p.roomID, &p.furniture); err != nil {
				rows.Close()
				return err
			}
			all = append(all, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		// First all items, so that items with a target room of their own leave the furniture before it moves.
		for _, p := range all {
			if _, err := tx.ExecContext(ctx, `UPDATE item SET storage_location_id = NULL, room_id = ?, updated_at = ? WHERE id = ?`,
				p.roomID, timeText(now), p.itemID); err != nil {
				return err
			}
		}
		for _, p := range all {
			if p.furniture {
				if err := carryStorageLocations(ctx, tx, p.itemID, p.roomID); err != nil {
					return err
				}
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE move SET active = 0, completed_at = ? WHERE id = ?`,
			timeText(now), active.ID); err != nil {
			return err
		}
		m, err = readMove(ctx, tx, active.ID)
		return err
	})
	if errors.Is(err, ErrItemsWithoutTargetRoom) {
		return Move{}, without, err
	}
	return m, nil, err
}
