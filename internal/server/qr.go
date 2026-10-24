package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"rsc.io/qr"
)

// qrQuietZone is the white margin around a QR code in modules, as the standard asks for.
const qrQuietZone = 4

// unpackLink builds the link to the unpacking mode of a box, like the invitation link from the
// address the app is currently opened under. The UI route stays German.
func (s *server) unpackLink(r *http.Request, boxID int64) string {
	return fmt.Sprintf("%s://%s/boxen/%d/auspacken", s.scheme(r), r.Host, boxID)
}

// storageLocationLink builds the link to the page of a storage location, the same way as unpackLink.
func (s *server) storageLocationLink(r *http.Request, storageLocationID int64) string {
	return fmt.Sprintf("%s://%s/lagerorte/%d", s.scheme(r), r.Host, storageLocationID)
}

// boxQRCode returns the QR code of the unpack link of a box as SVG, e.g. for the label.
func (s *server) boxQRCode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.Store.GetBox(r.Context(), id); err != nil {
		s.storeError(w, r, err)
		return
	}
	s.writeQRCode(w, r, s.unpackLink(r, id))
}

// storageLocationQRCode returns the QR code of the link to a storage location as SVG, for its label.
func (s *server) storageLocationQRCode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.Store.GetStorageLocation(r.Context(), id); err != nil {
		s.storeError(w, r, err)
		return
	}
	s.writeQRCode(w, r, s.storageLocationLink(r, id))
}

// writeQRCode answers with the QR code of link as SVG.
func (s *server) writeQRCode(w http.ResponseWriter, r *http.Request, link string) {
	svg, err := qrSVG(link)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(svg))
}

// qrSVG renders text as a QR code in SVG: black on white, one unit per module, scalable. Every row
// of dark modules becomes runs "M<x> <y>h<length>v1h-<length>z" of a single path.
func qrSVG(text string) (string, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	size := code.Size + 2*qrQuietZone
	var path strings.Builder
	for y := range code.Size {
		for x := 0; x < code.Size; {
			if !code.Black(x, y) {
				x++
				continue
			}
			start := x
			for x < code.Size && code.Black(x, y) {
				x++
			}
			fmt.Fprintf(&path, "M%d %dh%dv1h-%dz", start+qrQuietZone, y+qrQuietZone, x-start, x-start)
		}
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+
		`<title>%s</title><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="%s"/></svg>`,
		size, size, html.EscapeString(text), size, size, path.String()), nil
}
