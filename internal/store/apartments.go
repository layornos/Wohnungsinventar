package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrApartmentNotEmpty     = errors.New("apartment still has rooms")
	ErrRoomNotEmpty          = errors.New("room still has storage locations or items")
	ErrCycle                 = errors.New("storage location would be inside itself")
	ErrFurnitureInsideItself = errors.New("furniture would be inside one of its own storage locations")
	ErrUnknownPlace          = errors.New("unknown room or storage location")
)

// Named is something with ID and name, e.g. an apartment, a room or a storage location in a path.
type Named struct {
	ID   int64
	Name string
}

type Apartment struct {
	ID    int64
	Name  string
	Rooms []Named
}

func (s *Store) ListApartments(ctx context.Context) ([]Apartment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name, r.id, r.name
		FROM apartment a LEFT JOIN room r ON r.apartment_id = a.id
		ORDER BY a.name, a.id, r.name, r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	apartments := []Apartment{}
	for rows.Next() {
		var (
			a        Apartment
			roomID   sql.NullInt64
			roomName sql.NullString
		)
		if err := rows.Scan(&a.ID, &a.Name, &roomID, &roomName); err != nil {
			return nil, err
		}
		if n := len(apartments); n == 0 || apartments[n-1].ID != a.ID {
			a.Rooms = []Named{}
			apartments = append(apartments, a)
		}
		if roomID.Valid {
			last := &apartments[len(apartments)-1]
			last.Rooms = append(last.Rooms, Named{ID: roomID.Int64, Name: roomName.String})
		}
	}
	return apartments, rows.Err()
}

func (s *Store) CreateApartment(ctx context.Context, name string, now time.Time) (Named, error) {
	o := Named{Name: name}
	err := s.db.QueryRowContext(ctx, `INSERT INTO apartment (name, created_at) VALUES (?, ?) RETURNING id`, name, timeText(now)).Scan(&o.ID)
	return o, err
}

func (s *Store) RenameApartment(ctx context.Context, id int64, name string) (Named, error) {
	return rename(ctx, s.db, "apartment", id, name)
}

// rename sets the name of a row in one of the tables apartment, room or storage_location.
func rename(ctx context.Context, q querier, table string, id int64, name string) (Named, error) {
	res, err := q.ExecContext(ctx, `UPDATE `+table+` SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return Named{}, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return Named{}, errors.Join(err, ErrNotFound)
	}
	return Named{ID: id, Name: name}, nil
}

func (s *Store) DeleteApartment(ctx context.Context, id int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var exists, rooms, move bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM apartment WHERE id = ?), EXISTS (SELECT 1 FROM room WHERE apartment_id = ?),
				EXISTS (SELECT 1 FROM move WHERE old_apartment_id = ? OR new_apartment_id = ?)`,
			id, id, id, id).Scan(&exists, &rooms, &move)
		switch {
		case err != nil:
			return err
		case !exists:
			return ErrNotFound
		case move:
			return ErrApartmentInMove
		case rooms:
			return ErrApartmentNotEmpty
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM apartment WHERE id = ?`, id)
		return err
	})
}

func (s *Store) CreateRoom(ctx context.Context, apartmentID int64, name string, now time.Time) (Named, error) {
	o := Named{Name: name}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO room (apartment_id, name, created_at)
		SELECT id, ?, ? FROM apartment WHERE id = ? RETURNING id`, name, timeText(now), apartmentID).Scan(&o.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return Named{}, ErrNotFound
	}
	return o, err
}

func (s *Store) RenameRoom(ctx context.Context, id int64, name string) (Named, error) {
	return rename(ctx, s.db, "room", id, name)
}

// DeleteRoom deletes an empty room: without storage locations and without items placed directly in it.
func (s *Store) DeleteRoom(ctx context.Context, id int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var exists, occupied, inMove bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM room WHERE id = ?),
				EXISTS (SELECT 1 FROM storage_location WHERE room_id = ?) OR EXISTS (SELECT 1 FROM item WHERE room_id = ?),
				EXISTS (SELECT 1 FROM item_move WHERE origin_room_id = ? OR target_room_id = ?)
					OR EXISTS (SELECT 1 FROM box WHERE target_room_id = ?)`,
			id, id, id, id, id, id).Scan(&exists, &occupied, &inMove)
		switch {
		case err != nil:
			return err
		case !exists:
			return ErrNotFound
		case occupied:
			return ErrRoomNotEmpty
		case inMove:
			return ErrRoomInMove
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM room WHERE id = ?`, id)
		return err
	})
}

// Room is a room with its apartment and the tree of its storage locations.
type Room struct {
	Named
	Apartment        Named
	StorageLocations []StorageLocation
}

// StorageLocation is a node in the storage location tree.
type StorageLocation struct {
	Named
	Furniture *Named
	Children  []StorageLocation
}

func (s *Store) GetRoom(ctx context.Context, id int64) (Room, error) {
	var r Room
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.name, a.id, a.name FROM room r JOIN apartment a ON a.id = r.apartment_id WHERE r.id = ?`, id).
		Scan(&r.ID, &r.Name, &r.Apartment.ID, &r.Apartment.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, ErrNotFound
	}
	if err != nil {
		return Room{}, err
	}
	r.StorageLocations, err = storageLocationTree(ctx, s.db, id)
	return r, err
}

// storageLocationTree reads all storage locations of a room including nested ones as a tree.
func storageLocationTree(ctx context.Context, q querier, roomID int64) ([]StorageLocation, error) {
	rows, err := q.QueryContext(ctx, `
		WITH RECURSIVE tree (id, parent_id) AS (
			SELECT id, CAST(NULL AS INTEGER) FROM storage_location WHERE room_id = ?
			UNION ALL
			SELECT l.id, l.parent_id FROM storage_location l JOIN tree t ON l.parent_id = t.id
		)
		SELECT l.id, l.name, t.parent_id, i.id, i.name
		FROM tree t JOIN storage_location l ON l.id = t.id LEFT JOIN item i ON i.id = l.furniture_item_id
		ORDER BY l.name, l.id`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type node struct {
		l      StorageLocation
		parent sql.NullInt64
	}
	var all []node
	for rows.Next() {
		var (
			n             node
			furnitureID   sql.NullInt64
			furnitureName sql.NullString
		)
		if err := rows.Scan(&n.l.ID, &n.l.Name, &n.parent, &furnitureID, &furnitureName); err != nil {
			return nil, err
		}
		if furnitureID.Valid {
			n.l.Furniture = &Named{ID: furnitureID.Int64, Name: furnitureName.String}
		}
		all = append(all, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	children := map[int64][]int64{}
	var roots []int64
	byID := map[int64]StorageLocation{}
	for _, n := range all {
		byID[n.l.ID] = n.l
		if n.parent.Valid {
			children[n.parent.Int64] = append(children[n.parent.Int64], n.l.ID)
		} else {
			roots = append(roots, n.l.ID)
		}
	}
	var build func(ids []int64) []StorageLocation
	build = func(ids []int64) []StorageLocation {
		out := []StorageLocation{}
		for _, id := range ids {
			l := byID[id]
			l.Children = build(children[id])
			out = append(out, l)
		}
		return out
	}
	return build(roots), nil
}

// Path describes where something is: apartment › room › storage locations from outside in.
type Path struct {
	Apartment        Named
	Room             Named
	StorageLocations []Named
}

// Place is the placement of an item: a storage location, a room without storage location or (both 0) nothing.
type Place struct {
	StorageLocationID int64
	RoomID            int64
}

// roomPath returns the path of a room.
func roomPath(ctx context.Context, q querier, roomID int64) (Path, error) {
	var p Path
	err := q.QueryRowContext(ctx, `
		SELECT a.id, a.name, r.id, r.name FROM room r JOIN apartment a ON a.id = r.apartment_id WHERE r.id = ?`, roomID).
		Scan(&p.Apartment.ID, &p.Apartment.Name, &p.Room.ID, &p.Room.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Path{}, ErrNotFound
	}
	p.StorageLocations = []Named{}
	return p, err
}

// storageLocationPath returns the path up to and including the storage location, along the chain of its parents.
func storageLocationPath(ctx context.Context, q querier, storageLocationID int64) (Path, error) {
	rows, err := q.QueryContext(ctx, `
		WITH RECURSIVE chain (id, name, parent_id, room_id, depth) AS (
			SELECT id, name, parent_id, room_id, 0 FROM storage_location WHERE id = ?
			UNION ALL
			SELECT l.id, l.name, l.parent_id, l.room_id, c.depth + 1 FROM storage_location l JOIN chain c ON l.id = c.parent_id
		)
		SELECT id, name, room_id FROM chain ORDER BY depth DESC`, storageLocationID)
	if err != nil {
		return Path{}, err
	}
	var (
		locations []Named
		roomID    sql.NullInt64
	)
	for rows.Next() {
		var (
			o Named
			r sql.NullInt64
		)
		if err := rows.Scan(&o.ID, &o.Name, &r); err != nil {
			rows.Close()
			return Path{}, err
		}
		if r.Valid {
			roomID = r
		}
		locations = append(locations, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Path{}, err
	}
	if len(locations) == 0 {
		return Path{}, ErrNotFound
	}
	p, err := roomPath(ctx, q, roomID.Int64)
	p.StorageLocations = locations
	return p, err
}

// StorageLocationDetail is a storage location with path, furniture and direct children.
type StorageLocationDetail struct {
	Named
	Path      Path
	Furniture *Named
	Children  []Named
	// Subtree holds the IDs of the storage location and all nested ones; it cannot be moved there.
	Subtree []int64
}

type NewStorageLocation struct {
	Name string
	// Exactly one of the two: the room or the parent storage location.
	RoomID   int64
	ParentID int64
}

func (s *Store) CreateStorageLocation(ctx context.Context, n NewStorageLocation, now time.Time) (StorageLocationDetail, error) {
	var l StorageLocationDetail
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := checkPlace(ctx, tx, n.RoomID, n.ParentID); err != nil {
			return err
		}
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO storage_location (name, room_id, parent_id, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
			n.Name, nullID(n.RoomID), nullID(n.ParentID), timeText(now)).Scan(&id)
		if err != nil {
			return err
		}
		l, err = readStorageLocation(ctx, tx, id)
		return err
	})
	return l, err
}

// checkPlace checks that the room or storage location something is to go into exists.
func checkPlace(ctx context.Context, q querier, roomID, storageLocationID int64) error {
	table, id := "room", roomID
	if storageLocationID != 0 {
		table, id = "storage_location", storageLocationID
	}
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = ?)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrUnknownPlace
	}
	return nil
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *Store) GetStorageLocation(ctx context.Context, id int64) (StorageLocationDetail, error) {
	return readStorageLocation(ctx, s.db, id)
}

func readStorageLocation(ctx context.Context, q querier, id int64) (StorageLocationDetail, error) {
	var (
		l             StorageLocationDetail
		furnitureID   sql.NullInt64
		furnitureName sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT l.id, l.name, i.id, i.name FROM storage_location l LEFT JOIN item i ON i.id = l.furniture_item_id WHERE l.id = ?`, id).
		Scan(&l.ID, &l.Name, &furnitureID, &furnitureName)
	if errors.Is(err, sql.ErrNoRows) {
		return StorageLocationDetail{}, ErrNotFound
	}
	if err != nil {
		return StorageLocationDetail{}, err
	}
	if furnitureID.Valid {
		l.Furniture = &Named{ID: furnitureID.Int64, Name: furnitureName.String}
	}
	if l.Path, err = storageLocationPath(ctx, q, id); err != nil {
		return StorageLocationDetail{}, err
	}
	if l.Children, err = readNamed(q.QueryContext(ctx, `SELECT id, name FROM storage_location WHERE parent_id = ? ORDER BY name, id`, id)); err != nil {
		return StorageLocationDetail{}, err
	}
	l.Subtree, err = readIDs(q.QueryContext(ctx, withSubtree+`SELECT id FROM subtree`, id))
	return l, err
}

func readIDs(rows *sql.Rows, err error) ([]int64, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func readNamed(rows *sql.Rows, err error) ([]Named, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Named{}
	for rows.Next() {
		var o Named
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) RenameStorageLocation(ctx context.Context, id int64, name string) (Named, error) {
	return rename(ctx, s.db, "storage_location", id, name)
}

// SetFurniture assigns the storage location to an item (the piece of furniture); itemID 0 removes the assignment.
func (s *Store) SetFurniture(ctx context.Context, storageLocationID, itemID int64) (StorageLocationDetail, error) {
	var l StorageLocationDetail
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if itemID != 0 {
			// An archived item is no furniture any more.
			var handedOverOn sql.NullString
			err := tx.QueryRowContext(ctx, `SELECT handed_over_on FROM item WHERE id = ?`, itemID).Scan(&handedOverOn)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			if handedOverOn.Valid {
				return ErrArchived
			}
		}
		res, err := tx.ExecContext(ctx, `UPDATE storage_location SET furniture_item_id = ? WHERE id = ?`, nullID(itemID), storageLocationID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			return errors.Join(err, ErrNotFound)
		}
		if err := checkFurniture(ctx, tx); err != nil {
			return err
		}
		l, err = readStorageLocation(ctx, tx, storageLocationID)
		return err
	})
	return l, err
}

// checkFurniture makes sure no piece of furniture is placed in one of its own storage locations
// or their nested ones.
func checkFurniture(ctx context.Context, q querier) error {
	var violated bool
	err := q.QueryRowContext(ctx, `
		WITH RECURSIVE own (furniture_item_id, id) AS (
			SELECT furniture_item_id, id FROM storage_location WHERE furniture_item_id IS NOT NULL
			UNION ALL
			SELECT o.furniture_item_id, l.id FROM storage_location l JOIN own o ON l.parent_id = o.id
		)
		SELECT EXISTS (SELECT 1 FROM own o JOIN item i ON i.id = o.furniture_item_id AND i.storage_location_id = o.id)`).Scan(&violated)
	if err != nil {
		return err
	}
	if violated {
		return ErrFurnitureInsideItself
	}
	return nil
}

// setPlace sets the placement of an item; at most one of storage location and room stays set.
func setPlace(ctx context.Context, q querier, itemID int64, p Place) error {
	if p.StorageLocationID != 0 || p.RoomID != 0 {
		if err := checkPlace(ctx, q, p.RoomID, p.StorageLocationID); err != nil {
			return err
		}
	}
	if p.StorageLocationID != 0 {
		p.RoomID = 0
	}
	if _, err := q.ExecContext(ctx, `UPDATE item SET storage_location_id = ?, room_id = ? WHERE id = ?`,
		nullID(p.StorageLocationID), nullID(p.RoomID), itemID); err != nil {
		return err
	}
	return checkFurniture(ctx, q)
}

func (s *Store) SetPlacement(ctx context.Context, itemID int64, p Place, now time.Time) (Item, error) {
	return s.updateItem(ctx, itemID, now, func(tx *sql.Tx) error {
		return setPlace(ctx, tx, itemID, p)
	})
}

// withSubtree is the recursive expression with the storage location for the placeholder and all its nested ones.
var withSubtree = "WITH RECURSIVE " + subtree("subtree", "id = ?") + " "

// MoveStorageLocation puts the storage location with its contents into a room or under another storage location.
func (s *Store) MoveStorageLocation(ctx context.Context, id, roomID, parentID int64) (StorageLocationDetail, error) {
	var l StorageLocationDetail
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM storage_location WHERE id = ?)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if err := checkPlace(ctx, tx, roomID, parentID); err != nil {
			return err
		}
		if parentID != 0 {
			var cycle bool
			if err := tx.QueryRowContext(ctx, withSubtree+`SELECT EXISTS (SELECT 1 FROM subtree WHERE id = ?)`, id, parentID).Scan(&cycle); err != nil {
				return err
			}
			if cycle {
				return ErrCycle
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE storage_location SET room_id = ?, parent_id = ? WHERE id = ?`, nullID(roomID), nullID(parentID), id); err != nil {
			return err
		}
		if err := checkFurniture(ctx, tx); err != nil {
			return err
		}
		var err error
		l, err = readStorageLocation(ctx, tx, id)
		return err
	})
	return l, err
}

// DeleteStorageLocation deletes the storage location including nested ones; their items are
// then unstored in the room the storage location was in.
func (s *Store) DeleteStorageLocation(ctx context.Context, id int64, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		return removeStorageLocation(ctx, tx, id, now)
	})
}

func removeStorageLocation(ctx context.Context, q querier, id int64, now time.Time) error {
	path, err := storageLocationPath(ctx, q, id)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, withSubtree+`
		UPDATE item SET storage_location_id = NULL, room_id = ?, updated_at = ?
		WHERE storage_location_id IN (SELECT id FROM subtree)`, id, path.Room.ID, timeText(now)); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, withSubtree+`DELETE FROM storage_location WHERE id IN (SELECT id FROM subtree)`, id)
	return err
}

// PlaceKind tells rooms and storage locations apart as places.
type PlaceKind string

const (
	PlaceRoom            PlaceKind = "room"
	PlaceStorageLocation PlaceKind = "storageLocation"
)

// PlaceEntry is a room or storage location with its path as text, e.g. for a selection list.
type PlaceEntry struct {
	Kind        PlaceKind
	ID          int64
	ApartmentID int64
	Path        string
}

// ListPlaces returns all rooms and storage locations, sorted by path.
func (s *Store) ListPlaces(ctx context.Context) ([]PlaceEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE stored (id, apartment_id, path) AS (
			SELECT l.id, a.id, a.name || ' › ' || r.name || ' › ' || l.name
			FROM storage_location l JOIN room r ON r.id = l.room_id JOIN apartment a ON a.id = r.apartment_id
			UNION ALL
			SELECT l.id, s.apartment_id, s.path || ' › ' || l.name FROM storage_location l JOIN stored s ON l.parent_id = s.id
		)
		SELECT 'room', r.id, a.id, a.name || ' › ' || r.name FROM room r JOIN apartment a ON a.id = r.apartment_id
		UNION ALL
		SELECT 'storageLocation', id, apartment_id, path FROM stored
		ORDER BY 4, 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	places := []PlaceEntry{}
	for rows.Next() {
		var p PlaceEntry
		if err := rows.Scan(&p.Kind, &p.ID, &p.ApartmentID, &p.Path); err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, rows.Err()
}
