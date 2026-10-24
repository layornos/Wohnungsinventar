package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inventar/internal/store"
)

type moveJSON struct {
	ID           int64      `json:"id"`
	OldApartment namedJSON  `json:"oldApartment"`
	NewApartment namedJSON  `json:"newApartment"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt"`
	Boxes        []boxJSON  `json:"boxes"`
}

type boxJSON struct {
	ID         int64      `json:"id"`
	Number     int        `json:"number"`
	Label      string     `json:"label"`
	TargetRoom *namedJSON `json:"targetRoom"`
	Count      int        `json:"count"`
}

func toBoxJSON(b store.Box) boxJSON {
	return boxJSON{ID: b.ID, Number: b.Number, Label: b.Label, TargetRoom: toNamedPtr(b.TargetRoom), Count: b.Count}
}

func toMoveJSON(m store.Move) moveJSON {
	out := moveJSON{
		ID: m.ID, OldApartment: toNamedJSON(m.OldApartment), NewApartment: toNamedJSON(m.NewApartment),
		Active: m.Active, CreatedAt: m.CreatedAt, CompletedAt: m.CompletedAt, Boxes: []boxJSON{},
	}
	for _, b := range m.Boxes {
		out.Boxes = append(out.Boxes, toBoxJSON(b))
	}
	return out
}

// activeMove returns {"move": …} or {"move": null} if none is active.
func (s *server) activeMove(w http.ResponseWriter, r *http.Request) {
	m, err := s.Store.ActiveMove(r.Context())
	if errors.Is(err, store.ErrNoMove) {
		writeJSON(w, http.StatusOK, map[string]any{"move": nil})
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"move": toMoveJSON(m)})
}

func (s *server) createMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldApartmentID int64 `json:"oldApartmentId"`
		NewApartmentID int64 `json:"newApartmentId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	m, err := s.Store.CreateMove(r.Context(), req.OldApartmentID, req.NewApartmentID, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMoveJSON(m))
}

func (s *server) listMoves(w http.ResponseWriter, r *http.Request) {
	moves, err := s.Store.ListMoves(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]moveJSON, 0, len(moves))
	for _, m := range moves {
		out = append(out, toMoveJSON(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"moves": out})
}

func (s *server) getMove(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	m, err := s.Store.GetMove(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMoveJSON(m))
}

// readBoxRequest reads {"label", "targetRoomId"}. targetRoomID is nil if the field is missing,
// and points to 0 if it is null.
func readBoxRequest(w http.ResponseWriter, r *http.Request) (label string, targetRoomID *int64, ok bool) {
	var fields map[string]json.RawMessage
	if !readJSON(w, r, &fields) {
		return "", nil, false
	}
	if raw, present := fields["label"]; present {
		if err := json.Unmarshal(raw, &label); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige Bezeichnung")
			return "", nil, false
		}
	}
	if raw, present := fields["targetRoomId"]; present {
		var id *int64
		if err := json.Unmarshal(raw, &id); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige ID für targetRoomId")
			return "", nil, false
		}
		if id == nil {
			id = new(int64)
		}
		targetRoomID = id
	}
	return strings.TrimSpace(label), targetRoomID, true
}

func (s *server) createBox(w http.ResponseWriter, r *http.Request) {
	label, targetRoomID, ok := readBoxRequest(w, r)
	if !ok {
		return
	}
	var target int64
	if targetRoomID != nil {
		target = *targetRoomID
	}
	b, err := s.Store.CreateBox(r.Context(), label, target, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toBoxJSON(b))
}

// editBox sets label and target room of a box; without "targetRoomId" the target room stays.
func (s *server) editBox(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	label, targetRoomID, ok := readBoxRequest(w, r)
	if !ok {
		return
	}
	b, err := s.Store.EditBox(r.Context(), id, label, targetRoomID)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toBoxJSON(b))
}

// writeBoxChange answers a change to the contents of a box with {"count", "box"}.
func (s *server) writeBoxChange(w http.ResponseWriter, r *http.Request, count int, b store.Box, err error) {
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count, "box": toBoxJSON(b)})
}

// packStorageLocation packs everything in a storage location and below into the box ({"storageLocationId"}).
func (s *server) packStorageLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		StorageLocationID int64 `json:"storageLocationId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	count, b, err := s.Store.PackStorageLocation(r.Context(), id, req.StorageLocationID, s.Now())
	s.writeBoxChange(w, r, count, b, err)
}

// unpackBox places items of the box at a place of the new apartment: {"itemIds": […] or null
// for all, "storageLocationId" or "roomId"}.
func (s *server) unpackBox(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		ItemIDs           *[]int64 `json:"itemIds"`
		StorageLocationID int64    `json:"storageLocationId"`
		RoomID            int64    `json:"roomId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if (req.StorageLocationID == 0) == (req.RoomID == 0) {
		invalidPlace(w, "Bitte genau einen Lagerort oder Raum wählen")
		return
	}
	var itemIDs []int64
	if req.ItemIDs != nil {
		itemIDs = append([]int64{}, *req.ItemIDs...)
	}
	count, b, err := s.Store.UnpackBox(r.Context(), id, itemIDs, store.Place{StorageLocationID: req.StorageLocationID, RoomID: req.RoomID}, s.Now())
	s.writeBoxChange(w, r, count, b, err)
}

type moveDetailsJSON struct {
	OriginRoom *namedJSON `json:"originRoom"`
	// TargetRoom is the effective target room: the item's own one, otherwise the one of its box.
	TargetRoom        *namedJSON      `json:"targetRoom"`
	TargetRoomFromBox bool            `json:"targetRoomFromBox"`
	Box               *boxSummaryJSON `json:"box"`
}

// boxSummaryJSON is a box in the move details of an item, without contents.
type boxSummaryJSON struct {
	ID         int64      `json:"id"`
	Number     int        `json:"number"`
	Label      string     `json:"label"`
	TargetRoom *namedJSON `json:"targetRoom"`
}

func toMoveDetailsJSON(d *store.MoveDetails) *moveDetailsJSON {
	if d == nil {
		return nil
	}
	out := &moveDetailsJSON{OriginRoom: toNamedPtr(d.OriginRoom), TargetRoom: toNamedPtr(d.TargetRoom), TargetRoomFromBox: d.TargetRoomFromBox}
	if d.Box != nil {
		out.Box = &boxSummaryJSON{ID: d.Box.ID, Number: d.Box.Number, Label: d.Box.Label, TargetRoom: toNamedPtr(d.Box.TargetRoom)}
	}
	return out
}

func toNamedPtr(n *store.Named) *namedJSON {
	if n == nil {
		return nil
	}
	j := toNamedJSON(*n)
	return &j
}

// readMoveDetailsChange reads {"originRoomId", "targetRoomId", "boxId"}: missing fields stay as
// they are, null resets. extra names additional fields of the same body and where they are read into.
func readMoveDetailsChange(w http.ResponseWriter, r *http.Request, extra map[string]any) (store.MoveDetailsChange, bool) {
	var fields map[string]json.RawMessage
	if !readJSON(w, r, &fields) {
		return store.MoveDetailsChange{}, false
	}
	var c store.MoveDetailsChange
	for name, target := range map[string]**int64{"originRoomId": &c.OriginRoomID, "targetRoomId": &c.TargetRoomID, "boxId": &c.BoxID} {
		raw, present := fields[name]
		if !present {
			continue
		}
		var id *int64
		if err := json.Unmarshal(raw, &id); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige ID für "+name)
			return store.MoveDetailsChange{}, false
		}
		if id == nil {
			id = new(int64)
		}
		*target = id
	}
	for name, target := range extra {
		if raw, present := fields[name]; present {
			if err := json.Unmarshal(raw, target); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_request", "Ungültiger Wert für "+name)
				return store.MoveDetailsChange{}, false
			}
		}
	}
	return c, true
}

func (s *server) setMoveDetails(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	c, ok := readMoveDetailsChange(w, r, nil)
	if !ok {
		return
	}
	item, err := s.Store.SetMoveDetails(r.Context(), id, c, s.Now())
	s.writeItem(w, r, item, err)
}

// setMoveDetailsBulk sets box, target room or origin room for several items ({"itemIds": […], …}).
func (s *server) setMoveDetailsBulk(w http.ResponseWriter, r *http.Request) {
	var itemIDs []int64
	c, ok := readMoveDetailsChange(w, r, map[string]any{"itemIds": &itemIDs})
	if !ok {
		return
	}
	if len(itemIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Bitte mindestens ein Item wählen")
		return
	}
	if err := s.Store.SetMoveDetailsBulk(r.Context(), itemIDs, c, s.Now()); err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// packingListItemJSON is an item in the packing list, with the path of where it was before packing.
type packingListItemJSON struct {
	itemListJSON
	PreviousPlace string `json:"previousPlace"`
}

// writePackingList returns the box with all items in it including their move details.
func (s *server) writePackingList(w http.ResponseWriter, r *http.Request, b store.Box, err error) {
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	entries, _, err := s.Store.ListItems(r.Context(), store.ItemFilter{MoveID: b.MoveID, BoxID: b.ID, Limit: maxComplete})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	items := make([]packingListItemJSON, 0, len(entries))
	for i, item := range toItemListJSON(entries) {
		items = append(items, packingListItemJSON{itemListJSON: item, PreviousPlace: entries[i].PlacePath})
	}
	writeJSON(w, http.StatusOK, map[string]any{"box": toBoxJSON(b), "items": items})
}

func (s *server) getBox(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	b, err := s.Store.GetBox(r.Context(), id)
	s.writePackingList(w, r, b, err)
}

func (s *server) boxByNumber(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Diese Box gibt es nicht")
		return
	}
	b, err := s.Store.BoxByNumber(r.Context(), number)
	s.writePackingList(w, r, b, err)
}

// targetRooms returns, for every room of the new apartment, the boxes and items that go there.
func (s *server) targetRooms(w http.ResponseWriter, r *http.Request) {
	overview, err := s.Store.TargetRoomOverview(r.Context())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	type targetRoomJSON struct {
		Room  namedJSON      `json:"room"`
		Boxes []boxJSON      `json:"boxes"`
		Items []itemListJSON `json:"items"`
	}
	out := []targetRoomJSON{}
	for _, t := range overview {
		boxes := []boxJSON{}
		for _, b := range t.Boxes {
			boxes = append(boxes, toBoxJSON(b))
		}
		out = append(out, targetRoomJSON{Room: toNamedJSON(t.Room), Boxes: boxes, Items: toItemListJSON(t.Items)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"targetRooms": out})
}

// completeMove completes the active move. If items of the old apartment have no target room,
// it requires {"confirmed": true} and otherwise answers with these items.
func (s *server) completeMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Confirmed bool `json:"confirmed"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	m, without, err := s.Store.CompleteMove(r.Context(), req.Confirmed, s.Now())
	if errors.Is(err, store.ErrItemsWithoutTargetRoom) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":    "items_without_target_room",
			"message": "Diese Items der alten Wohnung haben noch keinen Zielraum und blieben, wo sie sind",
			"items":   toItemListJSON(without),
		})
		return
	}
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMoveJSON(m))
}
