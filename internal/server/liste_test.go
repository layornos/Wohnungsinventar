package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"inventar/internal/server"
)

// namenInListe ruft die Item-Liste mit der Abfrage ab und liefert die Namen in ihrer Reihenfolge.
func (c *apiClient) namenInListe(abfrage string) []string {
	c.t.Helper()
	res := c.mustStatus(http.StatusOK, "GET", "/api/items"+abfrage, nil)
	var namen []string
	for _, i := range res.Body["items"].([]any) {
		namen = append(namen, i.(map[string]any)["name"].(string))
	}
	return namen
}

func (c *apiClient) itemMitVerbleib(name string, besitzer []int64, verbleib string) {
	c.t.Helper()
	item := c.itemAnlegen(name, besitzer, testJPEG(c.t))
	if verbleib != "" {
		c.mustStatus(http.StatusOK, "PUT", itemPfad(item), map[string]any{"name": name, "verbleib": verbleib})
	}
}

func TestItemListeFiltern(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	ich, kim := c.ich(), c.personAnlegen("Kim")
	c.itemMitVerbleib("Schreibtischlampe", []int64{ich}, "verkaufen")
	c.itemMitVerbleib("Lampenschirm", []int64{kim}, "")
	c.itemMitVerbleib("Äpfelkorb", []int64{ich, kim}, "behalten")
	c.itemMitVerbleib("Schere 50%_rabatt", []int64{kim}, "")

	for _, fall := range []struct {
		abfrage string
		namen   []string
	}{
		{"", []string{"Schere 50%_rabatt", "Äpfelkorb", "Lampenschirm", "Schreibtischlampe"}},
		{"?q=LAMPE", []string{"Lampenschirm", "Schreibtischlampe"}},
		{"?q=äpfel", []string{"Äpfelkorb"}},
		{"?q=50%25_", []string{"Schere 50%_rabatt"}},
		{"?q=%25", []string{"Schere 50%_rabatt"}},
		{fmt.Sprintf("?besitzer=%d", ich), []string{"Äpfelkorb", "Schreibtischlampe"}},
		{"?verbleib=offen", []string{"Schere 50%_rabatt", "Lampenschirm"}},
		{"?verbleib=behalten", []string{"Äpfelkorb"}},
		{fmt.Sprintf("?q=lampe&besitzer=%d&verbleib=offen", kim), []string{"Lampenschirm"}},
		{"?verbleib=verschenken", nil},
	} {
		if got := c.namenInListe(fall.abfrage); !slices.Equal(got, fall.namen) {
			t.Errorf("%s: %v, erwartet %v", fall.abfrage, got, fall.namen)
		}
	}
	for _, abfrage := range []string{"?verbleib=wegwerfen", "?besitzer=kim", "?seite=0", "?proSeite=1000"} {
		if res := c.do("GET", "/api/items"+abfrage, nil); res.Status != http.StatusBadRequest {
			t.Errorf("%s: Status %d", abfrage, res.Status)
		}
	}
}

func TestZuletztGeaenderteItemsStehenOben(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	alt := c.itemAnlegen("Alt", []int64{c.ich()}, testJPEG(t))
	c.itemAnlegen("Neu", []int64{c.ich()}, testJPEG(t))

	if got := c.namenInListe(""); !slices.Equal(got, []string{"Neu", "Alt"}) {
		t.Fatalf("vor der Änderung: %v", got)
	}
	c.mustStatus(http.StatusOK, "PUT", itemPfad(alt), map[string]any{"name": "Alt", "verbleib": "behalten"})
	if got := c.namenInListe(""); !slices.Equal(got, []string{"Alt", "Neu"}) {
		t.Fatalf("nach der Änderung: %v", got)
	}
}

func TestItemListeSeitenweise(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	for _, name := range []string{"A", "B", "C"} {
		c.itemAnlegen(name, []int64{c.ich()}, testJPEG(t))
	}

	erste := c.mustStatus(http.StatusOK, "GET", "/api/items?proSeite=2", nil)
	zweite := c.mustStatus(http.StatusOK, "GET", "/api/items?proSeite=2&seite=2", nil)
	if erste.Body["weitere"] != true || zweite.Body["weitere"] != false {
		t.Fatalf("weitere: %v, %v", erste.Body["weitere"], zweite.Body["weitere"])
	}
	if got := c.namenInListe("?proSeite=2"); !slices.Equal(got, []string{"C", "B"}) {
		t.Errorf("erste Seite: %v", got)
	}
	if got := c.namenInListe("?proSeite=2&seite=2"); !slices.Equal(got, []string{"A"}) {
		t.Errorf("zweite Seite: %v", got)
	}
}

func TestSortierungStimmtAuchInnerhalbEinerSekunde(t *testing.T) {
	jetzt := time.Date(2026, 5, 1, 12, 0, 5, 0, time.UTC)
	e := newTestEnvMit(t, func(d *server.Deps) { d.Jetzt = func() time.Time { return jetzt } })
	c := e.loggedIn()

	c.itemAnlegen("Volle Sekunde", []int64{c.ich()}, testJPEG(t))
	jetzt = jetzt.Add(500 * time.Millisecond)
	c.itemAnlegen("Halbe Sekunde später", []int64{c.ich()}, testJPEG(t))

	if got := c.namenInListe(""); !slices.Equal(got, []string{"Halbe Sekunde später", "Volle Sekunde"}) {
		t.Fatalf("Reihenfolge %v", got)
	}
}
