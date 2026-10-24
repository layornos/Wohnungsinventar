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

func TestNurJPEGPNGUndWebPWerdenAngenommen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	ich := fmt.Sprint(c.ich())

	for name, foto := range map[string][]byte{"JPEG": testJPEG(t), "PNG": testPNG(t), "WebP": testWebP(t)} {
		res := c.multipart("POST", "/api/items", formular{felder: [][2]string{{"name", name}, {"besitzer", ich}}, fotos: [][]byte{foto}})
		if res.Status != http.StatusCreated {
			t.Errorf("%s: Status %d: %s", name, res.Status, res.Raw)
		}
	}

	// Der Typ ergibt sich aus dem Inhalt, nicht aus dem Dateinamen foto0.jpg.
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	kaputtesJPEG := testJPEG(t)[:100]
	for name, foto := range map[string][]byte{"Text": []byte("kein Bild"), "GIF": gif, "kaputtes JPEG": kaputtesJPEG} {
		res := c.multipart("POST", "/api/items", formular{felder: [][2]string{{"name", name}, {"besitzer", ich}}, fotos: [][]byte{foto}})
		if res.Status != http.StatusBadRequest || res.errorCode() != "falscher_dateityp" {
			t.Errorf("%s: Status %d, Fehlercode %q", name, res.Status, res.errorCode())
		}
	}
}

func TestZuGrosseFotosWerdenAbgelehnt(t *testing.T) {
	e := newTestEnvMit(t, func(d *server.Deps) { d.Config.MaxFotoGroesse = 1000 })
	c := e.loggedIn()

	groß := bytes.Repeat([]byte{0}, 2000)
	copy(groß, testPNG(t))
	res := c.multipart("POST", "/api/items", formular{felder: [][2]string{{"name", "Lampe"}, {"besitzer", fmt.Sprint(c.ich())}}, fotos: [][]byte{groß}})
	if res.Status != http.StatusRequestEntityTooLarge || res.errorCode() != "zu_gross" {
		t.Fatalf("Status %d, Fehlercode %q", res.Status, res.errorCode())
	}
	e.keineFotoDateien()
}

func TestOriginalUndVorschauSindAbrufbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	original := testPNG(t)
	foto := fotosVon(c.itemAnlegen("Lampe", []int64{c.ich()}, original))[0]

	res := c.get(foto["url"].(string))
	if res.Status != http.StatusOK || !bytes.Equal(res.Raw, original) || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("Original: Status %d, Typ %q, %d Bytes", res.Status, res.Header.Get("Content-Type"), len(res.Raw))
	}
	vorschau := c.get(foto["vorschauUrl"].(string))
	if vorschau.Status != http.StatusOK || vorschau.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("Vorschau: Status %d, Typ %q", vorschau.Status, vorschau.Header.Get("Content-Type"))
	}
}

func TestVorschauIstVerkleinert(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	var b bytes.Buffer
	jpeg.Encode(&b, zweifarbig(1200, 600), nil)
	foto := fotosVon(c.itemAnlegen("Lampe", []int64{c.ich()}, b.Bytes()))[0]

	img := vorschauBild(t, c, foto)
	if img.Bounds().Dx() != 512 || img.Bounds().Dy() != 256 {
		t.Fatalf("Vorschau ist %v groß, erwartet 512×256", img.Bounds().Size())
	}
}

func TestVorschauBeruecksichtigtExifAusrichtung(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	// Das Original ist 40×20, links rot, rechts blau. Ausrichtung 6 heißt z. B.: zum Anzeigen
	// um 90° im Uhrzeigersinn drehen, die rote Hälfte kommt also nach oben.
	for _, fall := range []struct {
		ausrichtung   uint16
		breite, hoehe int
		rot           string
	}{
		{1, 40, 20, "links"},
		{2, 40, 20, "rechts"},
		{3, 40, 20, "rechts"},
		{4, 40, 20, "links"},
		{5, 20, 40, "oben"},
		{6, 20, 40, "oben"},
		{7, 20, 40, "unten"},
		{8, 20, 40, "unten"},
	} {
		original := mitExifAusrichtung(t, testJPEG(t), fall.ausrichtung)
		foto := fotosVon(c.itemAnlegen("Lampe", []int64{c.ich()}, original))[0]

		img := vorschauBild(t, c, foto)
		if img.Bounds().Dx() != fall.breite || img.Bounds().Dy() != fall.hoehe {
			t.Errorf("Ausrichtung %d: Vorschau ist %v groß, erwartet %d×%d", fall.ausrichtung, img.Bounds().Size(), fall.breite, fall.hoehe)
			continue
		}
		if got := roteSeite(img); got != fall.rot {
			t.Errorf("Ausrichtung %d: Rot ist %s, erwartet %s", fall.ausrichtung, got, fall.rot)
		}
		if res := c.get(foto["url"].(string)); !bytes.Equal(res.Raw, original) {
			t.Errorf("Ausrichtung %d: das Original soll unverändert bleiben", fall.ausrichtung)
		}
	}
}

func TestOhneSitzungSindFotosNichtAbrufbar(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	foto := fotosVon(c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t)))[0]

	fremd := e.client()
	for _, url := range []string{foto["url"].(string), foto["vorschauUrl"].(string)} {
		if res := fremd.get(url); res.Status != http.StatusUnauthorized {
			t.Errorf("%s ohne Sitzung: Status %d", url, res.Status)
		}
	}
}

func vorschauBild(t *testing.T, c *apiClient, foto map[string]any) image.Image {
	t.Helper()
	res := c.get(foto["vorschauUrl"].(string))
	if res.Status != http.StatusOK {
		t.Fatalf("Vorschau: Status %d", res.Status)
	}
	img, err := jpeg.Decode(bytes.NewReader(res.Raw))
	if err != nil {
		t.Fatalf("Vorschau ist kein JPEG: %v", err)
	}
	return img
}

// roteSeite sagt, auf welcher Seite eines zweifarbigen Bildes die rote Hälfte liegt.
func roteSeite(img image.Image) string {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	links, rechts := image.Pt(w/8, h/2), image.Pt(w*7/8, h/2)
	oben, unten := image.Pt(w/2, h/8), image.Pt(w/2, h*7/8)
	for _, seite := range []struct {
		name      string
		rot, blau image.Point
	}{
		{"links", links, rechts}, {"rechts", rechts, links}, {"oben", oben, unten}, {"unten", unten, oben},
	} {
		if farbe(img, seite.rot) == "rot" && farbe(img, seite.blau) == "blau" {
			return seite.name
		}
	}
	return "unklar"
}

func farbe(img image.Image, p image.Point) string {
	r, _, b, _ := img.At(p.X, p.Y).RGBA()
	switch {
	case r > 2*b:
		return "rot"
	case b > 2*r:
		return "blau"
	}
	return "unklar"
}

func hauptfotoID(r response) any {
	for _, f := range fotosVon(r) {
		if f["hauptfoto"] == true {
			return f["id"]
		}
	}
	return nil
}

func TestFotosHinzufuegenUndHauptfotoSetzen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))

	res := c.multipart("POST", itemPfad(item)+"/fotos", formular{fotos: [][]byte{testPNG(t), testWebP(t)}})
	if res.Status != http.StatusOK || len(fotosVon(res)) != 3 {
		t.Fatalf("Status %d: %s", res.Status, res.Raw)
	}
	if hauptfotoID(res) != fotosVon(item)[0]["id"] {
		t.Fatal("neue Fotos sollen das Hauptfoto nicht ändern")
	}

	dritteFoto := fotosVon(res)[2]
	dritte := dritteFoto["id"]
	res = c.mustStatus(http.StatusOK, "PUT", itemPfad(item)+"/hauptfoto", map[string]any{"fotoId": dritte})
	if hauptfotoID(res) != dritte {
		t.Fatalf("Hauptfoto: %s", res.Raw)
	}
	liste := c.mustStatus(http.StatusOK, "GET", "/api/items", nil)
	if got := liste.Body["items"].([]any)[0].(map[string]any)["vorschauUrl"]; got != dritteFoto["vorschauUrl"] {
		t.Fatalf("die Liste soll die Vorschau des Hauptfotos zeigen, zeigt %v", got)
	}

	// Ein Foto eines anderen Items lässt sich nicht als Hauptfoto setzen.
	anderes := c.itemAnlegen("Schere", []int64{c.ich()}, testJPEG(t))
	c.mustStatus(http.StatusNotFound, "PUT", itemPfad(item)+"/hauptfoto", map[string]any{"fotoId": fotosVon(anderes)[0]["id"]})

	if res := c.multipart("POST", itemPfad(item)+"/fotos", formular{}); res.errorCode() != "foto_fehlt" {
		t.Fatalf("ohne Foto: Status %d, Fehlercode %q", res.Status, res.errorCode())
	}
	if res := c.multipart("POST", itemPfad(item)+"/fotos", formular{fotos: [][]byte{[]byte("kein Bild")}}); res.errorCode() != "falscher_dateityp" {
		t.Fatalf("falscher Typ: Status %d, Fehlercode %q", res.Status, res.errorCode())
	}
}

func TestNachDemLoeschenDesHauptfotosRuecktDasNaechsteNach(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t), testPNG(t), testWebP(t))
	fotos := fotosVon(item)

	res := c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/fotos/%v", fotos[0]["id"]), nil)
	if len(fotosVon(res)) != 2 || hauptfotoID(res) != fotos[1]["id"] {
		t.Fatalf("nach dem Löschen: %s", res.Raw)
	}
	if r := c.get(fotos[0]["url"].(string)); r.Status != http.StatusNotFound {
		t.Errorf("gelöschtes Foto: Status %d", r.Status)
	}

	// Ein anderes Foto als das Hauptfoto zu löschen, ändert das Hauptfoto nicht.
	res = c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/fotos/%v", fotos[2]["id"]), nil)
	if len(fotosVon(res)) != 1 || hauptfotoID(res) != fotos[1]["id"] {
		t.Fatalf("nach dem zweiten Löschen: %s", res.Raw)
	}
}

func TestDasLetzteFotoLaesstSichNichtLoeschen(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t))
	foto := fotosVon(item)[0]

	res := c.mustStatus(http.StatusConflict, "DELETE", fmt.Sprintf("/api/fotos/%v", foto["id"]), nil)
	if res.errorCode() != "letztes_foto" {
		t.Fatalf("Fehlercode %q", res.errorCode())
	}
	if r := c.get(foto["vorschauUrl"].(string)); r.Status != http.StatusOK {
		t.Fatalf("das Foto soll erhalten bleiben: Status %d", r.Status)
	}
}

func TestGeloeschteFotosVerschwindenAusDemFotoOrdner(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t), testPNG(t))
	c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/fotos/%v", fotosVon(item)[0]["id"]), nil)
	c.mustStatus(http.StatusNoContent, "DELETE", itemPfad(item), nil)
	e.keineFotoDateien()
}

func TestTransparenteFlaechenWerdenInDerVorschauWeiss(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	var b bytes.Buffer
	png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	foto := fotosVon(c.itemAnlegen("Aufkleber", []int64{c.ich()}, b.Bytes()))[0]

	r, g, bl, _ := vorschauBild(t, c, foto).At(5, 5).RGBA()
	if r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Fatalf("Pixel (%d, %d, %d), erwartet weiß", r>>8, g>>8, bl>>8)
	}
}

// Fotos dürfen dauerhaft im Browser-Cache bleiben. Ein neues Foto braucht deshalb eine Adresse,
// unter der nie ein anderes Bild ausgeliefert wurde, auch wenn vorher eines gelöscht wurde.
func TestNeuesFotoBekommtNachDemLoeschenEineNeueAdresse(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	item := c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t), testPNG(t))
	alt := fotosVon(item)[1]
	c.mustStatus(http.StatusOK, "DELETE", fmt.Sprintf("/api/fotos/%v", alt["id"]), nil)

	neuesBild := testWebP(t)
	res := c.multipart("POST", itemPfad(item)+"/fotos", formular{fotos: [][]byte{neuesBild}})
	if res.Status != http.StatusOK || len(fotosVon(res)) != 2 {
		t.Fatalf("Status %d: %s", res.Status, res.Raw)
	}
	neu := fotosVon(res)[1]
	if neu["id"] == alt["id"] {
		t.Errorf("die ID %v des gelöschten Fotos wurde wieder vergeben", alt["id"])
	}
	for _, feld := range []string{"url", "vorschauUrl"} {
		if neu[feld] == alt[feld] {
			t.Errorf("%s: das neue Foto hat die Adresse des gelöschten: %v", feld, neu[feld])
		}
		if r := c.get(alt[feld].(string)); r.Status != http.StatusNotFound {
			t.Errorf("%s des gelöschten Fotos: Status %d", feld, r.Status)
		}
	}
	if r := c.get(neu["url"].(string)); r.Status != http.StatusOK || !bytes.Equal(r.Raw, neuesBild) {
		t.Fatalf("neues Foto: Status %d, %d Bytes", r.Status, len(r.Raw))
	}
}

// Nur die Adresse mit dem richtigen Dateinamen liefert das Foto, sonst könnte eine alte
// Adresse nach einer Änderung der Datei ein anderes Bild zeigen.
func TestFotoMitFalschemDateinamenWirdNichtAusgeliefert(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	foto := fotosVon(c.itemAnlegen("Lampe", []int64{c.ich()}, testJPEG(t)))[0]

	for _, url := range []string{
		fmt.Sprintf("/api/fotos/%v", foto["id"]),
		fmt.Sprintf("/api/fotos/%v/vorschau", foto["id"]),
		fmt.Sprintf("/api/fotos/%v/falsch.jpg", foto["id"]),
		fmt.Sprintf("/api/fotos/%v/falsch.jpg/vorschau", foto["id"]),
	} {
		if r := c.get(url); r.Status != http.StatusNotFound && r.Status != http.StatusMethodNotAllowed {
			t.Errorf("%s: Status %d", url, r.Status)
		}
	}
}
