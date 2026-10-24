package server

import (
	"net/http"
	"strconv"
	"strings"

	"inventar/internal/store"
)

type pathJSON struct {
	Apartment        namedJSON   `json:"apartment"`
	Room             namedJSON   `json:"room"`
	StorageLocations []namedJSON `json:"storageLocations"`
}

func toPathJSON(p store.Path) pathJSON {
	return pathJSON{Apartment: toNamedJSON(p.Apartment), Room: toNamedJSON(p.Room), StorageLocations: toNamedListJSON(p.StorageLocations)}
}

type storageLocationJSON struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Path      pathJSON    `json:"path"`
	Furniture *namedJSON  `json:"furniture"`
	Children  []namedJSON `json:"children"`
	Subtree   []int64     `json:"subtree"`
}

func toStorageLocationJSON(l store.StorageLocationDetail) storageLocationJSON {
	out := storageLocationJSON{ID: l.ID, Name: l.Name, Path: toPathJSON(l.Path), Children: toNamedListJSON(l.Children), Subtree: l.Subtree}
	if l.Furniture != nil {
		f := toNamedJSON(*l.Furniture)
		out.Furniture = &f
	}
	return out
}

func invalidPlace(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, "invalid_place", message)
}

func parentMissing(w http.ResponseWriter) {
	invalidPlace(w, "Ein Lagerort liegt entweder in einem Raum oder in einem anderen Lagerort")
}

// placeFrom makes a placement from optional IDs; both at once are not allowed.
func placeFrom(w http.ResponseWriter, storageLocationID, roomID int64) (store.Place, bool) {
	if storageLocationID != 0 && roomID != 0 {
		invalidPlace(w, "Entweder ein Lagerort oder ein Raum, nicht beides")
		return store.Place{}, false
	}
	return store.Place{StorageLocationID: storageLocationID, RoomID: roomID}, true
}

// placeFromForm reads the placement from the form fields "storageLocation" and "room".
func placeFromForm(w http.ResponseWriter, f form) (store.Place, bool) {
	var ids [2]int64
	for i, field := range []string{"storageLocation", "room"} {
		if value := f.field(field); value != "" {
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "unknown_place", "Diesen Raum oder Lagerort gibt es nicht")
				return store.Place{}, false
			}
			ids[i] = id
		}
	}
	return placeFrom(w, ids[0], ids[1])
}

func (s *server) setPlacement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		StorageLocationID int64 `json:"storageLocationId"`
		RoomID            int64 `json:"roomId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	p, ok := placeFrom(w, req.StorageLocationID, req.RoomID)
	if !ok {
		return
	}
	item, err := s.Store.SetPlacement(r.Context(), id, p, s.Now())
	s.writeItem(w, r, item, err)
}

func (s *server) createStorageLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		RoomID   int64  `json:"roomId"`
		ParentID int64  `json:"parentId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	n := store.NewStorageLocation{Name: strings.TrimSpace(req.Name), RoomID: req.RoomID, ParentID: req.ParentID}
	if n.Name == "" {
		nameMissing(w)
		return
	}
	if (n.RoomID == 0) == (n.ParentID == 0) {
		parentMissing(w)
		return
	}
	l, err := s.Store.CreateStorageLocation(r.Context(), n, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStorageLocationJSON(l))
}

func (s *server) getStorageLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	l, err := s.Store.GetStorageLocation(r.Context(), id)
	s.writeStorageLocation(w, r, l, err)
}

func (s *server) writeStorageLocation(w http.ResponseWriter, r *http.Request, l store.StorageLocationDetail, err error) {
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStorageLocationJSON(l))
}

// setFurniture assigns the storage location to an item; {"itemId": null} removes the assignment.
func (s *server) setFurniture(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		ItemID int64 `json:"itemId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	l, err := s.Store.SetFurniture(r.Context(), id, req.ItemID)
	s.writeStorageLocation(w, r, l, err)
}

func (s *server) moveStorageLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		RoomID   int64 `json:"roomId"`
		ParentID int64 `json:"parentId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if (req.RoomID == 0) == (req.ParentID == 0) {
		parentMissing(w)
		return
	}
	l, err := s.Store.MoveStorageLocation(r.Context(), id, req.RoomID, req.ParentID)
	s.writeStorageLocation(w, r, l, err)
}

func (s *server) deleteStorageLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Store.DeleteStorageLocation(r.Context(), id, s.Now()); err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type placeJSON struct {
	Kind        store.PlaceKind `json:"kind"`
	ID          int64           `json:"id"`
	ApartmentID int64           `json:"apartmentId"`
	Path        string          `json:"path"`
}

func (s *server) listPlaces(w http.ResponseWriter, r *http.Request) {
	places, err := s.Store.ListPlaces(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]placeJSON, 0, len(places))
	for _, p := range places {
		out = append(out, placeJSON{Kind: p.Kind, ID: p.ID, ApartmentID: p.ApartmentID, Path: p.Path})
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": out})
}
