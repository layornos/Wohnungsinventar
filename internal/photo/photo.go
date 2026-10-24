// Package photo stores photos as files in the data directory and creates thumbnails (ADR-0002).
package photo

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	// ThumbnailEdge is the longer edge of the thumbnail in pixels.
	ThumbnailEdge = 512
	// maxPixels protects the Pi from images that blow up memory when decoded.
	maxPixels = 100_000_000
	// thumbnailSuffix replaces the extension of the original for its thumbnail. It stays German
	// so that thumbnails in existing data directories are still found.
	thumbnailSuffix = ".vorschau.jpg"
)

var (
	ErrWrongType = errors.New("not a JPEG, PNG or WebP image")
	ErrTooLarge  = errors.New("photo too large")
)

var extensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// File is a stored photo: name of the original file in the photo folder and its content type.
type File struct {
	Name string
	Type string
}

type Storage struct {
	Dir     string
	MaxSize int64
}

// Save checks the photo by its content, stores the original under a random name and creates
// the thumbnail.
func (s Storage) Save(r io.Reader) (File, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return File{}, err
	}
	head = head[:n]
	typ := http.DetectContentType(head)
	ext, ok := extensions[typ]
	if !ok {
		return File{}, ErrWrongType
	}
	id, err := randomName()
	if err != nil {
		return File{}, err
	}
	f := File{Name: id + ext, Type: typ}

	if err := s.writeOriginal(f.Name, io.MultiReader(bytes.NewReader(head), r)); err != nil {
		return File{}, err
	}
	if err := s.createThumbnail(f.Name); err != nil {
		s.Delete(f.Name)
		return File{}, err
	}
	return f, nil
}

func (s Storage) writeOriginal(name string, r io.Reader) error {
	path := filepath.Join(s.Dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	written, err := io.Copy(f, io.LimitReader(r, s.MaxSize+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && written > s.MaxSize {
		err = ErrTooLarge
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

func (s Storage) createThumbnail(name string) error {
	original, err := os.Open(s.Path(name))
	if err != nil {
		return err
	}
	defer original.Close()
	cfg, _, err := image.DecodeConfig(original)
	if err != nil || int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return ErrWrongType
	}
	if _, err := original.Seek(0, io.SeekStart); err != nil {
		return err
	}
	img, format, err := image.Decode(original)
	if err != nil {
		return ErrWrongType
	}

	thumb := shrink(img, ThumbnailEdge)
	if format == "jpeg" {
		// Phones often store photos unrotated and only note the rotation in EXIF.
		thumb = orient(thumb, s.readOrientation(name))
	}
	f, err := os.OpenFile(s.ThumbnailPath(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	err = jpeg.Encode(f, thumb, &jpeg.Options{Quality: 80})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// readOrientation reads the EXIF orientation from the head of the file, where the EXIF segment is.
func (s Storage) readOrientation(name string) int {
	f, err := os.Open(s.Path(name))
	if err != nil {
		return 1
	}
	defer f.Close()
	head := make([]byte, 128<<10)
	n, _ := io.ReadFull(f, head)
	return exifOrientation(head[:n])
}

// shrink scales the image to the given longer edge; smaller images keep their size.
// Transparent areas turn white because JPEG has no transparency.
func shrink(img image.Image, edge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if max(w, h) > edge {
		if w >= h {
			w, h = edge, max(1, h*edge/w)
		} else {
			w, h = max(1, w*edge/h), edge
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

// Delete removes original and thumbnail; missing files are not an error.
func (s Storage) Delete(name string) error {
	var errs []error
	for _, path := range []string{s.Path(name), s.ThumbnailPath(name)} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s Storage) Path(name string) string { return filepath.Join(s.Dir, name) }

func (s Storage) ThumbnailPath(name string) string {
	return filepath.Join(s.Dir, strings.TrimSuffix(name, filepath.Ext(name))+thumbnailSuffix)
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random name: %w", err)
	}
	return hex.EncodeToString(b), nil
}
