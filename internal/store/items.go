package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrLastPhoto = errors.New("last photo of an item")

// Disposition says what is to happen to an item; Undecided means that is not settled yet.
type Disposition string

const (
	Undecided Disposition = ""
	Keep      Disposition = "keep"
	Sell      Disposition = "sell"
	SellLater Disposition = "sell_later"
	GiveAway  Disposition = "give_away"
)

func (d Disposition) Valid() bool {
	switch d {
	case Undecided, Keep, Sell, SellLater, GiveAway:
		return true
	}
	return false
}

type Item struct {
	ID          int64
	Name        string
	Description string
	Disposition Disposition
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Owners      []Person
	// Photos in their order; exactly one is the main photo.
	Photos []Photo
	// Placement is nil as long as the item is not placed.
	Placement *Path
	// FurnitureStorageLocations are the storage locations that belong to this item as furniture.
	FurnitureStorageLocations []Named
	// Move holds the details in the active move; nil if none is active or there are none.
	Move *MoveDetails
}

type Photo struct {
	ID        int64
	ItemID    int64
	File      string
	Type      string
	MainPhoto bool
}

// NewPhoto is an already stored file that gets assigned to an item.
type NewPhoto struct {
	File string
	Type string
}

type NewItem struct {
	Name        string
	Description string
	Disposition Disposition
	OwnerIDs    []int64
	Photos      []NewPhoto
	Place       Place
	// BoxID packs the new item straight into a box of the active move.
	BoxID int64
}

// querier is what *sql.DB and *sql.Tx have in common.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// dispositionFrom reads the stored disposition; NULL means undecided.
func dispositionFrom(n sql.NullString) Disposition { return Disposition(n.String) }

func dispositionValue(d Disposition) any {
	if d == Undecided {
		return nil
	}
	return string(d)
}

// CreateItem creates an item with its owners and photos; the first photo becomes the main photo.
// The caller makes sure there is at least one owner and one photo.
func (s *Store) CreateItem(ctx context.Context, n NewItem, now time.Time) (Item, error) {
	var item Item
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO item (name, name_lower, description, disposition, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
			n.Name, strings.ToLower(n.Name), n.Description, dispositionValue(n.Disposition), timeText(now), timeText(now)).Scan(&id)
		if err != nil {
			return err
		}
		for _, p := range n.OwnerIDs {
			if err := insertOwner(ctx, tx, id, p); err != nil {
				return err
			}
		}
		if err := insertPhotos(ctx, tx, id, n.Photos); err != nil {
			return err
		}
		if err := setPlace(ctx, tx, id, n.Place); err != nil {
			return err
		}
		if n.BoxID != 0 {
			if err := setMoveDetails(ctx, tx, id, MoveDetailsChange{BoxID: &n.BoxID}); err != nil {
				return err
			}
		}
		item, err = readItem(ctx, tx, id)
		return err
	})
	return item, err
}

func insertOwner(ctx context.Context, q querier, itemID, personID int64) error {
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM person WHERE id = ?)`, personID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrUnknownPerson
	}
	_, err := q.ExecContext(ctx, `INSERT INTO ownership (item_id, person_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, itemID, personID)
	return err
}

// insertPhotos appends photos at the end; if the item has no main photo yet, the first new one becomes it.
func insertPhotos(ctx context.Context, q querier, itemID int64, photos []NewPhoto) error {
	for _, p := range photos {
		_, err := q.ExecContext(ctx, `
			INSERT INTO photo (item_id, file, content_type, main_photo, sort_order)
			SELECT ?, ?, ?,
				CASE WHEN EXISTS (SELECT 1 FROM photo WHERE item_id = ? AND main_photo = 1) THEN 0 ELSE 1 END,
				COALESCE((SELECT MAX(sort_order) FROM photo WHERE item_id = ?), 0) + 1`,
			itemID, p.File, p.Type, itemID, itemID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetItem(ctx context.Context, id int64) (Item, error) {
	return readItem(ctx, s.db, id)
}

func readItem(ctx context.Context, q querier, id int64) (Item, error) {
	var (
		item                      Item
		disposition               sql.NullString
		createdAt, updatedAt      string
		storageLocationID, roomID sql.NullInt64
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, description, disposition, created_at, updated_at, storage_location_id, room_id FROM item WHERE id = ?`, id).
		Scan(&item.ID, &item.Name, &item.Description, &disposition, &createdAt, &updatedAt, &storageLocationID, &roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	item.Disposition = dispositionFrom(disposition)
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return Item{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Item{}, err
	}
	if item.Owners, err = readOwners(ctx, q, id); err != nil {
		return Item{}, err
	}
	if item.Placement, err = readPlacement(ctx, q, storageLocationID.Int64, roomID.Int64); err != nil {
		return Item{}, err
	}
	if item.FurnitureStorageLocations, err = readNamed(q.QueryContext(ctx,
		`SELECT id, name FROM storage_location WHERE furniture_item_id = ? ORDER BY name, id`, id)); err != nil {
		return Item{}, err
	}
	if item.Move, err = readMoveDetails(ctx, q, id); err != nil {
		return Item{}, err
	}
	item.Photos, err = readPhotos(ctx, q, id)
	return item, err
}

func readPlacement(ctx context.Context, q querier, storageLocationID, roomID int64) (*Path, error) {
	var (
		p   Path
		err error
	)
	switch {
	case storageLocationID != 0:
		p, err = storageLocationPath(ctx, q, storageLocationID)
	case roomID != 0:
		p, err = roomPath(ctx, q, roomID)
	default:
		return nil, nil
	}
	return &p, err
}

func readOwners(ctx context.Context, q querier, itemID int64) ([]Person, error) {
	return readPersons(q.QueryContext(ctx,
		personSelect+` JOIN ownership o ON o.person_id = p.id WHERE o.item_id = ? ORDER BY p.name, p.id`, itemID))
}

const photoSelect = `SELECT id, item_id, file, content_type, main_photo = 1 FROM photo`

// row is what *sql.Row and *sql.Rows have in common for reading.
type row interface{ Scan(dest ...any) error }

func scanPhoto(r row) (Photo, error) {
	var p Photo
	err := r.Scan(&p.ID, &p.ItemID, &p.File, &p.Type, &p.MainPhoto)
	return p, err
}

func readPhotos(ctx context.Context, q querier, itemID int64) ([]Photo, error) {
	rows, err := q.QueryContext(ctx, photoSelect+` WHERE item_id = ? ORDER BY sort_order, id`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []Photo
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

// ItemListEntry is an item in the list, with its main photo for the thumbnail.
type ItemListEntry struct {
	ID            int64
	Name          string
	Disposition   Disposition
	MainPhotoID   int64
	MainPhotoFile string
	// Place is the name of the storage location or, without one, of the room the item is in; empty without placement.
	Place string
	// StorageLocationID is the storage location the item is placed in directly; 0 if none.
	StorageLocationID int64
	// PlacePath is the path of the placement below the apartment, e.g. "Arbeitszimmer › Schreibtisch › oben";
	// empty without placement.
	PlacePath string
	// Move holds the move details in the active move (or the one named in the filter).
	Move *MoveDetails
}

type ItemFilter struct {
	// Text must occur in the name, ignoring case.
	Text    string
	OwnerID int64
	// Disposition filters when set; a pointer to Undecided returns the items without a disposition.
	Disposition *Disposition
	// RoomID and StorageLocationID also include items in nested storage locations.
	RoomID            int64
	StorageLocationID int64
	// FurnitureID returns the items in the storage locations of this piece of furniture.
	FurnitureID int64
	Unstored    bool
	// Move filters refer to the active move, unless MoveID names another one.
	MoveID       int64
	BoxID        int64
	TargetRoomID int64
	OriginRoomID int64
	// WithoutTargetRoom returns the items that are in the old apartment and have no target room yet.
	WithoutTargetRoom bool
	Limit             int
	Offset            int
}

// subtree is a recursive common table expression that collects, starting from the storage
// locations matching start, all nested storage locations.
func subtree(name, start string) string {
	return name + ` (id) AS (
		SELECT id FROM storage_location WHERE ` + start + `
		UNION ALL
		SELECT l.id FROM storage_location l JOIN ` + name + ` t ON l.parent_id = t.id)`
}

// furnitureWithTargetRoom selects the items with an effective target room in the move given as
// placeholder (NULL for the active one). Used for furniture that takes its storage locations along.
const furnitureWithTargetRoom = `
	SELECT fm.item_id FROM item_move fm LEFT JOIN box fb ON fb.id = fm.box_id
	WHERE fm.move_id = COALESCE(?, (SELECT id FROM move WHERE active = 1))
		AND COALESCE(fm.target_room_id, fb.target_room_id) IS NOT NULL`

// ListItems returns the matching items, most recently updated first, and whether there are more.
func (s *Store) ListItems(ctx context.Context, f ItemFilter) ([]ItemListEntry, bool, error) {
	return listItems(ctx, s.db, f)
}

// allItems is the limit for lists that are meant to be complete, e.g. packing lists.
const allItems = 10_000

func listItems(ctx context.Context, q querier, f ItemFilter) ([]ItemListEntry, bool, error) {
	var (
		ctes, conditions []string
		cteArgs, args    []any
	)
	if f.Text != "" {
		conditions = append(conditions, `i.name_lower LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(strings.ToLower(f.Text))+"%")
	}
	if f.OwnerID != 0 {
		conditions = append(conditions, `EXISTS (SELECT 1 FROM ownership o WHERE o.item_id = i.id AND o.person_id = ?)`)
		args = append(args, f.OwnerID)
	}
	if f.Disposition != nil {
		if *f.Disposition == Undecided {
			conditions = append(conditions, `i.disposition IS NULL`)
		} else {
			conditions = append(conditions, `i.disposition = ?`)
			args = append(args, string(*f.Disposition))
		}
	}
	if f.RoomID != 0 {
		ctes = append(ctes, subtree("in_room", "room_id = ?"))
		cteArgs = append(cteArgs, f.RoomID)
		conditions = append(conditions, `(i.room_id = ? OR i.storage_location_id IN (SELECT id FROM in_room))`)
		args = append(args, f.RoomID)
	}
	if f.StorageLocationID != 0 {
		ctes = append(ctes, subtree("in_storage_location", "id = ?"))
		cteArgs = append(cteArgs, f.StorageLocationID)
		conditions = append(conditions, `i.storage_location_id IN (SELECT id FROM in_storage_location)`)
	}
	if f.FurnitureID != 0 {
		ctes = append(ctes, subtree("in_furniture", "furniture_item_id = ?"))
		cteArgs = append(cteArgs, f.FurnitureID)
		conditions = append(conditions, `i.storage_location_id IN (SELECT id FROM in_furniture)`)
	}
	if f.Unstored {
		conditions = append(conditions, `i.storage_location_id IS NULL`)
	}
	if f.BoxID != 0 {
		conditions = append(conditions, `im.box_id = ?`)
		args = append(args, f.BoxID)
	}
	if f.TargetRoomID != 0 {
		conditions = append(conditions, `COALESCE(im.target_room_id, b.target_room_id) = ?`)
		args = append(args, f.TargetRoomID)
	}
	// Without move details the current room counts as origin room, just as it would be prefilled on creation.
	if f.OriginRoomID != 0 {
		ctes = append(ctes, subtree("in_origin_room", "room_id = ?"))
		cteArgs = append(cteArgs, f.OriginRoomID)
		conditions = append(conditions, `((im.item_id IS NOT NULL AND im.origin_room_id = ?)
			OR (im.item_id IS NULL AND (i.room_id = ? OR i.storage_location_id IN (SELECT id FROM in_origin_room))))`)
		args = append(args, f.OriginRoomID, f.OriginRoomID)
	}
	// The effective target room counts. Items in the storage locations of furniture with an effective
	// target room go along with it, unless they are in a box.
	if f.WithoutTargetRoom {
		ctes = append(ctes, `old_rooms (id) AS (
			SELECT r.id FROM room r JOIN move m ON m.old_apartment_id = r.apartment_id
			WHERE m.id = COALESCE(?, (SELECT id FROM move WHERE active = 1)))`,
			subtree("in_old_apartment", "room_id IN (SELECT id FROM old_rooms)"),
			subtree("carried", `furniture_item_id IN (`+furnitureWithTargetRoom+`)`))
		cteArgs = append(cteArgs, nullID(f.MoveID), nullID(f.MoveID))
		conditions = append(conditions, `COALESCE(im.target_room_id, b.target_room_id) IS NULL
			AND (i.room_id IN (SELECT id FROM old_rooms) OR i.storage_location_id IN (SELECT id FROM in_old_apartment))
			AND NOT (im.box_id IS NULL AND COALESCE(i.storage_location_id IN (SELECT id FROM carried), FALSE))`)
	}

	// The path of every storage location below its room, e.g. "Arbeitszimmer › Schreibtisch › oben".
	ctes = append(ctes, `place_path (id, path) AS (
		SELECT l.id, r.name || ' › ' || l.name FROM storage_location l JOIN room r ON r.id = l.room_id
		UNION ALL
		SELECT l.id, pp.path || ' › ' || l.name FROM storage_location l JOIN place_path pp ON l.parent_id = pp.id)`)
	query := "WITH RECURSIVE " + strings.Join(ctes, ", ") + " "
	query += `
		SELECT i.id, i.name, i.disposition, p.id, p.file, COALESCE(l.name, r.name, ''), COALESCE(i.storage_location_id, 0), COALESCE(pp.path, r.name, ''),
			im.item_id IS NOT NULL, o.id, o.name, t.id, t.name, b.id, b.number, b.label, bt.id, bt.name
		FROM item i JOIN photo p ON p.item_id = i.id AND p.main_photo = 1
		LEFT JOIN storage_location l ON l.id = i.storage_location_id
		LEFT JOIN place_path pp ON pp.id = i.storage_location_id
		LEFT JOIN room r ON r.id = i.room_id
		LEFT JOIN item_move im ON im.item_id = i.id
			AND im.move_id = COALESCE(?, (SELECT id FROM move WHERE active = 1))
		LEFT JOIN room o ON o.id = im.origin_room_id
		LEFT JOIN room t ON t.id = im.target_room_id
		LEFT JOIN box b ON b.id = im.box_id
		LEFT JOIN room bt ON bt.id = b.target_room_id`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	// One more than asked for, to tell whether there are more.
	query += ` ORDER BY i.updated_at DESC, i.id DESC LIMIT ? OFFSET ?`
	args = append(append(append(cteArgs, nullID(f.MoveID)), args...), f.Limit+1, f.Offset)

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	entries := []ItemListEntry{}
	for rows.Next() {
		var (
			e                                               ItemListEntry
			disposition                                     sql.NullString
			hasDetails                                      bool
			originID, targetID, boxID, boxTargetID          sql.NullInt64
			originName, targetName, boxLabel, boxTargetName sql.NullString
			boxNumber                                       sql.NullInt64
		)
		if err := rows.Scan(&e.ID, &e.Name, &disposition, &e.MainPhotoID, &e.MainPhotoFile, &e.Place, &e.StorageLocationID, &e.PlacePath, &hasDetails,
			&originID, &originName, &targetID, &targetName, &boxID, &boxNumber, &boxLabel, &boxTargetID, &boxTargetName); err != nil {
			return nil, false, err
		}
		e.Disposition = dispositionFrom(disposition)
		if hasDetails {
			e.Move = &MoveDetails{OriginRoom: namedFrom(originID, originName), TargetRoom: namedFrom(targetID, targetName)}
			if boxID.Valid {
				e.Move.Box = &Box{ID: boxID.Int64, Number: int(boxNumber.Int64), Label: boxLabel.String,
					TargetRoom: namedFrom(boxTargetID, boxTargetName)}
			}
			e.Move.useBoxTargetRoom()
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(entries) > f.Limit {
		return entries[:f.Limit], true, nil
	}
	return entries, false, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Store) GetPhoto(ctx context.Context, id int64) (Photo, error) {
	p, err := scanPhoto(s.db.QueryRowContext(ctx, photoSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Photo{}, ErrNotFound
	}
	return p, err
}

// updateItem runs a change to an existing item in its own transaction.
func (s *Store) updateItem(ctx context.Context, id int64, now time.Time, change func(*sql.Tx) error) (Item, error) {
	var item Item
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var err error
		item, err = updateItemIn(ctx, tx, id, now, change)
		return err
	})
	return item, err
}

// updateItemIn runs the change, sets updated_at and returns the item afterwards.
func updateItemIn(ctx context.Context, tx *sql.Tx, id int64, now time.Time, change func(*sql.Tx) error) (Item, error) {
	res, err := tx.ExecContext(ctx, `UPDATE item SET updated_at = ? WHERE id = ?`, timeText(now), id)
	if err != nil {
		return Item{}, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return Item{}, errors.Join(err, ErrNotFound)
	}
	if err := change(tx); err != nil {
		return Item{}, err
	}
	return readItem(ctx, tx, id)
}

func (s *Store) EditItem(ctx context.Context, id int64, name, description string, disposition Disposition, now time.Time) (Item, error) {
	return s.updateItem(ctx, id, now, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE item SET name = ?, name_lower = ?, description = ?, disposition = ? WHERE id = ?`,
			name, strings.ToLower(name), description, dispositionValue(disposition), id)
		return err
	})
}

// DeleteItem deletes the item together with its photos and ownerships and returns the names of
// the photo files, which the caller removes afterwards. Storage locations of the item as furniture go away.
func (s *Store) DeleteItem(ctx context.Context, id int64, now time.Time) ([]string, error) {
	var files []string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		photos, err := readPhotos(ctx, tx, id)
		if err != nil {
			return err
		}
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
		res, err := tx.ExecContext(ctx, `DELETE FROM item WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			return errors.Join(err, ErrNotFound)
		}
		for _, p := range photos {
			files = append(files, p.File)
		}
		return nil
	})
	return files, err
}

func (s *Store) AddOwner(ctx context.Context, itemID, personID int64, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		return insertOwner(ctx, tx, itemID, personID)
	})
}

// RemoveOwner removes a person as owner; the last owner stays.
func (s *Store) RemoveOwner(ctx context.Context, itemID, personID int64, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		var others bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM ownership WHERE item_id = ? AND person_id <> ?)`,
			itemID, personID).Scan(&others)
		if err != nil {
			return err
		}
		if !others {
			return ErrLastOwner
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM ownership WHERE item_id = ? AND person_id = ?`, itemID, personID)
		return err
	})
}

func (s *Store) AddPhotos(ctx context.Context, itemID int64, photos []NewPhoto, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		return insertPhotos(ctx, tx, itemID, photos)
	})
}

// SetMainPhoto makes a photo of the item its main photo.
func (s *Store) SetMainPhoto(ctx context.Context, itemID, photoID int64, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		var belongs bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM photo WHERE id = ? AND item_id = ?)`, photoID, itemID).Scan(&belongs)
		if err != nil {
			return err
		}
		if !belongs {
			return ErrNotFound
		}
		// Reset the old main photo first, otherwise the "exactly one main photo" index kicks in.
		if _, err := tx.ExecContext(ctx, `UPDATE photo SET main_photo = 0 WHERE item_id = ?`, itemID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE photo SET main_photo = 1 WHERE id = ?`, photoID)
		return err
	})
}

// ReplacePhoto swaps the file of a photo for an already stored new one, e.g. after rotating or
// cropping. ID, order and main photo stay. It returns the old file name, which the caller
// removes afterwards, and the item.
func (s *Store) ReplacePhoto(ctx context.Context, photoID int64, n NewPhoto, now time.Time) (string, Item, error) {
	var (
		old  string
		item Item
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var itemID int64
		err := tx.QueryRowContext(ctx, `SELECT item_id, file FROM photo WHERE id = ?`, photoID).Scan(&itemID, &old)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		item, err = updateItemIn(ctx, tx, itemID, now, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `UPDATE photo SET file = ?, content_type = ? WHERE id = ?`, n.File, n.Type, photoID)
			return err
		})
		return err
	})
	if err != nil {
		return "", Item{}, err
	}
	return old, item, nil
}

// DeletePhoto deletes a photo unless it is the last one of its item. If it was the main photo,
// the next one in order becomes the main photo, after the last one the first. It returns the
// file name, which the caller removes afterwards, and the item.
func (s *Store) DeletePhoto(ctx context.Context, photoID int64, now time.Time) (string, Item, error) {
	var (
		file string
		item Item
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var itemID int64
		var sortOrder, count int
		err := tx.QueryRowContext(ctx, `
			SELECT p.item_id, p.file, p.sort_order, (SELECT COUNT(*) FROM photo a WHERE a.item_id = p.item_id)
			FROM photo p WHERE p.id = ?`, photoID).Scan(&itemID, &file, &sortOrder, &count)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastPhoto
		}
		item, err = updateItemIn(ctx, tx, itemID, now, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, `DELETE FROM photo WHERE id = ?`, photoID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `
				UPDATE photo SET main_photo = 1
				WHERE id = (
					SELECT id FROM photo WHERE item_id = ?
					ORDER BY CASE WHEN sort_order > ? THEN 0 ELSE 1 END, sort_order, id
					LIMIT 1)
				AND NOT EXISTS (SELECT 1 FROM photo WHERE item_id = ? AND main_photo = 1)`,
				itemID, sortOrder, itemID)
			return err
		})
		return err
	})
	if err != nil {
		return "", Item{}, err
	}
	return file, item, nil
}
