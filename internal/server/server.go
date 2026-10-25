// Package server provides the JSON API and the embedded client.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"inventar/internal/config"
	"inventar/internal/mail"
	"inventar/internal/photo"
	"inventar/internal/store"
	"inventar/internal/webui"
)

// Store is the part of data access the API needs (ADR-0002), grouped by domain area.
type Store interface {
	AccessStore
	PersonStore
	ItemStore
	PlaceStore
	MoveStore
	InvitationStore
	SaleStore
}

// SaleStore covers spec 06: handing items over into the archive, household settings and the proceeds summary.
type SaleStore interface {
	HandOver(ctx context.Context, id int64, h store.Handover, now time.Time) (store.Item, error)
	TakeBack(ctx context.Context, id int64, now time.Time) (store.Item, error)
	Settings(ctx context.Context, keys ...string) (map[string]string, error)
	SaveSettings(ctx context.Context, values map[string]string) error
	ProceedsSummary(ctx context.Context, from, to string, ownerID int64) (store.Proceeds, error)
}

type MoveStore interface {
	ActiveMove(ctx context.Context) (store.Move, error)
	GetMove(ctx context.Context, id int64) (store.Move, error)
	ListMoves(ctx context.Context) ([]store.Move, error)
	CreateMove(ctx context.Context, oldApartmentID, newApartmentID int64, now time.Time) (store.Move, error)
	CreateBox(ctx context.Context, label string, targetRoomID int64, now time.Time) (store.Box, error)
	GetBox(ctx context.Context, id int64) (store.Box, error)
	BoxByNumber(ctx context.Context, number int) (store.Box, error)
	EditBox(ctx context.Context, id int64, label string, targetRoomID *int64) (store.Box, error)
	PackStorageLocation(ctx context.Context, boxID, storageLocationID int64, now time.Time) (int, store.Box, error)
	UnpackBox(ctx context.Context, boxID int64, itemIDs []int64, p store.Place, now time.Time) (int, store.Box, error)
	DeleteBox(ctx context.Context, id int64) error
	SetMoveDetails(ctx context.Context, itemID int64, c store.MoveDetailsChange, now time.Time) (store.Item, error)
	SetMoveDetailsBulk(ctx context.Context, itemIDs []int64, c store.MoveDetailsChange, now time.Time) error
	TargetRoomOverview(ctx context.Context) ([]store.TargetRoomEntry, error)
	CompleteMove(ctx context.Context, confirmed bool, now time.Time) (store.Move, []store.ItemListEntry, error)
}

type AccessStore interface {
	UserByEmail(ctx context.Context, email string) (store.User, error)
	CreateSession(ctx context.Context, tokenHash string, personID int64, now, expiresAt time.Time) error
	SessionUser(ctx context.Context, tokenHash string, now time.Time) (store.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	SetTheme(ctx context.Context, personID int64, t store.Theme) (store.User, error)
}

type PersonStore interface {
	ListPersons(ctx context.Context) ([]store.Person, error)
	CreatePerson(ctx context.Context, name string, now time.Time) (store.Person, error)
	RenamePerson(ctx context.Context, id int64, name string) (store.Person, error)
	DeletePerson(ctx context.Context, id int64) error
}

type ItemStore interface {
	CreateItem(ctx context.Context, n store.NewItem, now time.Time) (store.Item, error)
	GetItem(ctx context.Context, id int64) (store.Item, error)
	ListItems(ctx context.Context, f store.ItemFilter) ([]store.ItemListEntry, bool, error)
	EditItem(ctx context.Context, id int64, e store.ItemEdit, now time.Time) (store.Item, error)
	DeleteItem(ctx context.Context, id int64, now time.Time) ([]string, error)
	AddOwner(ctx context.Context, itemID, personID int64, now time.Time) (store.Item, error)
	RemoveOwner(ctx context.Context, itemID, personID int64, now time.Time) (store.Item, error)
	GetPhoto(ctx context.Context, id int64) (store.Photo, error)
	AddPhotos(ctx context.Context, itemID int64, photos []store.NewPhoto, now time.Time) (store.Item, error)
	SetMainPhoto(ctx context.Context, itemID, photoID int64, now time.Time) (store.Item, error)
	ReplacePhoto(ctx context.Context, photoID int64, n store.NewPhoto, now time.Time) (string, store.Item, error)
	DeletePhoto(ctx context.Context, photoID int64, now time.Time) (string, store.Item, error)
	SetPlacement(ctx context.Context, itemID int64, p store.Place, now time.Time) (store.Item, error)
	ChangeItems(ctx context.Context, itemIDs []int64, c store.ItemsChange, now time.Time) (int, error)
}

// PlaceStore covers apartments, rooms and storage locations.
type PlaceStore interface {
	ListApartments(ctx context.Context) ([]store.Apartment, error)
	CreateApartment(ctx context.Context, name string, now time.Time) (store.Named, error)
	RenameApartment(ctx context.Context, id int64, name string) (store.Named, error)
	DeleteApartment(ctx context.Context, id int64) error
	CreateRoom(ctx context.Context, apartmentID int64, name string, now time.Time) (store.Named, error)
	GetRoom(ctx context.Context, id int64) (store.Room, error)
	RenameRoom(ctx context.Context, id int64, name string) (store.Named, error)
	DeleteRoom(ctx context.Context, id int64) error
	CreateStorageLocation(ctx context.Context, n store.NewStorageLocation, now time.Time) (store.StorageLocationDetail, error)
	GetStorageLocation(ctx context.Context, id int64) (store.StorageLocationDetail, error)
	RenameStorageLocation(ctx context.Context, id int64, name string) (store.Named, error)
	MoveStorageLocation(ctx context.Context, id, roomID, parentID int64) (store.StorageLocationDetail, error)
	DeleteStorageLocation(ctx context.Context, id int64, now time.Time) error
	SetFurniture(ctx context.Context, storageLocationID, itemID int64) (store.StorageLocationDetail, error)
	ListPlaces(ctx context.Context) ([]store.PlaceEntry, error)
}

// InvitationStore covers invitations and the user list.
type InvitationStore interface {
	CreateInvitation(ctx context.Context, n store.NewInvitation, now, expiresAt time.Time) (store.Invitation, error)
	OpenInvitations(ctx context.Context, now time.Time) ([]store.Invitation, error)
	RevokeInvitation(ctx context.Context, id int64, now time.Time) error
	InvitationByToken(ctx context.Context, tokenHash string, now time.Time) (store.Invitation, error)
	AcceptInvitation(ctx context.Context, a store.Acceptance, now time.Time) (store.User, error)
	ListUsers(ctx context.Context) ([]store.User, error)
}

type Deps struct {
	Config config.Config
	Store  Store
	Mailer mail.Mailer
	Logger *slog.Logger
	Now    func() time.Time
	// Client holds the files of the built client; the embedded ones if not set.
	Client fs.FS
	// Suggester proposes names and descriptions from photos; nil turns the feature off (ADR-0006).
	Suggester Suggester
}

type server struct {
	Deps
	photos photo.Storage
}

func New(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Client == nil {
		d.Client = webui.Dist()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	maxPhoto := d.Config.MaxPhotoSize
	if maxPhoto == 0 {
		maxPhoto = config.DefaultMaxPhotoSize
	}
	s := &server{Deps: d, photos: photo.Storage{
		Dir:     filepath.Join(d.Config.DataDir, store.PhotoDir),
		MaxSize: maxPhoto,
	}}

	// Everything except logging in and accepting an invitation needs a session, unknown endpoints too.
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/logout", s.logout)
	protected.HandleFunc("GET /api/me", s.me)
	protected.HandleFunc("PUT /api/me/theme", s.setTheme)
	protected.HandleFunc("GET /api/features", s.features)
	protected.HandleFunc("POST /api/suggestions", s.suggest)
	protected.HandleFunc("GET /api/persons", s.listPersons)
	protected.HandleFunc("POST /api/persons", s.createPerson)
	protected.HandleFunc("PUT /api/persons/{id}", s.renamePerson)
	protected.HandleFunc("DELETE /api/persons/{id}", s.deletePerson)
	protected.HandleFunc("GET /api/items", s.listItems)
	protected.HandleFunc("POST /api/items", s.createItem)
	protected.HandleFunc("POST /api/items/bulk", s.changeItems)
	protected.HandleFunc("GET /api/items/{id}", s.getItem)
	protected.HandleFunc("PUT /api/items/{id}", s.editItem)
	protected.HandleFunc("DELETE /api/items/{id}", s.deleteItem)
	protected.HandleFunc("PUT /api/items/{id}/owners/{personId}", s.changeOwner(s.Store.AddOwner))
	protected.HandleFunc("DELETE /api/items/{id}/owners/{personId}", s.changeOwner(s.Store.RemoveOwner))
	protected.HandleFunc("POST /api/items/{id}/photos", s.addPhotos)
	protected.HandleFunc("PUT /api/items/{id}/main-photo", s.setMainPhoto)
	protected.HandleFunc("GET /api/photos/{id}/{file}", s.servePhoto(false))
	protected.HandleFunc("PUT /api/photos/{id}", s.replacePhoto)
	protected.HandleFunc("DELETE /api/photos/{id}", s.deletePhoto)
	protected.HandleFunc("GET /api/photos/{id}/{file}/thumbnail", s.servePhoto(true))
	protected.HandleFunc("GET /api/apartments", s.listApartments)
	protected.HandleFunc("POST /api/apartments", s.createApartment)
	protected.HandleFunc("PUT /api/apartments/{id}", s.rename(s.Store.RenameApartment))
	protected.HandleFunc("DELETE /api/apartments/{id}", s.delete(s.Store.DeleteApartment))
	protected.HandleFunc("POST /api/apartments/{id}/rooms", s.createRoom)
	protected.HandleFunc("GET /api/rooms/{id}", s.getRoom)
	protected.HandleFunc("PUT /api/rooms/{id}", s.rename(s.Store.RenameRoom))
	protected.HandleFunc("DELETE /api/rooms/{id}", s.delete(s.Store.DeleteRoom))
	protected.HandleFunc("GET /api/places", s.listPlaces)
	protected.HandleFunc("POST /api/storage-locations", s.createStorageLocation)
	protected.HandleFunc("GET /api/storage-locations/{id}", s.getStorageLocation)
	protected.HandleFunc("PUT /api/storage-locations/{id}", s.rename(s.Store.RenameStorageLocation))
	protected.HandleFunc("DELETE /api/storage-locations/{id}", s.deleteStorageLocation)
	protected.HandleFunc("PUT /api/storage-locations/{id}/parent", s.moveStorageLocation)
	protected.HandleFunc("GET /api/storage-locations/{id}/removal", s.removalPreview(
		func(ctx context.Context, id int64) error { _, err := s.Store.GetStorageLocation(ctx, id); return err },
		func(f *store.ItemFilter, id int64) { f.StorageLocationID = id }))
	protected.HandleFunc("GET /api/items/{id}/removal", s.removalPreview(
		func(ctx context.Context, id int64) error { _, err := s.Store.GetItem(ctx, id); return err },
		func(f *store.ItemFilter, id int64) { f.FurnitureID = id }))
	protected.HandleFunc("PUT /api/storage-locations/{id}/furniture", s.setFurniture)
	protected.HandleFunc("GET /api/storage-locations/{id}/qr", s.storageLocationQRCode)
	protected.HandleFunc("PUT /api/items/{id}/placement", s.setPlacement)
	protected.HandleFunc("GET /api/move", s.activeMove)
	protected.HandleFunc("POST /api/move", s.createMove)
	protected.HandleFunc("POST /api/move/boxes", s.createBox)
	protected.HandleFunc("GET /api/move/boxes/{number}", s.boxByNumber)
	protected.HandleFunc("GET /api/move/target-rooms", s.targetRooms)
	protected.HandleFunc("POST /api/move/completion", s.completeMove)
	protected.HandleFunc("GET /api/boxes/{id}", s.getBox)
	protected.HandleFunc("PUT /api/boxes/{id}", s.editBox)
	protected.HandleFunc("POST /api/boxes/{id}/pack", s.packStorageLocation)
	protected.HandleFunc("POST /api/boxes/{id}/unpack", s.unpackBox)
	protected.HandleFunc("GET /api/boxes/{id}/qr", s.boxQRCode)
	protected.HandleFunc("DELETE /api/boxes/{id}", s.delete(s.Store.DeleteBox))
	protected.HandleFunc("PATCH /api/items/{id}/move", s.setMoveDetails)
	protected.HandleFunc("POST /api/move/bulk", s.setMoveDetailsBulk)
	protected.HandleFunc("GET /api/moves", s.listMoves)
	protected.HandleFunc("GET /api/moves/{id}", s.getMove)
	protected.HandleFunc("GET /api/items/{id}/listing", s.listingText)
	protected.HandleFunc("PUT /api/items/{id}/handover", s.handOver)
	protected.HandleFunc("DELETE /api/items/{id}/handover", s.takeBack)
	protected.HandleFunc("GET /api/archive", s.listArchive)
	protected.HandleFunc("GET /api/proceeds", s.proceeds)
	protected.HandleFunc("GET /api/settings/listing", s.listingSettings)
	protected.HandleFunc("PUT /api/settings/listing", s.saveListingSettings)
	protected.HandleFunc("GET /api/users", s.listUsers)
	protected.HandleFunc("GET /api/invitations", s.listInvitations)
	protected.HandleFunc("POST /api/invitations", s.createInvitation)
	protected.HandleFunc("DELETE /api/invitations/{id}", s.revokeInvitation)
	protected.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Unbekannter Endpunkt")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/invitations/check", s.checkInvitation)
	mux.HandleFunc("POST /api/invitations/accept", s.acceptInvitation)
	mux.Handle("/api/", s.requireSession(protected))
	mux.Handle("/", webui.Handler(d.Client))
	return mux
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError answers with an error code for the client and a German message for the UI.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiError{Code: code, Message: message})
}

func (s *server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.Logger.Error("internal error", "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal", "Interner Fehler")
}

// readJSON reads a small JSON body and answers errors itself.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", "Anfrage zu groß")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültiges JSON")
		}
		return false
	}
	return true
}

// pathID reads a numeric ID from the path; unusable IDs count as not found.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Nicht gefunden")
		return 0, false
	}
	return id, true
}

// storeError translates domain errors of the store into API errors.
func (s *server) storeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Nicht gefunden")
	case errors.Is(err, store.ErrHasUserAccess):
		writeError(w, http.StatusConflict, "has_user_access", "Personen mit Nutzerzugang können nicht gelöscht werden")
	case errors.Is(err, store.ErrLastOwner):
		writeError(w, http.StatusConflict, "last_owner", msgOwnerMissing)
	case errors.Is(err, store.ErrLastPhoto):
		writeError(w, http.StatusConflict, "last_photo", msgPhotoMissing)
	case errors.Is(err, store.ErrApartmentNotEmpty):
		writeError(w, http.StatusConflict, "apartment_not_empty", "Die Wohnung hat noch Räume")
	case errors.Is(err, store.ErrRoomNotEmpty):
		writeError(w, http.StatusConflict, "room_not_empty", "Im Raum sind noch Lagerorte oder Items")
	case errors.Is(err, store.ErrCycle):
		writeError(w, http.StatusConflict, "cycle", "Ein Lagerort kann nicht in sich selbst oder einem seiner Unterlagerorte liegen")
	case errors.Is(err, store.ErrFurnitureInsideItself):
		writeError(w, http.StatusConflict, "furniture_inside_itself", "Ein Möbelstück kann nicht in einem eigenen Lagerort liegen")
	case errors.Is(err, store.ErrUnknownPlace):
		writeError(w, http.StatusBadRequest, "unknown_place", "Diesen Raum oder Lagerort gibt es nicht")
	case errors.Is(err, store.ErrNoMove):
		writeError(w, http.StatusConflict, "no_move", "Es läuft kein Umzug")
	case errors.Is(err, store.ErrMoveInProgress):
		writeError(w, http.StatusConflict, "move_in_progress", "Es läuft schon ein Umzug")
	case errors.Is(err, store.ErrMoveCompleted):
		writeError(w, http.StatusConflict, "move_completed", "Der Umzug ist abgeschlossen und lässt sich nicht mehr ändern")
	case errors.Is(err, store.ErrSameApartment):
		writeError(w, http.StatusBadRequest, "same_apartment", "Alte und neue Wohnung müssen verschieden sein")
	case errors.Is(err, store.ErrUnknownApartment):
		writeError(w, http.StatusBadRequest, "unknown_apartment", "Diese Wohnung gibt es nicht")
	case errors.Is(err, store.ErrApartmentInMove):
		writeError(w, http.StatusConflict, "apartment_in_move", "Die Wohnung gehört zu einem Umzug")
	case errors.Is(err, store.ErrRoomInMove):
		writeError(w, http.StatusConflict, "room_in_move", "Der Raum ist Herkunfts- oder Zielraum in einem Umzug")
	case errors.Is(err, store.ErrBoxNotEmpty):
		writeError(w, http.StatusConflict, "box_not_empty", "Nur leere Boxen lassen sich löschen")
	case errors.Is(err, store.ErrUnknownBox):
		writeError(w, http.StatusBadRequest, "unknown_box", msgUnknownBox)
	case errors.Is(err, store.ErrTargetRoomNotInNewApartment):
		writeError(w, http.StatusBadRequest, "target_room_not_in_new_apartment", "Der Zielraum muss in der neuen Wohnung liegen")
	case errors.Is(err, store.ErrPlaceNotInNewApartment):
		writeError(w, http.StatusBadRequest, "place_not_in_new_apartment", "Der Platz muss in der neuen Wohnung liegen")
	case errors.Is(err, store.ErrUnknownItem):
		writeError(w, http.StatusBadRequest, "unknown_item", "Dieses Item gibt es nicht")
	case errors.Is(err, store.ErrUnknownPerson):
		writeError(w, http.StatusBadRequest, "unknown_person", msgUnknownPerson)
	case errors.Is(err, store.ErrArchived):
		writeError(w, http.StatusConflict, "item_archived", "Das Item ist im Archiv und lässt sich nicht ändern")
	case errors.Is(err, store.ErrNotArchived):
		writeError(w, http.StatusConflict, "not_archived", "Das Item ist nicht im Archiv")
	case errors.Is(err, store.ErrNoSale):
		writeError(w, http.StatusBadRequest, "no_sale", msgNoSale)
	default:
		s.internalError(w, r, err)
	}
}
