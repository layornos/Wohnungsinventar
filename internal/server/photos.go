package server

import (
	"errors"
	"net/http"
	"os"

	"inventar/internal/store"
)

// servePhoto returns the original or the thumbnail of a photo. The address contains the random
// file name and matches exactly one file; the browser may therefore cache it for good, but
// only for itself (private).
func (s *server) servePhoto(thumbnail bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		p, err := s.Store.GetPhoto(r.Context(), id)
		if err == nil && p.File != r.PathValue("file") {
			err = store.ErrNotFound
		}
		if err != nil {
			s.storeError(w, r, err)
			return
		}
		path, typ := s.photos.Path(p.File), p.Type
		if thumbnail {
			path, typ = s.photos.ThumbnailPath(p.File), "image/jpeg"
		}
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "not_found", "Foto-Datei fehlt")
			return
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", typ)
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, "", info.ModTime(), file)
	}
}

func (s *server) addPhotos(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	f, ok := s.readForm(w, r)
	if !ok {
		return
	}
	if len(f.photos) == 0 {
		photoMissing(w)
		return
	}
	item, err := s.Store.AddPhotos(r.Context(), id, f.newPhotos(), s.Now())
	if err != nil {
		s.discardPhotos(f.photos)
	}
	s.writeItem(w, r, item, err)
}

func photoMissing(w http.ResponseWriter) {
	writeError(w, http.StatusBadRequest, "photo_missing", msgPhotoMissing)
}

func (s *server) setMainPhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		PhotoID int64 `json:"photoId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	item, err := s.Store.SetMainPhoto(r.Context(), id, req.PhotoID, s.Now())
	s.writeItem(w, r, item, err)
}

// replacePhoto takes exactly one photo (field "photo") that was rotated or cropped in the
// browser. It is stored like a new photo under a new random name so that the address
// changes; the old file and its thumbnail are removed afterwards.
func (s *server) replacePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	f, ok := s.readForm(w, r)
	if !ok {
		return
	}
	if len(f.photos) == 0 {
		photoMissing(w)
		return
	}
	if len(f.photos) > 1 {
		s.discardPhotos(f.photos)
		writeError(w, http.StatusBadRequest, "too_many_photos", "Genau ein Foto erwartet")
		return
	}
	old, item, err := s.Store.ReplacePhoto(r.Context(), id, f.newPhotos()[0], s.Now())
	if err != nil {
		s.discardPhotos(f.photos)
	} else {
		s.deleteFile(old)
	}
	s.writeItem(w, r, item, err)
}

func (s *server) deletePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	file, item, err := s.Store.DeletePhoto(r.Context(), id, s.Now())
	if err == nil {
		s.deleteFile(file)
	}
	s.writeItem(w, r, item, err)
}
