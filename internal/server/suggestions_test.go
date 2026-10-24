package server_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"net/http"
	"testing"

	"inventar/internal/server"
	"inventar/internal/suggest"
)

// fakeSuggester answers without network and remembers the photo it got.
type fakeSuggester struct {
	suggestion suggest.Suggestion
	err        error
	photo      []byte
}

func (f *fakeSuggester) Suggest(_ context.Context, jpeg []byte) (suggest.Suggestion, error) {
	f.photo = jpeg
	return f.suggestion, f.err
}

func withSuggester(s server.Suggester) func(*server.Deps) {
	return func(d *server.Deps) { d.Suggester = s }
}

func TestSuggestionsAreOffWithoutAPIKey(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()

	if res := c.mustStatus(http.StatusOK, "GET", "/api/features", nil); res.Body["suggestions"] != false {
		t.Fatalf("features: %s", res.Raw)
	}
	res := c.multipart("POST", "/api/suggestions", form{photos: [][]byte{testJPEG(t)}})
	if res.Status != http.StatusNotFound || res.errorCode() != "feature_disabled" {
		t.Fatalf("status %d, error code %q", res.Status, res.errorCode())
	}
}

func TestSuggestionForPhoto(t *testing.T) {
	fake := &fakeSuggester{suggestion: suggest.Suggestion{Name: "Stehlampe Messing", Description: "Mit rundem Fuß."}}
	e := newTestEnvWith(t, withSuggester(fake))
	c := e.loggedIn()

	if res := c.mustStatus(http.StatusOK, "GET", "/api/features", nil); res.Body["suggestions"] != true {
		t.Fatalf("features: %s", res.Raw)
	}
	var large bytes.Buffer
	jpeg.Encode(&large, twoColored(3000, 1500), nil)
	res := c.multipart("POST", "/api/suggestions", form{photos: [][]byte{large.Bytes()}})
	if res.Status != http.StatusOK || res.Body["name"] != "Stehlampe Messing" || res.Body["description"] != "Mit rundem Fuß." {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}

	// Claude gets a downscaled JPEG, and nothing is stored.
	img, err := jpeg.Decode(bytes.NewReader(fake.photo))
	if err != nil {
		t.Fatalf("suggester did not get a JPEG: %v", err)
	}
	if got := img.Bounds().Size(); got != (image.Point{X: 1280, Y: 640}) {
		t.Errorf("photo sent is %v, want 1280×640", got)
	}
	e.noPhotoFiles()
}

func TestSuggestionNeedsAnImage(t *testing.T) {
	fake := &fakeSuggester{}
	e := newTestEnvWith(t, withSuggester(fake))
	c := e.loggedIn()

	res := c.multipart("POST", "/api/suggestions", form{photos: [][]byte{[]byte("kein Bild")}})
	if res.Status != http.StatusBadRequest || res.errorCode() != "wrong_file_type" {
		t.Fatalf("wrong type: status %d, error code %q", res.Status, res.errorCode())
	}
	res = c.multipart("POST", "/api/suggestions", form{fields: [][2]string{{"name", "Lampe"}}})
	if res.Status != http.StatusBadRequest || res.errorCode() != "photo_missing" {
		t.Fatalf("no photo: status %d, error code %q", res.Status, res.errorCode())
	}
	if fake.photo != nil {
		t.Error("suggester was called")
	}
}

func TestTooLargePhotoGetsNoSuggestion(t *testing.T) {
	e := newTestEnvWith(t, func(d *server.Deps) {
		d.Suggester = &fakeSuggester{}
		d.Config.MaxPhotoSize = 1000
	})
	c := e.loggedIn()

	large := bytes.Repeat([]byte{0}, 2000)
	copy(large, testPNG(t))
	res := c.multipart("POST", "/api/suggestions", form{photos: [][]byte{large}})
	if res.Status != http.StatusRequestEntityTooLarge || res.errorCode() != "too_large" {
		t.Fatalf("status %d, error code %q", res.Status, res.errorCode())
	}
}

func TestFailedSuggestion(t *testing.T) {
	for name, err := range map[string]error{"API error": errors.New("overloaded"), "refusal": suggest.ErrRefused} {
		e := newTestEnvWith(t, withSuggester(&fakeSuggester{err: err}))
		c := e.loggedIn()

		res := c.multipart("POST", "/api/suggestions", form{photos: [][]byte{testPNG(t)}})
		if res.Status != http.StatusBadGateway || res.errorCode() != "suggestion_failed" {
			t.Errorf("%s: status %d, error code %q", name, res.Status, res.errorCode())
		}
	}
}

func TestSuggestionsNeedASession(t *testing.T) {
	e := newTestEnvWith(t, withSuggester(&fakeSuggester{}))
	c := e.client()

	for _, res := range []response{
		c.do("GET", "/api/features", nil),
		c.multipart("POST", "/api/suggestions", form{photos: [][]byte{testJPEG(t)}}),
	} {
		if res.Status != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", res.Status)
		}
	}
}
