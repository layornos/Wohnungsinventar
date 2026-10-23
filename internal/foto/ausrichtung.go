package foto

import (
	"bytes"
	"encoding/binary"
	"image"
)

// exifAusrichtung liest das Orientation-Tag (1–8) aus dem EXIF-Segment eines JPEGs.
// Ohne lesbares Tag gilt 1, also keine Drehung.
func exifAusrichtung(jpg []byte) int {
	if len(jpg) < 4 || jpg[0] != 0xff || jpg[1] != 0xd8 {
		return 1
	}
	for i := 2; i+4 <= len(jpg); {
		if jpg[i] != 0xff {
			return 1
		}
		marker := jpg[i+1]
		laenge := int(binary.BigEndian.Uint16(jpg[i+2:]))
		ende := i + 2 + laenge
		// Ab dem Bildbeginn (SOS) folgen keine Metadaten mehr.
		if marker == 0xda || laenge < 2 || ende > len(jpg) {
			return 1
		}
		segment := jpg[i+4 : ende]
		if marker == 0xe1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return tiffAusrichtung(segment[6:])
		}
		i = ende
	}
	return 1
}

func tiffAusrichtung(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var ordnung binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		ordnung = binary.LittleEndian
	case "MM":
		ordnung = binary.BigEndian
	default:
		return 1
	}
	ifd := int(ordnung.Uint32(tiff[4:]))
	if ifd+2 > len(tiff) {
		return 1
	}
	eintraege := int(ordnung.Uint16(tiff[ifd:]))
	for n := range eintraege {
		e := ifd + 2 + n*12
		if e+12 > len(tiff) {
			return 1
		}
		if ordnung.Uint16(tiff[e:]) == 0x0112 {
			if wert := int(ordnung.Uint16(tiff[e+8:])); wert >= 1 && wert <= 8 {
				return wert
			}
			return 1
		}
	}
	return 1
}

// ausrichten dreht und spiegelt das Bild so, wie es die EXIF-Ausrichtung verlangt.
func ausrichten(img image.Image, ausrichtung int) image.Image {
	if ausrichtung <= 1 || ausrichtung > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// quelle liefert zu einem Zielpunkt den Punkt im Original.
	var quelle func(x, y int) (int, int)
	switch ausrichtung {
	case 2: // horizontal gespiegelt
		quelle = func(x, y int) (int, int) { return w - 1 - x, y }
	case 3: // 180° gedreht
		quelle = func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }
	case 4: // vertikal gespiegelt
		quelle = func(x, y int) (int, int) { return x, h - 1 - y }
	case 5: // an der Hauptdiagonale gespiegelt
		quelle = func(x, y int) (int, int) { return y, x }
	case 6: // 90° im Uhrzeigersinn drehen
		quelle = func(x, y int) (int, int) { return y, h - 1 - x }
	case 7: // an der Nebendiagonale gespiegelt
		quelle = func(x, y int) (int, int) { return w - 1 - y, h - 1 - x }
	case 8: // 90° gegen den Uhrzeigersinn drehen
		quelle = func(x, y int) (int, int) { return w - 1 - y, x }
	}
	zw, zh := w, h
	if ausrichtung >= 5 {
		zw, zh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, zw, zh))
	for y := range zh {
		for x := range zw {
			qx, qy := quelle(x, y)
			dst.Set(x, y, img.At(b.Min.X+qx, b.Min.Y+qy))
		}
	}
	return dst
}
