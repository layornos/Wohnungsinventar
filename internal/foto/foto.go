// Package foto speichert Fotos als Dateien im Datenverzeichnis und erzeugt Vorschauen (ADR-0002).
package foto

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
	// VorschauKante ist die längere Kante der Vorschau in Pixeln.
	VorschauKante = 512
	// maxPixel schützt den Pi vor Bildern, die beim Dekodieren den Speicher sprengen.
	maxPixel = 100_000_000
)

var (
	ErrFalscherTyp = errors.New("kein JPEG-, PNG- oder WebP-Bild")
	ErrZuGross     = errors.New("Foto zu groß")
)

var endungen = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Datei ist ein gespeichertes Foto: Name der Originaldatei im Foto-Ordner und ihr Inhaltstyp.
type Datei struct {
	Name string
	Typ  string
}

type Speicher struct {
	Verzeichnis string
	MaxGroesse  int64
}

// Speichern prüft das Foto anhand seines Inhalts, legt das Original unter einem zufälligen
// Namen ab und erzeugt die Vorschau.
func (s Speicher) Speichern(r io.Reader) (Datei, error) {
	kopf := make([]byte, 512)
	n, err := io.ReadFull(r, kopf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Datei{}, err
	}
	kopf = kopf[:n]
	typ := http.DetectContentType(kopf)
	endung, ok := endungen[typ]
	if !ok {
		return Datei{}, ErrFalscherTyp
	}
	id, err := zufallsName()
	if err != nil {
		return Datei{}, err
	}
	d := Datei{Name: id + endung, Typ: typ}

	if err := s.originalSchreiben(d.Name, io.MultiReader(bytes.NewReader(kopf), r)); err != nil {
		return Datei{}, err
	}
	if err := s.vorschauErzeugen(d.Name); err != nil {
		s.Loeschen(d.Name)
		return Datei{}, err
	}
	return d, nil
}

func (s Speicher) originalSchreiben(name string, r io.Reader) error {
	pfad := filepath.Join(s.Verzeichnis, name)
	f, err := os.OpenFile(pfad, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	geschrieben, err := io.Copy(f, io.LimitReader(r, s.MaxGroesse+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && geschrieben > s.MaxGroesse {
		err = ErrZuGross
	}
	if err != nil {
		os.Remove(pfad)
	}
	return err
}

func (s Speicher) vorschauErzeugen(name string) error {
	original, err := os.Open(s.Pfad(name))
	if err != nil {
		return err
	}
	defer original.Close()
	cfg, _, err := image.DecodeConfig(original)
	if err != nil || int64(cfg.Width)*int64(cfg.Height) > maxPixel {
		return ErrFalscherTyp
	}
	if _, err := original.Seek(0, io.SeekStart); err != nil {
		return err
	}
	img, format, err := image.Decode(original)
	if err != nil {
		return ErrFalscherTyp
	}

	vorschau := verkleinern(img, VorschauKante)
	if format == "jpeg" {
		// Handys speichern Fotos oft ungedreht und vermerken die Drehung nur in EXIF.
		vorschau = ausrichten(vorschau, s.ausrichtungLesen(name))
	}
	f, err := os.OpenFile(s.VorschauPfad(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	err = jpeg.Encode(f, vorschau, &jpeg.Options{Quality: 80})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// ausrichtungLesen liest die EXIF-Ausrichtung aus dem Kopf der Datei, wo das EXIF-Segment liegt.
func (s Speicher) ausrichtungLesen(name string) int {
	f, err := os.Open(s.Pfad(name))
	if err != nil {
		return 1
	}
	defer f.Close()
	kopf := make([]byte, 128<<10)
	n, _ := io.ReadFull(f, kopf)
	return exifAusrichtung(kopf[:n])
}

// verkleinern skaliert das Bild auf die angegebene längere Kante; kleinere Bilder behalten ihre Größe.
// Transparente Flächen werden weiß, weil JPEG keine Transparenz kennt.
func verkleinern(img image.Image, kante int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if max(w, h) > kante {
		if w >= h {
			w, h = kante, max(1, h*kante/w)
		} else {
			w, h = max(1, w*kante/h), kante
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

// Loeschen entfernt Original und Vorschau; fehlende Dateien sind kein Fehler.
func (s Speicher) Loeschen(name string) error {
	var fehler []error
	for _, pfad := range []string{s.Pfad(name), s.VorschauPfad(name)} {
		if err := os.Remove(pfad); err != nil && !errors.Is(err, os.ErrNotExist) {
			fehler = append(fehler, err)
		}
	}
	return errors.Join(fehler...)
}

func (s Speicher) Pfad(name string) string { return filepath.Join(s.Verzeichnis, name) }

func (s Speicher) VorschauPfad(name string) string {
	return filepath.Join(s.Verzeichnis, strings.TrimSuffix(name, filepath.Ext(name))+".vorschau.jpg")
}

func zufallsName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("Zufallsname: %w", err)
	}
	return hex.EncodeToString(b), nil
}
