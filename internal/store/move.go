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
	m.Boxes, err = readBoxes(q.QueryContext(ctx, boxSelect+` WHERE b.move_id = ? GROUP BY b.id ORDER BY b.number`, m.ID))
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
	SELECT b.id, b.move_id, b.number, b.label, COUNT(im.item_id)
	FROM box b LEFT JOIN item_move im ON im.box_id = b.id`

func readBoxes(rows *sql.Rows, err error) ([]Box, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	boxes := []Box{}
	for rows.Next() {
		var b Box
		if err := rows.Scan(&b.ID, &b.MoveID, &b.Number, &b.Label, &b.Count); err != nil {
			return nil, err
		}
		boxes = append(boxes, b)
	}
	return boxes, rows.Err()
}

func readBox(ctx context.Context, q querier, condition string, args ...any) (Box, error) {
	boxes, err := readBoxes(q.QueryContext(ctx, boxSelect+` WHERE `+condition+` GROUP BY b.id`, args...))
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
func (s *Store) CreateBox(ctx context.Context, label string, now time.Time) (Box, error) {
	var b Box
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var moveID int64
		var number int
		err := tx.QueryRowContext(ctx, `
			UPDATE move SET last_box_number = last_box_number + 1 WHERE active = 1 RETURNING id, last_box_number`).
			Scan(&moveID, &number)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoMove
		}
		if err != nil {
			return err
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO box (move_id, number, label, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
			moveID, number, label, timeText(now)).Scan(&id); err != nil {
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

func (s *Store) RenameBox(ctx context.Context, id int64, label string) (Box, error) {
	return s.updateBox(ctx, id, func(tx *sql.Tx, _ Box) error {
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
	TargetRoom *Named
	Box        *Box
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
	if c.TargetRoomID != nil && *c.TargetRoomID != 0 {
		var apartmentID int64
		err := q.QueryRowContext(ctx, `SELECT apartment_id FROM room WHERE id = ?`, *c.TargetRoomID).Scan(&apartmentID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUnknownPlace
		}
		if err != nil {
			return err
		}
		if apartmentID != newApartmentID {
			return ErrTargetRoomNotInNewApartment
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
			WHERE b.id IN (SELECT box_id FROM item_move WHERE move_id = ? AND target_room_id = ?)
			GROUP BY b.id ORDER BY b.number`, m.ID, r.ID))
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

// CompleteMove puts all items with a target room directly into their target room, without storage
// location, and completes the move. Items without target room keep their placement; if such items
// are still in the old apartment, confirmation is needed, otherwise ErrItemsWithoutTargetRoom and
// these items are returned. The move details stay as history.
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
		if _, err := tx.ExecContext(ctx, `
			UPDATE item SET storage_location_id = NULL, updated_at = ?,
				room_id = (SELECT target_room_id FROM item_move WHERE item_id = item.id AND move_id = ?)
			WHERE id IN (SELECT item_id FROM item_move WHERE move_id = ? AND target_room_id IS NOT NULL)`,
			timeText(now), active.ID, active.ID); err != nil {
			return err
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
