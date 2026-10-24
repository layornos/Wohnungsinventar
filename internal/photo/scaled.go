package photo

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"
)

// Scaled reads a photo of at most maxSize bytes without storing it, checks it like Save and
// returns it as an upright JPEG whose longer edge is at most edge pixels.
func Scaled(r io.Reader, maxSize int64, edge int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, ErrTooLarge
	}
	head := data[:min(len(data), 512)]
	if _, ok := extensions[http.DetectContentType(head)]; !ok {
		return nil, ErrWrongType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return nil, ErrWrongType
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrWrongType
	}
	scaled := shrink(img, edge)
	if format == "jpeg" {
		scaled = orient(scaled, exifOrientation(data[:min(len(data), 128<<10)]))
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, scaled, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
