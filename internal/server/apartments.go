package server

import (
	"context"
	"net/http"

	"inventar/internal/store"
)

type namedJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func toNamedJSON(n store.Named) namedJSON { return namedJSON{ID: n.ID, Name: n.Name} }

func toNamedListJSON(list []store.Named) []namedJSON {
	out := make([]namedJSON, 0, len(list))
	for _, n := range list {
		out = append(out, toNamedJSON(n))
	}
	return out
}

type apartmentJSON struct {
	ID    int64       `json:"id"`
	Name  string      `json:"name"`
	Rooms []namedJSON `json:"rooms"`
}

type storageLocationNodeJSON struct {
	ID        int64                     `json:"id"`
	Name      string                    `json:"name"`
	Furniture *namedJSON                `json:"furniture"`
	Children  []storageLocationNodeJSON `json:"children"`
}

func toTreeJSON(locations []store.StorageLocation) []storageLocationNodeJSON {
	out := make([]storageLocationNodeJSON, 0, len(locations))
	for _, l := range locations {
		n := storageLocationNodeJSON{ID: l.ID, Name: l.Name, Children: toTreeJSON(l.Children)}
		if l.Furniture != nil {
			f := toNamedJSON(*l.Furniture)
			n.Furniture = &f
		}
		out = append(out, n)
	}
	return out
}

func (s *server) listApartments(w http.ResponseWriter, r *http.Request) {
	apartments, err := s.Store.ListApartments(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]apartmentJSON, 0, len(apartments))
	for _, a := range apartments {
		out = append(out, apartmentJSON{ID: a.ID, Name: a.Name, Rooms: toNamedListJSON(a.Rooms)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"apartments": out})
}

func (s *server) createApartment(w http.ResponseWriter, r *http.Request) {
	name, ok := readName(w, r)
	if !ok {
		return
	}
	n, err := s.Store.CreateApartment(r.Context(), name, s.Now())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toNamedJSON(n))
}

func (s *server) createRoom(w http.ResponseWriter, r *http.Request) {
	apartmentID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	name, ok := readName(w, r)
	if !ok {
		return
	}
	n, err := s.Store.CreateRoom(r.Context(), apartmentID, name, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toNamedJSON(n))
}

// rename renames the apartment, room or storage location with the ID in the path.
func (s *server) rename(change func(ctx context.Context, id int64, name string) (store.Named, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		name, ok := readName(w, r)
		if !ok {
			return
		}
		n, err := change(r.Context(), id, name)
		if err != nil {
			s.storeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toNamedJSON(n))
	}
}

// delete deletes the element with the ID in the path.
func (s *server) delete(del func(ctx context.Context, id int64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		if err := del(r.Context(), id); err != nil {
			s.storeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) getRoom(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	room, err := s.Store.GetRoom(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": room.ID, "name": room.Name, "apartment": toNamedJSON(room.Apartment), "storageLocations": toTreeJSON(room.StorageLocations),
	})
}
