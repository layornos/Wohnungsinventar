package server_test

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"testing"

	"inventar/internal/server"
)

func TestOnlyJPEGPNGAndWebPAreAccepted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	me := fmt.Sprint(c.me())

	for name, p := range map[string][]byte{"JPEG": testJPEG(t), "PNG": testPNG(t), "WebP": testWebP(t)} {
		res := c.multipart("POST", "/api/items", form{fields: [][2]string{{"name", name}, {"owner", me}}, photos: [][]byte{p}})
		if res.Status != http.StatusCreated {
			t.Errorf("%s: status %d: %s", name, res.Status, res.Raw)
		}
	}

	// The type follows from the content, not from the file name photo0.jpg.
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	brokenJPEG := testJPEG(t)[:100]
	for name, p := range map[string][]byte{"text": []byte("kein Bild"), "GIF": gif, "broken JPEG": brokenJPEG} {
		res := c.multipart("POST", "/api/items", form{fields: [][2]string{{"name", name}, {"owner", me}}, photos: [][]byte{p}})
		if res.Status != http.StatusBadRequest || res.errorCode() != "wrong_file_type" {
			t.Errorf("%s: status %d, error code %q", name, res.Status, res.errorCode())
		}
	}
}

func TestTooLargePhotosAreRejected(t *testing.T) {
	e := newTestEnvWith(t, func(d *server.Deps) { d.Config.MaxPhotoSize = 1000 })
	c := e.loggedIn()

	large := bytes.Repeat([]byte{0}, 2000)
	copy(large, testPNG(t))
	res := c.multipart("POST", "/api/items", form{fields: [][2]string{{"name", "Lampe"}, {"owner", fmt.Sprint(c.me())}}, photos: [][]byte{large}})
	if res.Status != http.StatusRequestEntityTooLarge || res.errorCode() != "too_large" {
		t.Fatalf("status %d, error code %q", res.Status, res.errorCode())
	}
	e.noPhotoFiles()
}

func TestOriginalAndThumbnailCanBeFetched(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	original := testPNG(t)
	p := photosOf(c.createItem("Lampe", []int64{c.me()}, original))[0]

	res := c.get(p["url"].(string))
	if res.Status != http.StatusOK || !bytes.Equal(res.Raw, original) || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("original: status %d, type %q, %d bytes", res.Status, res.Header.Get("Content-Type"), len(res.Raw))
	}
	thumb := c.get(p["thumbnailUrl"].(string))
	if thumb.Status != http.StatusOK || thumb.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("thumbnail: status %d, type %q", thumb.Status, thumb.Header.Get("Content-Type"))
	}
}

func TestThumbnailIsShrunk(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	var b bytes.Buffer
	jpeg.Encode(&b, twoColored(1200, 600), nil)
	p := photosOf(c.createItem("Lampe", []int64{c.me()}, b.Bytes()))[0]

	img := thumbnailImage(t, c, p)
	if img.Bounds().Dx() != 512 || img.Bounds().Dy() != 256 {
		t.Fatalf("thumbnail is %v, want 512×256", img.Bounds().Size())
	}
}

func TestThumbnailRespectsExifOrientation(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	// The original is 40×20, red on the left, blue on the right. Orientation 6, for example, means:
	// rotate 90° clockwise for display, so the red half ends up at the top.
	for _, tc := range []struct {
		orientation   uint16
		width, height int
		red           string
	}{
		{1, 40, 20, "left"},
		{2, 40, 20, "right"},
		{3, 40, 20, "right"},
		{4, 40, 20, "left"},
		{5, 20, 40, "top"},
		{6, 20, 40, "top"},
		{7, 20, 40, "bottom"},
		{8, 20, 40, "bottom"},
	} {
		original := withExifOrientation(t, testJPEG(t), tc.orientation)
		p := photosOf(c.createItem("Lampe", []int64{c.me()}, original))[0]

		img := thumbnailImage(t, c, p)
		if img.Bounds().Dx() != tc.width || img.Bounds().Dy() != tc.height {
			t.Errorf("orientation %d: thumbnail is %v, want %d×%d", tc.orientation, img.Bounds().Size(), tc.width, tc.height)
			continue
		}
		if got := redSide(img); got != tc.red {
			t.Errorf("orientation %d: red is %s, want %s", tc.orientation, got, tc.red)
		}
		if res := c.get(p["url"].(string)); !bytes.Equal(res.Raw, original) {
			t.Errorf("orientation %d: the original should stay unchanged", tc.orientation)
		}
	}
}

func TestPhotosCannotBeFetchedWithoutSession(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	p := photosOf(c.createItem("Lampe", []int64{c.me()}, testJPEG(t)))[0]

	stranger := e.client()
	for _, url := range []string{p["url"].(string), p["thumbnailUrl"].(string)} {
		if res := stranger.get(url); res.Status != http.StatusUnauthorized {
			t.Errorf("%s without session: status %d", url, res.Status)
		}
	}
}

func thumbnailImage(t *testing.T, c *apiClient, p map[string]any) image.Image {
	t.Helper()
	res := c.get(p["thumbnailUrl"].(string))
	if res.Status != http.StatusOK {
		t.Fatalf("thumbnail: status %d", res.Status)
	}
	img, err := jpeg.Decode(bytes.NewReader(res.Raw))
	if err != nil {
		t.Fatalf("thumbnail is not a JPEG: %v", err)
	}
	return img
}

// redSide says on which side of a two-colored image the red half is.
func redSide(img image.Image) string {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	left, right := image.Pt(w/8, h/2), image.Pt(w*7/8, h/2)
	top, bottom := image.Pt(w/2, h/8), image.Pt(w/2, h*7/8)
	for _, side := range []struct {
		name      string
		red, blue image.Point
	}{
		{"left", left, right}, {"right", right, left}, {"top", top, bottom}, {"bottom", bottom, top},
	} {
		if colorAt(img, side.red) == "red" && colorAt(img, side.blue) == "blue" {
			return side.name
		}
	}
	return "unclear"
}

func colorAt(img image.Image, p image.Point) string {
	r, _, b, _ := img.At(p.X, p.Y).RGBA()
	switch {
	case r > 2*b:
		return "red"
	case b > 2*r:
		return "blue"
	}
	return "unclear"
}

func mainPhotoID(r response) any {
	for _, p := range photosOf(r) {
		if p["mainPhoto"] == true {
			return p["id"]
		}
	}
	return nil
}

func TestAddPhotosAndSetMainPhoto(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))

	res := c.multipart("POST", itemPath(item)+"/photos", form{photos: [][]byte{testPNG(t), testWebP(t)}})
	if res.Status != http.StatusOK || len(photosOf(res)) != 3 {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	if mainPhotoID(res) != photosOf(item)[0]["id"] {
		t.Fatal("new photos should not change the main photo")
	}

	thirdPhoto := photosOf(res)[2]
	third := thirdPhoto["id"]
	res = c.mustStatus(http.StatusOK, "PUT", itemPath(item)+"/main-photo", map[string]any{"photoId": third})
	if mainPhotoID(res) != third {
		t.Fatalf("main photo: %s", res.Raw)
	}
	list := c.mustStatus(http.StatusOK, "GET", "/api/items", nil)
	if got := list.Body["items"].([]any)[0].(map[string]any)["thumbnailUrl"]; got != thirdPhoto["thumbnailUrl"] {
		t.Fatalf("the list should show the thumbnail of the main photo, shows %v", got)
	}

	// A photo of another item cannot be set as main photo.
	other := c.createItem("Schere", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusNotFound, "PUT", itemPath(item)+"/main-photo", map[string]any{"photoId": photosOf(other)[0]["id"]})

	if res := c.multipart("POST", itemPath(item)+"/photos", form{}); res.errorCode() != "photo_missing" {
		t.Fatalf("without photo: status %d, error code %q", res.Status, res.errorCode())
	}
	if res := c.multipart("POST", itemPath(item)+"/photos", form{photos: [][]byte{[]byte("kein Bild")}}); res.errorCode() != "wrong_file_type" {
		t.Fatalf("wrong type: status %d, error code %q", res.Status, res.errorCode())
	}
}

func TestAfterDeletingTheMainPhotoTheNextOneTakesOver(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t), testPNG(t), testWebP(t))
	photos := photosOf(item)

	res := c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/photos/%v", photos[0]["id"]), nil)
	if len(photosOf(res)) != 2 || mainPhotoID(res) != photos[1]["id"] {
		t.Fatalf("after deleting: %s", res.Raw)
	}
	if r := c.get(photos[0]["url"].(string)); r.Status != http.StatusNotFound {
		t.Errorf("deleted photo: status %d", r.Status)
	}

	// Deleting a photo other than the main photo does not change the main photo.
	res = c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/photos/%v", photos[2]["id"]), nil)
	if len(photosOf(res)) != 1 || mainPhotoID(res) != photos[1]["id"] {
		t.Fatalf("after deleting a second time: %s", res.Raw)
	}
}

func TestLastPhotoCannotBeDeleted(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	p := photosOf(item)[0]

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/photos/%v", p["id"]), nil)
	if res.errorCode() != "last_photo" {
		t.Fatalf("error code %q", res.errorCode())
	}
	if r := c.get(p["thumbnailUrl"].(string)); r.Status != http.StatusOK {
		t.Fatalf("the photo should be kept: status %d", r.Status)
	}
}

func TestDeletedPhotosDisappearFromThePhotoFolder(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t), testPNG(t))
	c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/photos/%v", photosOf(item)[0]["id"]), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", itemPath(item), nil)
	e.noPhotoFiles()
}

func TestTransparentAreasTurnWhiteInTheThumbnail(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	var b bytes.Buffer
	png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	p := photosOf(c.createItem("Aufkleber", []int64{c.me()}, b.Bytes()))[0]

	r, g, bl, _ := thumbnailImage(t, c, p).At(5, 5).RGBA()
	if r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Fatalf("pixel (%d, %d, %d), want white", r>>8, g>>8, bl>>8)
	}
}

// Photos may stay in the browser cache for good. A new photo therefore needs an address under
// which no other image has ever been served, even if one was deleted before.
func TestNewPhotoGetsANewAddressAfterDeleting(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t), testPNG(t))
	old := photosOf(item)[1]
	c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/photos/%v", old["id"]), nil)

	newImage := testWebP(t)
	res := c.multipart("POST", itemPath(item)+"/photos", form{photos: [][]byte{newImage}})
	if res.Status != http.StatusOK || len(photosOf(res)) != 2 {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	added := photosOf(res)[1]
	if added["id"] == old["id"] {
		t.Errorf("the ID %v of the deleted photo was given out again", old["id"])
	}
	for _, field := range []string{"url", "thumbnailUrl"} {
		if added[field] == old[field] {
			t.Errorf("%s: the new photo has the address of the deleted one: %v", field, added[field])
		}
		if r := c.get(old[field].(string)); r.Status != http.StatusNotFound {
			t.Errorf("%s of the deleted photo: status %d", field, r.Status)
		}
	}
	if r := c.get(added["url"].(string)); r.Status != http.StatusOK || !bytes.Equal(r.Raw, newImage) {
		t.Fatalf("new photo: status %d, %d bytes", r.Status, len(r.Raw))
	}
}

// Only the address with the right file name returns the photo, otherwise an old address
// could show a different image after the file changed.
func TestPhotoWithWrongFileNameIsNotServed(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	p := photosOf(c.createItem("Lampe", []int64{c.me()}, testJPEG(t)))[0]

	for _, url := range []string{
		fmt.Sprintf("/api/photos/%v", p["id"]),
		fmt.Sprintf("/api/photos/%v/thumbnail", p["id"]),
		fmt.Sprintf("/api/photos/%v/wrong.jpg", p["id"]),
		fmt.Sprintf("/api/photos/%v/wrong.jpg/thumbnail", p["id"]),
	} {
		if r := c.get(url); r.Status != http.StatusNotFound && r.Status != http.StatusMethodNotAllowed {
			t.Errorf("%s: status %d", url, r.Status)
		}
	}
}

// A photo rotated or cropped in the browser replaces the original. It keeps its ID, position
// in the order and main photo flag, but gets a new address because the browser caches the
// old one for good.
func TestEditedPhotoReplacesTheOriginal(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.createItem("Lampe", []int64{c.me()}, testJPEG(t), testPNG(t), testWebP(t))
	before := photosOf(c.mustStatus(http.StatusOK, "PUT", itemPath(item)+"/main-photo", map[string]any{"photoId": photosOf(item)[1]["id"]}))
	old := before[1]

	var b bytes.Buffer
	jpeg.Encode(&b, twoColored(20, 40), nil)
	edited := b.Bytes()
	res := c.multipart("PUT", fmt.Sprintf("/api/photos/%v", old["id"]), form{photos: [][]byte{edited}})
	if res.Status != http.StatusOK {
		t.Fatalf("status %d: %s", res.Status, res.Raw)
	}
	after := photosOf(res)
	if len(after) != 3 {
		t.Fatalf("%d photos, want 3", len(after))
	}
	for i := range before {
		if after[i]["id"] != before[i]["id"] || after[i]["mainPhoto"] != before[i]["mainPhoto"] {
			t.Errorf("photo %d: before %v, after %v", i, before[i], after[i])
		}
		if i != 1 && (after[i]["url"] != before[i]["url"] || after[i]["thumbnailUrl"] != before[i]["thumbnailUrl"]) {
			t.Errorf("photo %d was not edited, its address should stay the same", i)
		}
	}
	replaced := after[1]
	for _, field := range []string{"url", "thumbnailUrl"} {
		if replaced[field] == old[field] {
			t.Errorf("%s: the edited photo has the old address %v", field, replaced[field])
		}
		if r := c.get(old[field].(string)); r.Status != http.StatusNotFound {
			t.Errorf("old %s: status %d", field, r.Status)
		}
	}
	if r := c.get(replaced["url"].(string)); r.Status != http.StatusOK || !bytes.Equal(r.Raw, edited) || r.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("new original: status %d, type %q, %d bytes", r.Status, r.Header.Get("Content-Type"), len(r.Raw))
	}
	if img := thumbnailImage(t, c, replaced); img.Bounds().Dx() != 20 || img.Bounds().Dy() != 40 {
		t.Fatalf("the thumbnail should match the edited photo, is %v", img.Bounds().Size())
	}
	list := c.mustStatus(http.StatusOK, "GET", "/api/items", nil)
	if got := list.Body["items"].([]any)[0].(map[string]any)["thumbnailUrl"]; got != replaced["thumbnailUrl"] {
		t.Errorf("the list should show the new thumbnail of the main photo, shows %v", got)
	}
	// The old file and its thumbnail are cleaned up.
	e.noOrphanedPhotos(3)
}

func TestReplacePhotoRejectsInvalidRequests(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	p := photosOf(c.createItem("Lampe", []int64{c.me()}, testJPEG(t)))[0]
	path := fmt.Sprintf("/api/photos/%v", p["id"])

	for _, tc := range []struct {
		name   string
		path   string
		photos [][]byte
		status int
		code   string
	}{
		{"unknown ID", "/api/photos/9999", [][]byte{testPNG(t)}, http.StatusNotFound, "not_found"},
		{"no ID", "/api/photos/abc", [][]byte{testPNG(t)}, http.StatusNotFound, "not_found"},
		{"without photo", path, nil, http.StatusBadRequest, "photo_missing"},
		{"two photos", path, [][]byte{testPNG(t), testPNG(t)}, http.StatusBadRequest, "too_many_photos"},
		{"not an image", path, [][]byte{[]byte("kein Bild")}, http.StatusBadRequest, "wrong_file_type"},
	} {
		res := c.multipart("PUT", tc.path, form{photos: tc.photos})
		if res.Status != tc.status || res.errorCode() != tc.code {
			t.Errorf("%s: status %d, error code %q", tc.name, res.Status, res.errorCode())
		}
	}
	if res := e.client().multipart("PUT", path, form{photos: [][]byte{testPNG(t)}}); res.Status != http.StatusUnauthorized {
		t.Errorf("without session: status %d", res.Status)
	}

	// The original stays, and rejected uploads leave no files behind.
	if r := c.get(p["url"].(string)); r.Status != http.StatusOK {
		t.Errorf("original: status %d", r.Status)
	}
	e.noOrphanedPhotos(1)
}
