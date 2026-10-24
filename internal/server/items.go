package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inventar/internal/photo"
	"inventar/internal/store"
)

const (
	maxPhotosPerRequest = 10
	maxFormParts        = 50
	maxFieldLength      = 10_000
	defaultPerPage      = 50
	maxPerPage          = 100
	// maxComplete limits lists that are meant to be complete, e.g. removal preview and packing list.
	maxComplete = 10_000
)

type itemJSON struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Disposition *store.Disposition `json:"disposition"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	Owners      []personJSON       `json:"owners"`
	Photos      []photoJSON        `json:"photos"`
	// Placement is null as long as the item is not placed.
	Placement                 *pathJSON   `json:"placement"`
	FurnitureStorageLocations []namedJSON `json:"furnitureStorageLocations"`
	// Move holds the details in the active move, otherwise null.
	Move *moveDetailsJSON `json:"move"`
}

type photoJSON struct {
	ID           int64  `json:"id"`
	MainPhoto    bool   `json:"mainPhoto"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// dispositionJSON represents Undecided as null.
func dispositionJSON(d store.Disposition) *store.Disposition {
	if d == store.Undecided {
		return nil
	}
	return &d
}

// photoURL contains the random file name besides the ID: photos stay in the browser cache
// for good, so the address has to change as soon as the file changes.
func photoURL(id int64, file string) string {
	return fmt.Sprintf("/api/photos/%d/%s", id, file)
}

func thumbnailURL(photoID int64, file string) string { return photoURL(photoID, file) + "/thumbnail" }

func toItemJSON(item store.Item) itemJSON {
	out := itemJSON{
		ID:                        item.ID,
		Name:                      item.Name,
		Description:               item.Description,
		Disposition:               dispositionJSON(item.Disposition),
		CreatedAt:                 item.CreatedAt,
		UpdatedAt:                 item.UpdatedAt,
		Owners:                    toPersonsJSON(item.Owners),
		Photos:                    []photoJSON{},
		FurnitureStorageLocations: toNamedListJSON(item.FurnitureStorageLocations),
		Move:                      toMoveDetailsJSON(item.Move),
	}
	if item.Placement != nil {
		p := toPathJSON(*item.Placement)
		out.Placement = &p
	}
	for _, p := range item.Photos {
		out.Photos = append(out.Photos, photoJSON{ID: p.ID, MainPhoto: p.MainPhoto, URL: photoURL(p.ID, p.File), ThumbnailURL: thumbnailURL(p.ID, p.File)})
	}
	return out
}

// form is a parsed multipart request; the photos are already stored.
type form struct {
	fields map[string][]string
	photos []photo.File
}

func (f form) field(name string) string {
	if v := f.fields[name]; len(v) > 0 {
		return strings.TrimSpace(v[0])
	}
	return ""
}

func (f form) newPhotos() []store.NewPhoto {
	out := make([]store.NewPhoto, 0, len(f.photos))
	for _, p := range f.photos {
		out = append(out, store.NewPhoto{File: p.Name, Type: p.Type})
	}
	return out
}

// readForm reads a multipart request and stores the photos (field "photo") right away.
// On errors it answers itself and discards photos already stored.
func (s *server) readForm(w http.ResponseWriter, r *http.Request) (form, bool) {
	f := form{fields: map[string][]string{}}
	abort := func(status int, code, message string) (form, bool) {
		s.discardPhotos(f.photos)
		writeError(w, status, code, message)
		return form{}, false
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return abort(http.StatusBadRequest, "invalid_request", "Formulardaten erwartet")
	}
	for parts := 0; ; parts++ {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return f, true
		}
		if err != nil || parts >= maxFormParts {
			return abort(http.StatusBadRequest, "invalid_request", "Ungültige Formulardaten")
		}
		if part.FormName() == "photo" {
			if len(f.photos) >= maxPhotosPerRequest {
				return abort(http.StatusBadRequest, "too_many_photos", fmt.Sprintf("Höchstens %d Fotos auf einmal", maxPhotosPerRequest))
			}
			p, err := s.photos.Save(part)
			switch {
			case errors.Is(err, photo.ErrWrongType):
				return abort(http.StatusBadRequest, "wrong_file_type", "Nur JPEG-, PNG- und WebP-Fotos")
			case errors.Is(err, photo.ErrTooLarge):
				return abort(http.StatusRequestEntityTooLarge, "too_large",
					fmt.Sprintf("Fotos dürfen höchstens %d MB groß sein", s.photos.MaxSize>>20))
			case err != nil:
				s.discardPhotos(f.photos)
				s.internalError(w, r, err)
				return form{}, false
			}
			f.photos = append(f.photos, p)
			continue
		}
		value, err := io.ReadAll(io.LimitReader(part, maxFieldLength+1))
		if err != nil || len(value) > maxFieldLength {
			return abort(http.StatusBadRequest, "invalid_request", "Formularfeld zu lang")
		}
		f.fields[part.FormName()] = append(f.fields[part.FormName()], string(value))
	}
}

// discardPhotos removes stored photos that were not assigned to any item.
func (s *server) discardPhotos(files []photo.File) {
	for _, f := range files {
		s.deleteFile(f.Name)
	}
}

func (s *server) deleteFile(name string) {
	if err := s.photos.Delete(name); err != nil {
		s.Logger.Error("delete photo file", "file", name, "error", err)
	}
}

func readDisposition(w http.ResponseWriter, value string) (store.Disposition, bool) {
	d := store.Disposition(value)
	if !d.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_disposition", "Unbekannter Verbleib")
		return "", false
	}
	return d, true
}

// Messages shown in the UI.
const (
	msgOwnerMissing  = "Jedes Item braucht mindestens einen Besitzer"
	msgPhotoMissing  = "Jedes Item braucht mindestens ein Foto"
	msgUnknownPerson = "Diese Person gibt es nicht"
	msgUnknownBox    = "Diese Box gibt es im laufenden Umzug nicht"
)

func nameMissing(w http.ResponseWriter) {
	writeError(w, http.StatusBadRequest, "name_missing", "Bitte einen Namen angeben")
}

// readNewItem checks the form fields of a new item and answers errors itself.
func readNewItem(w http.ResponseWriter, f form) (store.NewItem, bool) {
	n := store.NewItem{Name: f.field("name"), Description: f.field("description"), Photos: f.newPhotos()}
	if n.Name == "" {
		nameMissing(w)
		return n, false
	}
	d, ok := readDisposition(w, f.field("disposition"))
	if !ok {
		return n, false
	}
	n.Disposition = d
	for _, value := range f.fields["owner"] {
		id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "unknown_person", msgUnknownPerson)
			return n, false
		}
		n.OwnerIDs = append(n.OwnerIDs, id)
	}
	if len(n.OwnerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "owner_missing", msgOwnerMissing)
		return n, false
	}
	if len(n.Photos) == 0 {
		photoMissing(w)
		return n, false
	}
	if n.Place, ok = placeFromForm(w, f); !ok {
		return n, false
	}
	if box := f.field("box"); box != "" {
		id, err := strconv.ParseInt(box, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "unknown_box", msgUnknownBox)
			return n, false
		}
		n.BoxID = id
	}
	return n, true
}

func (s *server) createItem(w http.ResponseWriter, r *http.Request) {
	f, ok := s.readForm(w, r)
	if !ok {
		return
	}
	n, ok := readNewItem(w, f)
	if !ok {
		s.discardPhotos(f.photos)
		return
	}
	item, err := s.Store.CreateItem(r.Context(), n, s.Now())
	if err != nil {
		s.discardPhotos(f.photos)
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toItemJSON(item))
}

func (s *server) getItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	item, err := s.Store.GetItem(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toItemJSON(item))
}

type itemListJSON struct {
	ID           int64              `json:"id"`
	Name         string             `json:"name"`
	Disposition  *store.Disposition `json:"disposition"`
	ThumbnailURL string             `json:"thumbnailUrl"`
	Place        string             `json:"place"`
	// StorageLocationID is the storage location the item is placed in directly, otherwise null.
	StorageLocationID *int64           `json:"storageLocationId"`
	Move              *moveDetailsJSON `json:"move"`
}

// idOrNull represents the ID 0 as null.
func idOrNull(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func toItemListJSON(entries []store.ItemListEntry) []itemListJSON {
	items := make([]itemListJSON, 0, len(entries))
	for _, e := range entries {
		items = append(items, itemListJSON{
			ID: e.ID, Name: e.Name, Disposition: dispositionJSON(e.Disposition), ThumbnailURL: thumbnailURL(e.MainPhotoID, e.MainPhotoFile),
			Place: e.Place, StorageLocationID: idOrNull(e.StorageLocationID), Move: toMoveDetailsJSON(e.Move),
		})
	}
	return items
}

// listItems filters by q (text in the name), owner (person ID), disposition (including "undecided"),
// room, storageLocation (both including nested storage locations), unstored as well as box, originRoom
// and withoutTargetRoom in the active move, and returns pages with page (from 1) and perPage.
func (s *server) listItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.ItemFilter{Text: strings.TrimSpace(q.Get("q")), Limit: defaultPerPage}
	invalid := func(message string) {
		writeError(w, http.StatusBadRequest, "invalid_request", message)
	}
	for _, param := range []struct {
		name   string
		target *int64
	}{
		{"owner", &filter.OwnerID}, {"room", &filter.RoomID}, {"storageLocation", &filter.StorageLocationID},
		{"box", &filter.BoxID}, {"originRoom", &filter.OriginRoomID},
	} {
		if value := q.Get(param.name); value != "" {
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				invalid("Ungültige ID für " + param.name)
				return
			}
			*param.target = id
		}
	}
	filter.Unstored = q.Get("unstored") != ""
	filter.WithoutTargetRoom = q.Get("withoutTargetRoom") != ""
	if v := q.Get("disposition"); v != "" {
		disposition := store.Undecided
		if v != "undecided" {
			var ok bool
			if disposition, ok = readDisposition(w, v); !ok {
				return
			}
		}
		filter.Disposition = &disposition
	}
	page := 1
	if p := q.Get("page"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 {
			invalid("Ungültige Seite")
			return
		}
		page = n
	}
	if p := q.Get("perPage"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > maxPerPage {
			invalid(fmt.Sprintf("perPage muss zwischen 1 und %d liegen", maxPerPage))
			return
		}
		filter.Limit = n
	}
	filter.Offset = (page - 1) * filter.Limit

	s.writeItemList(w, r, filter)
}

func (s *server) writeItemList(w http.ResponseWriter, r *http.Request, filter store.ItemFilter) {
	entries, more, err := s.Store.ListItems(r.Context(), filter)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": toItemListJSON(entries), "more": more})
}

// removalPreview returns all items that would be left without storage location on deletion. check
// makes sure the storage location or the piece of furniture exists; filter sets it in the filter.
func (s *server) removalPreview(check func(ctx context.Context, id int64) error, filter func(*store.ItemFilter, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		if err := check(r.Context(), id); err != nil {
			s.storeError(w, r, err)
			return
		}
		f := store.ItemFilter{Limit: maxComplete}
		filter(&f, id)
		s.writeItemList(w, r, f)
	}
}

func (s *server) editItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Disposition *string `json:"disposition"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		nameMissing(w)
		return
	}
	disposition := store.Undecided
	if req.Disposition != nil {
		var ok bool
		if disposition, ok = readDisposition(w, *req.Disposition); !ok {
			return
		}
	}
	item, err := s.Store.EditItem(r.Context(), id, name, strings.TrimSpace(req.Description), disposition, s.Now())
	s.writeItem(w, r, item, err)
}

// writeItem answers after a change with the item or the error.
func (s *server) writeItem(w http.ResponseWriter, r *http.Request, item store.Item, err error) {
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toItemJSON(item))
}

func (s *server) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	files, err := s.Store.DeleteItem(r.Context(), id, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	for _, f := range files {
		s.deleteFile(f)
	}
	w.WriteHeader(http.StatusNoContent)
}

// changeOwner adds the person in the path as owner or removes them.
func (s *server) changeOwner(change func(ctx context.Context, itemID, personID int64, now time.Time) (store.Item, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemID, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		personID, ok := pathID(w, r, "personId")
		if !ok {
			return
		}
		item, err := change(r.Context(), itemID, personID, s.Now())
		s.writeItem(w, r, item, err)
	}
}

// changeItems applies one or more changes to several items at once, all or nothing:
// {"itemIds": […], "disposition"?: string|null, "addOwnerId"?: id, "storageLocationId"?: id, "roomId"?: id}.
// Only the fields given are changed; disposition null means undecided.
func (s *server) changeItems(w http.ResponseWriter, r *http.Request) {
	var fields map[string]json.RawMessage
	if !readJSON(w, r, &fields) {
		return
	}
	var req struct {
		ItemIDs           []int64
		AddOwnerID        int64
		StorageLocationID int64
		RoomID            int64
	}
	for name, target := range map[string]any{
		"itemIds": &req.ItemIDs, "addOwnerId": &req.AddOwnerID, "storageLocationId": &req.StorageLocationID, "roomId": &req.RoomID,
	} {
		if raw, present := fields[name]; present {
			if err := json.Unmarshal(raw, target); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_request", "Ungültiger Wert für "+name)
				return
			}
		}
	}
	if len(req.ItemIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Bitte mindestens ein Item wählen")
		return
	}
	c := store.ItemsChange{AddOwnerID: req.AddOwnerID}
	if raw, present := fields["disposition"]; present {
		var value *string
		if err := json.Unmarshal(raw, &value); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_disposition", "Unbekannter Verbleib")
			return
		}
		d := store.Undecided
		if value != nil {
			var ok bool
			if d, ok = readDisposition(w, *value); !ok {
				return
			}
		}
		c.Disposition = &d
	}
	if req.StorageLocationID != 0 || req.RoomID != 0 {
		p, ok := placeFrom(w, req.StorageLocationID, req.RoomID)
		if !ok {
			return
		}
		c.Place = &p
	}
	if c.Disposition == nil && c.AddOwnerID == 0 && c.Place == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Keine Änderung angegeben")
		return
	}
	count, err := s.Store.ChangeItems(r.Context(), req.ItemIDs, c, s.Now())
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}
