package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// zweifarbig ist ein Bild, dessen linke Hälfte rot und rechte Hälfte blau ist.
func zweifarbig(breite, hoehe int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, breite, hoehe))
	for y := range hoehe {
		for x := range breite {
			c := color.RGBA{R: 220, A: 255}
			if x >= breite/2 {
				c = color.RGBA{B: 220, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func testJPEG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, zweifarbig(40, 20), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, zweifarbig(40, 20)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// testWebP ist ein verlustfreies 1×1-WebP.
func testWebP(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// mitExifAusrichtung fügt einem JPEG ein EXIF-Segment mit dem Ausrichtungs-Tag hinzu,
// so wie Handys gedreht gespeicherte Fotos markieren.
func mitExifAusrichtung(t *testing.T, jpg []byte, ausrichtung uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	tiff.WriteString("MM\x00\x2a")
	binary.Write(&tiff, binary.BigEndian, uint32(8))                // Offset von IFD0
	binary.Write(&tiff, binary.BigEndian, uint16(1))                // ein Eintrag
	binary.Write(&tiff, binary.BigEndian, []uint16{0x0112, 3})      // Tag Orientation, Typ SHORT
	binary.Write(&tiff, binary.BigEndian, uint32(1))                // Anzahl
	binary.Write(&tiff, binary.BigEndian, []uint16{ausrichtung, 0}) // Wert, aufgefüllt
	binary.Write(&tiff, binary.BigEndian, uint32(0))                // kein weiteres IFD
	nutzlast := append([]byte("Exif\x00\x00"), tiff.Bytes()...)

	var out bytes.Buffer
	out.Write(jpg[:2]) // SOI
	out.Write([]byte{0xff, 0xe1})
	binary.Write(&out, binary.BigEndian, uint16(len(nutzlast)+2))
	out.Write(nutzlast)
	out.Write(jpg[2:])
	return out.Bytes()
}
