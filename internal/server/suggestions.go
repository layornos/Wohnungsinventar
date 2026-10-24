package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"inventar/internal/photo"
	"inventar/internal/suggest"
)

// Suggester proposes a name and a description for an item from a JPEG photo (ADR-0006).
type Suggester interface {
	Suggest(ctx context.Context, jpeg []byte) (suggest.Suggestion, error)
}

const (
	// suggestionTimeout bounds the call to Claude including retries.
	suggestionTimeout = 30 * time.Second
	// suggestionEdge is the longer edge of the photo sent to Claude: enough to recognize an item,
	// small enough to keep tokens and upload time low.
	suggestionEdge = 1280
)

type featuresJSON struct {
	Suggestions bool `json:"suggestions"`
}

func (s *server) features(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, featuresJSON{Suggestions: s.Suggester != nil})
}

type suggestionJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// suggest takes one photo (field "photo"), does not store it and returns Claude's suggestion.
func (s *server) suggest(w http.ResponseWriter, r *http.Request) {
	if s.Suggester == nil {
		writeError(w, http.StatusNotFound, "feature_disabled", "Vorschläge sind nicht eingerichtet")
		return
	}
	jpeg, ok := s.readSuggestionPhoto(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), suggestionTimeout)
	defer cancel()
	sg, err := s.Suggester.Suggest(ctx, jpeg)
	if err != nil {
		s.Logger.Warn("suggestion failed", "error", err)
		writeError(w, http.StatusBadGateway, "suggestion_failed", "Kein Vorschlag möglich")
		return
	}
	writeJSON(w, http.StatusOK, suggestionJSON{Name: sg.Name, Description: sg.Description})
}

// readSuggestionPhoto reads the first photo of a multipart request with the same size and type
// checks as stored photos and returns it downscaled. Other fields are ignored.
func (s *server) readSuggestionPhoto(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Formulardaten erwartet")
		return nil, false
	}
	for parts := 0; ; parts++ {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			photoMissing(w)
			return nil, false
		}
		if err != nil || parts >= maxFormParts {
			writeError(w, http.StatusBadRequest, "invalid_request", "Ungültige Formulardaten")
			return nil, false
		}
		if part.FormName() != "photo" {
			continue
		}
		jpeg, err := photo.Scaled(part, s.photos.MaxSize, suggestionEdge)
		switch {
		case errors.Is(err, photo.ErrWrongType):
			writeError(w, http.StatusBadRequest, "wrong_file_type", "Nur JPEG-, PNG- und WebP-Fotos")
		case errors.Is(err, photo.ErrTooLarge):
			writeError(w, http.StatusRequestEntityTooLarge, "too_large",
				fmt.Sprintf("Fotos dürfen höchstens %d MB groß sein", s.photos.MaxSize>>20))
		case err != nil:
			s.internalError(w, r, err)
		default:
			return jpeg, true
		}
		return nil, false
	}
}
