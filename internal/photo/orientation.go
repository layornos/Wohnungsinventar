package photo

import (
	"bytes"
	"encoding/binary"
	"image"
)

// exifOrientation reads the Orientation tag (1–8) from the EXIF segment of a JPEG.
// Without a readable tag it is 1, i.e. no rotation.
func exifOrientation(jpg []byte) int {
	if len(jpg) < 4 || jpg[0] != 0xff || jpg[1] != 0xd8 {
		return 1
	}
	for i := 2; i+4 <= len(jpg); {
		if jpg[i] != 0xff {
			return 1
		}
		marker := jpg[i+1]
		length := int(binary.BigEndian.Uint16(jpg[i+2:]))
		end := i + 2 + length
		// From the start of scan (SOS) on there is no more metadata.
		if marker == 0xda || length < 2 || end > len(jpg) {
			return 1
		}
		segment := jpg[i+4 : end]
		if marker == 0xe1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return tiffOrientation(segment[6:])
		}
		i = end
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd+2 > len(tiff) {
		return 1
	}
	entries := int(order.Uint16(tiff[ifd:]))
	for n := range entries {
		e := ifd + 2 + n*12
		if e+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[e:]) == 0x0112 {
			if value := int(order.Uint16(tiff[e+8:])); value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}

// orient rotates and mirrors the image as the EXIF orientation demands.
func orient(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// source returns, for a target point, the point in the original.
	var source func(x, y int) (int, int)
	switch orientation {
	case 2: // mirrored horizontally
		source = func(x, y int) (int, int) { return w - 1 - x, y }
	case 3: // rotated by 180°
		source = func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }
	case 4: // mirrored vertically
		source = func(x, y int) (int, int) { return x, h - 1 - y }
	case 5: // mirrored along the main diagonal
		source = func(x, y int) (int, int) { return y, x }
	case 6: // rotate 90° clockwise
		source = func(x, y int) (int, int) { return y, h - 1 - x }
	case 7: // mirrored along the anti-diagonal
		source = func(x, y int) (int, int) { return w - 1 - y, h - 1 - x }
	case 8: // rotate 90° counterclockwise
		source = func(x, y int) (int, int) { return w - 1 - y, x }
	}
	tw, th := w, h
	if orientation >= 5 {
		tw, th = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := range th {
		for x := range tw {
			sx, sy := source(x, y)
			dst.Set(x, y, img.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}
