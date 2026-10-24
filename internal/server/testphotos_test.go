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

// twoColored is an image whose left half is red and whose right half is blue.
func twoColored(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			c := color.RGBA{R: 220, A: 255}
			if x >= width/2 {
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
	if err := jpeg.Encode(&b, twoColored(40, 20), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, twoColored(40, 20)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// testWebP is a lossless 1×1 WebP.
func testWebP(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// withExifOrientation adds an EXIF segment with the orientation tag to a JPEG,
// the way phones mark photos stored rotated.
func withExifOrientation(t *testing.T, jpg []byte, orientation uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	tiff.WriteString("MM\x00\x2a")
	binary.Write(&tiff, binary.BigEndian, uint32(8))                // offset of IFD0
	binary.Write(&tiff, binary.BigEndian, uint16(1))                // one entry
	binary.Write(&tiff, binary.BigEndian, []uint16{0x0112, 3})      // tag Orientation, type SHORT
	binary.Write(&tiff, binary.BigEndian, uint32(1))                // count
	binary.Write(&tiff, binary.BigEndian, []uint16{orientation, 0}) // value, padded
	binary.Write(&tiff, binary.BigEndian, uint32(0))                // no further IFD
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)

	var out bytes.Buffer
	out.Write(jpg[:2]) // SOI
	out.Write([]byte{0xff, 0xe1})
	binary.Write(&out, binary.BigEndian, uint16(len(payload)+2))
	out.Write(payload)
	out.Write(jpg[2:])
	return out.Bytes()
}
