package server_test

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"rsc.io/qr"
)

// createBoxFor creates a box with a target room and returns its representation.
func (c *apiClient) createBoxFor(label string, targetRoom int64) response {
	c.t.Helper()
	return c.mustStatus(http.StatusCreated, "POST", "/api/move/boxes", map[string]any{"label": label, "targetRoomId": targetRoom})
}

func boxPath(box response) string { return fmt.Sprintf("/api/boxes/%v", box.Body["id"]) }

// nameOf returns the name of a {id, name} object, or nil.
func nameOf(v any) any {
	if n, ok := v.(map[string]any); ok {
		return n["name"]
	}
	return nil
}

// fromBox returns targetRoomFromBox from the move details of an item.
func fromBox(item response) any {
	m, _ := item.Body["move"].(map[string]any)
	return m["targetRoomFromBox"]
}

// furnishedStudy is the study of the old apartment with a desk as furniture of the storage
// location "Schreibtisch" (Schreibtisch › obere Schublade › Trenner) and the new apartment.
type furnishedStudy struct {
	moveSetup
	deskItem response
	divider  int64
}

func (c *apiClient) createFurnishedStudy() furnishedStudy {
	c.t.Helper()
	f := furnishedStudy{moveSetup: c.createMove()}
	f.deskItem = c.createItem("Schreibtisch", []int64{c.me()}, testJPEG(c.t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(f.deskItem)+"/placement", map[string]any{"roomId": f.room})
	c.mustStatus(http.StatusOK, "PUT", fmt.Sprintf("/api/storage-locations/%d/furniture", f.desk), map[string]any{"itemId": f.deskItem.Body["id"]})
	f.divider = c.createStorageLocation(map[string]any{"name": "Trenner", "parentId": f.drawer})
	return f
}

func TestBoxTargetRoomIsTheEffectiveTargetRoomOfItsItems(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box := c.createBoxFor("Küche", m.kitchen)
	if nameOf(box.Body["targetRoom"]) != "Küche" {
		t.Fatalf("box: %s", box.Raw)
	}
	pot := c.itemIn("Topf", m.drawer)
	res := c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"boxId": box.Body["id"]})
	if detail(t, res, "targetRoom") != "Küche" || fromBox(res) != true {
		t.Fatalf("effective target room from the box: %s", res.Raw)
	}
	if b := res.Body["move"].(map[string]any)["box"].(map[string]any); nameOf(b["targetRoom"]) != "Küche" {
		t.Fatalf("box summary: %s", res.Raw)
	}

	// The box counts for the overviews too.
	if got := c.namesInList("?withoutTargetRoom=1"); len(got) != 0 {
		t.Fatalf("without target room: %v", got)
	}
	if got := c.namesInList(fmt.Sprintf("?box=%v", box.Body["id"])); !slices.Equal(got, []string{"Topf"}) {
		t.Fatalf("box filter: %v", got)
	}
	overview := c.mustStatus(http.StatusOK, "GET", "/api/move/target-rooms", nil)
	for _, tr := range overview.Body["targetRooms"].([]any) {
		tr := tr.(map[string]any)
		if nameOf(tr["room"]) != "Küche" {
			continue
		}
		boxes := tr["boxes"].([]any)
		if len(boxes) != 1 || !slices.Equal(namesOf(response{Body: tr}), []string{"Topf"}) {
			t.Fatalf("overview of the kitchen: %v", tr)
		}
		if nameOf(boxes[0].(map[string]any)["targetRoom"]) != "Küche" {
			t.Fatalf("box in the overview: %v", tr)
		}
	}

	// An own target room wins; removing it brings back the one of the box.
	res = c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"targetRoomId": m.office})
	if detail(t, res, "targetRoom") != "Büro" || fromBox(res) != false {
		t.Fatalf("own target room: %s", res.Raw)
	}
	list := c.mustStatus(http.StatusOK, "GET", boxPath(box), nil)
	if got := lines(t, list.Body["items"].([]any)); !slices.Equal(got, []string{"Topf 1 Arbeitszimmer→Büro"}) {
		t.Fatalf("packing list: %v", got)
	}
	res = c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"targetRoomId": nil})
	if detail(t, res, "targetRoom") != "Küche" || fromBox(res) != true {
		t.Fatalf("own target room removed: %s", res.Raw)
	}
}

func TestChangeLabelAndTargetRoomOfABox(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box := c.createBoxFor("", m.kitchen)

	res := c.mustStatus(http.StatusOK, "PUT", boxPath(box), map[string]any{"label": "Box A", "targetRoomId": m.office})
	if res.Body["label"] != "Box A" || nameOf(res.Body["targetRoom"]) != "Büro" {
		t.Fatalf("changed: %s", res.Raw)
	}
	// Without the field the target room stays (older clients only send the label).
	res = c.mustStatus(http.StatusOK, "PUT", boxPath(box), map[string]any{"label": "Box B"})
	if res.Body["label"] != "Box B" || nameOf(res.Body["targetRoom"]) != "Büro" {
		t.Fatalf("label only: %s", res.Raw)
	}
	res = c.mustStatus(http.StatusOK, "PUT", boxPath(box), map[string]any{"label": "Box B", "targetRoomId": nil})
	if res.Body["targetRoom"] != nil {
		t.Fatalf("target room removed: %s", res.Raw)
	}
	if b := c.createBoxFor("ohne", 0); b.Body["targetRoom"] != nil {
		t.Fatalf("box without target room: %s", b.Raw)
	}
}

func TestBoxTargetRoomIsChecked(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box := c.createBoxFor("Küche", m.kitchen)

	for _, tc := range []struct {
		method, path string
		body         map[string]any
		code         string
	}{
		{"POST", "/api/move/boxes", map[string]any{"label": "x", "targetRoomId": m.room}, "target_room_not_in_new_apartment"},
		{"POST", "/api/move/boxes", map[string]any{"label": "x", "targetRoomId": 999}, "unknown_place"},
		{"PUT", boxPath(box), map[string]any{"label": "x", "targetRoomId": m.room}, "target_room_not_in_new_apartment"},
		{"PUT", boxPath(box), map[string]any{"label": "x", "targetRoomId": 999}, "unknown_place"},
		{"PUT", boxPath(box), map[string]any{"label": "x", "targetRoomId": "Küche"}, "invalid_request"},
	} {
		if r := c.do(tc.method, tc.path, tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%s %s %v: status %d, error code %q", tc.method, tc.path, tc.body, r.Status, r.errorCode())
		}
	}
	if res := c.mustStatus(http.StatusOK, "GET", boxPath(box), nil); nameOf(res.Body["box"].(map[string]any)["targetRoom"]) != "Küche" {
		t.Fatalf("rejected changes should leave the box as it was: %s", res.Raw)
	}

	// A room that is the target room of a box cannot be deleted.
	r := c.do("DELETE", fmt.Sprintf("/api/rooms/%d", m.kitchen), nil)
	if r.Status != http.StatusConflict || r.errorCode() != "room_in_move" {
		t.Fatalf("delete target room of a box: status %d, error code %q", r.Status, r.errorCode())
	}
}

func TestPackingAStorageLocationPacksEverythingInsideButNotTheFurniture(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	f := c.createFurnishedStudy()
	box := c.createBoxFor("Büro", f.office)
	c.itemIn("Stift", f.drawer)
	c.itemIn("Büroklammer", f.divider)
	c.itemIn("Lampe", f.desk)
	shelf := c.createStorageLocation(map[string]any{"name": "Regal", "roomId": f.room})
	c.itemIn("Buch", shelf)

	pack := func(storageLocation int64) response {
		t.Helper()
		return c.mustStatus(http.StatusOK, "POST", boxPath(box)+"/pack", map[string]any{"storageLocationId": storageLocation})
	}
	res := pack(f.drawer)
	if res.Body["count"] != float64(2) || res.Body["box"].(map[string]any)["count"] != float64(2) {
		t.Fatalf("pack the drawer: %s", res.Raw)
	}
	res = pack(f.desk)
	if res.Body["count"] != float64(1) || res.Body["box"].(map[string]any)["count"] != float64(3) {
		t.Fatalf("pack the desk: %s", res.Raw)
	}
	if got := c.namesInList(fmt.Sprintf("?box=%v", box.Body["id"])); !slices.Equal(sorted(got), []string{"Büroklammer", "Lampe", "Stift"}) {
		t.Fatalf("in the box: %v", got)
	}
	if got := c.get(itemPath(f.deskItem)); got.Body["move"] != nil {
		t.Fatalf("the desk itself should not be packed: %s", got.Raw)
	}
	// Packing again packs nothing new.
	if res := pack(f.drawer); res.Body["count"] != float64(0) {
		t.Fatalf("packed again: %s", res.Raw)
	}

	r := c.do("POST", boxPath(box)+"/pack", map[string]any{"storageLocationId": 999})
	if r.Status != http.StatusBadRequest || r.errorCode() != "unknown_place" {
		t.Fatalf("unknown storage location: status %d, error code %q", r.Status, r.errorCode())
	}
	c.mustStatus(http.StatusNotFound, "POST", "/api/boxes/999/pack", map[string]any{"storageLocationId": f.drawer})

	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{"confirmed": true})
	r = c.do("POST", boxPath(box)+"/pack", map[string]any{"storageLocationId": shelf})
	if r.Status != http.StatusConflict || r.errorCode() != "no_move" {
		t.Fatalf("after completion: status %d, error code %q", r.Status, r.errorCode())
	}
	// With a new move the box of the completed one stays unchangeable.
	c.mustStatus(http.StatusCreated, "POST", "/api/move", map[string]any{"oldApartmentId": f.newApartment, "newApartmentId": f.apartment})
	r = c.do("POST", boxPath(box)+"/pack", map[string]any{"storageLocationId": shelf})
	if r.Status != http.StatusConflict || r.errorCode() != "move_completed" {
		t.Fatalf("box of a completed move: status %d, error code %q", r.Status, r.errorCode())
	}
}

func sorted(s []string) []string {
	slices.Sort(s)
	return s
}

func TestFurnitureTakesItsStorageLocationsAlongOnCompletion(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	f := c.createFurnishedStudy()
	newStudy := c.createRoom(f.newApartment, "Arbeitszimmer")
	c.mustStatus(http.StatusOK, "PATCH", itemPath(f.deskItem)+"/move", map[string]any{"targetRoomId": newStudy})
	pen := c.itemIn("Stift", f.drawer)
	cup := c.itemIn("Tasse", f.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(cup)+"/move", map[string]any{"targetRoomId": f.kitchen})

	if got := c.namesInList("?withoutTargetRoom=1"); len(got) != 0 {
		t.Fatalf("contents of the desk should not count as without target room: %v", got)
	}

	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{})
	for _, tc := range []struct {
		item response
		want string
	}{
		{f.deskItem, "Neue Wohnung › Arbeitszimmer"},
		{pen, "Neue Wohnung › Arbeitszimmer › Schreibtisch › obere Schublade"},
		{cup, "Neue Wohnung › Küche"},
	} {
		if got := pathText(t, c.get(itemPath(tc.item)).Body["placement"]); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.item.Body["name"], got, tc.want)
		}
	}
	l := c.mustStatus(http.StatusOK, "GET", fmt.Sprintf("/api/storage-locations/%d", f.divider), nil)
	if got := pathText(t, l.Body["path"]); got != "Neue Wohnung › Arbeitszimmer › Schreibtisch › obere Schublade › Trenner" {
		t.Fatalf("nested storage location: %q", got)
	}
}

func TestUnpackingAWholeBoxPlacesEverythingForGood(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	cupboard := c.createStorageLocation(map[string]any{"name": "Oberschrank", "roomId": m.kitchen})
	box := c.createBoxFor("Küche", m.kitchen)
	pot := c.itemIn("Topf", m.drawer)
	cup := c.itemIn("Tasse", m.desk)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"boxId": box.Body["id"]})
	c.mustStatus(http.StatusOK, "PATCH", itemPath(cup)+"/move", map[string]any{"boxId": box.Body["id"], "targetRoomId": m.office})

	list := c.mustStatus(http.StatusOK, "GET", boxPath(box), nil)
	var previous []string
	for _, i := range list.Body["items"].([]any) {
		i := i.(map[string]any)
		previous = append(previous, fmt.Sprintf("%s: %v", i["name"], i["previousPlace"]))
	}
	if want := []string{"Tasse: Arbeitszimmer › Schreibtisch", "Topf: Arbeitszimmer › Schreibtisch › obere Schublade"}; !slices.Equal(sorted(previous), want) {
		t.Fatalf("previous places: %v", previous)
	}

	res := c.mustStatus(http.StatusOK, "POST", boxPath(box)+"/unpack", map[string]any{"itemIds": nil, "storageLocationId": cupboard, "roomId": nil})
	if res.Body["count"] != float64(2) || res.Body["box"].(map[string]any)["count"] != float64(0) {
		t.Fatalf("unpack: %s", res.Raw)
	}
	check := func(when string) {
		t.Helper()
		for _, item := range []response{pot, cup} {
			got := c.get(itemPath(item))
			if p := pathText(t, got.Body["placement"]); p != "Neue Wohnung › Küche › Oberschrank" {
				t.Errorf("%s %s: %q", when, item.Body["name"], p)
			}
			if when == "unpacked" && (detail(t, got, "box") != nil || detail(t, got, "targetRoom") != "Küche" || fromBox(got) != false) {
				t.Errorf("%s %s: %s", when, item.Body["name"], got.Raw)
			}
		}
	}
	check("unpacked")
	if items := c.mustStatus(http.StatusOK, "GET", boxPath(box), nil).Body["items"].([]any); len(items) != 0 {
		t.Fatalf("box should be empty: %v", items)
	}

	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{})
	check("completed")
}

func TestUnpackingSingleItemsAndFurnitureTakesItsContentsAlong(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	f := c.createFurnishedStudy()
	box := c.createBoxFor("Büro", f.office)
	punch := c.itemIn("Locher", f.desk)
	c.mustStatus(http.StatusNoContent, "POST", "/api/move/bulk", map[string]any{"itemIds": []any{f.deskItem.Body["id"], punch.Body["id"]}, "boxId": box.Body["id"]})
	pen := c.itemIn("Stift", f.drawer)

	res := c.mustStatus(http.StatusOK, "POST", boxPath(box)+"/unpack", map[string]any{"itemIds": []any{f.deskItem.Body["id"]}, "roomId": f.office})
	if res.Body["count"] != float64(1) || res.Body["box"].(map[string]any)["count"] != float64(1) {
		t.Fatalf("unpack the desk: %s", res.Raw)
	}
	desk := c.get(itemPath(f.deskItem))
	if p := pathText(t, desk.Body["placement"]); p != "Neue Wohnung › Büro" || detail(t, desk, "targetRoom") != "Büro" {
		t.Fatalf("desk: %s", desk.Raw)
	}
	if p := pathText(t, c.get(itemPath(pen)).Body["placement"]); p != "Neue Wohnung › Büro › Schreibtisch › obere Schublade" {
		t.Fatalf("pen in the drawer: %q", p)
	}
	// The punch was packed out of the desk and is still in the box.
	if got := c.get(itemPath(punch)); detail(t, got, "box") != 1 {
		t.Fatalf("punch: %s", got.Raw)
	}
}

func TestUnpackingIsChecked(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()
	box := c.createBoxFor("Küche", m.kitchen)
	other := c.createBoxFor("Büro", m.office)
	pot := c.itemIn("Topf", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(pot)+"/move", map[string]any{"boxId": box.Body["id"]})
	elsewhere := c.itemIn("Locher", m.drawer)
	c.mustStatus(http.StatusOK, "PATCH", itemPath(elsewhere)+"/move", map[string]any{"boxId": other.Body["id"]})
	loose := c.itemIn("Tacker", m.drawer)

	for _, tc := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"itemIds": nil, "storageLocationId": m.drawer}, "place_not_in_new_apartment"},
		{map[string]any{"itemIds": nil, "roomId": m.room}, "place_not_in_new_apartment"},
		{map[string]any{"itemIds": nil}, "invalid_place"},
		{map[string]any{"itemIds": nil, "storageLocationId": m.drawer, "roomId": m.kitchen}, "invalid_place"},
		{map[string]any{"itemIds": nil, "roomId": 999}, "unknown_place"},
		{map[string]any{"itemIds": []any{pot.Body["id"], elsewhere.Body["id"]}, "roomId": m.kitchen}, "unknown_item"},
		{map[string]any{"itemIds": []any{pot.Body["id"], loose.Body["id"]}, "roomId": m.kitchen}, "unknown_item"},
		{map[string]any{"itemIds": []any{999}, "roomId": m.kitchen}, "unknown_item"},
	} {
		if r := c.do("POST", boxPath(box)+"/unpack", tc.body); r.Status != http.StatusBadRequest || r.errorCode() != tc.code {
			t.Errorf("%v: status %d, error code %q", tc.body, r.Status, r.errorCode())
		}
	}
	// Rejected requests change nothing.
	got := c.get(itemPath(pot))
	if detail(t, got, "box") != 1 || pathText(t, got.Body["placement"]) != "Altbau › Arbeitszimmer › Schreibtisch › obere Schublade" {
		t.Fatalf("after rejected requests: %s", got.Raw)
	}
	c.mustStatus(http.StatusNotFound, "POST", "/api/boxes/999/unpack", map[string]any{"itemIds": nil, "roomId": m.kitchen})

	c.mustStatus(http.StatusOK, "POST", "/api/move/completion", map[string]any{"confirmed": true})
	r := c.do("POST", boxPath(box)+"/unpack", map[string]any{"itemIds": nil, "roomId": m.kitchen})
	if r.Status != http.StatusConflict || r.errorCode() != "no_move" {
		t.Fatalf("after completion: status %d, error code %q", r.Status, r.errorCode())
	}
}

// svgModule matches one run of dark modules in the QR code SVG: "M<x> <y>h<length>".
var svgModule = regexp.MustCompile(`M(\d+) (\d+)h(\d+)`)

// qrContentMatches checks that the SVG shows exactly the QR code of want (with a quiet zone of 4).
func qrContentMatches(t *testing.T, svg []byte, want string) bool {
	t.Helper()
	code, err := qr.Encode(want, qr.M)
	if err != nil {
		t.Fatal(err)
	}
	dark := map[[2]int]bool{}
	for _, m := range svgModule.FindAllSubmatch(svg, -1) {
		x, _ := strconv.Atoi(string(m[1]))
		y, _ := strconv.Atoi(string(m[2]))
		n, _ := strconv.Atoi(string(m[3]))
		for i := range n {
			dark[[2]int{x + i - 4, y - 4}] = true
		}
	}
	count := 0
	for y := range code.Size {
		for x := range code.Size {
			if code.Black(x, y) {
				count++
				if !dark[[2]int{x, y}] {
					return false
				}
			}
		}
	}
	return count == len(dark)
}

// boxQR fetches the QR code of a box under the given host, with the session cookie by hand
// (the cookie jar does not send Secure cookies over plain HTTP).
func boxQR(t *testing.T, e *testEnv, session *http.Cookie, box int64) response {
	t.Helper()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/boxes/%d/qr", e.srv.URL, box), nil)
	req.Host = "inventar.example.duckdns.org"
	req.AddCookie(session)
	return e.client().send(req)
}

func TestQRCodeOfABoxLinksToUnpacking(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    func(*testing.T) *testEnv
		scheme string
	}{
		{"without proxy", newTestEnv, "http"},
		{"behind proxy", newTestEnvBehindProxy, "https"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.env(t)
			session := sessionCookieOf(t, loginResponse(t, e))
			c := e.client()
			c.http.Jar.SetCookies(mustURL(t, e.srv.URL), []*http.Cookie{{Name: session.Name, Value: session.Value}})
			m := c.createMove()
			box := c.createBoxFor("Küche", m.kitchen)
			id := int64(box.Body["id"].(float64))

			res := boxQR(t, e, session, id)
			if res.Status != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "image/svg+xml") {
				t.Fatalf("status %d, type %q: %s", res.Status, res.Header.Get("Content-Type"), res.Raw)
			}
			want := fmt.Sprintf("%s://inventar.example.duckdns.org/boxen/%d/auspacken", tc.scheme, id)
			if !strings.Contains(string(res.Raw), "<svg") || !qrContentMatches(t, res.Raw, want) {
				t.Fatalf("QR code does not encode %q:\n%s", want, res.Raw)
			}
			if res := boxQR(t, e, session, 999); res.Status != http.StatusNotFound {
				t.Fatalf("unknown box: status %d", res.Status)
			}
		})
	}
}

func TestItemListEntriesNameTheStorageLocation(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	s := c.createStudy()
	c.itemIn("Schere", s.drawer)
	lamp := c.createItem("Lampe", []int64{c.me()}, testJPEG(t))
	c.mustStatus(http.StatusOK, "PUT", itemPath(lamp)+"/placement", map[string]any{"roomId": s.room})
	c.createItem("Kiste", []int64{c.me()}, testJPEG(t))

	var got []string
	for _, i := range c.mustStatus(http.StatusOK, "GET", "/api/items", nil).Body["items"].([]any) {
		i := i.(map[string]any)
		got = append(got, fmt.Sprintf("%s %v", i["name"], i["storageLocationId"]))
	}
	want := []string{"Kiste <nil>", "Lampe <nil>", fmt.Sprintf("Schere %d", s.drawer)}
	if !slices.Equal(sorted(got), want) {
		t.Fatalf("storage locations: %v, want %v", got, want)
	}
}

func TestPlacesNameTheirApartment(t *testing.T) {
	e := newTestEnv(t)
	c := e.loggedIn()
	m := c.createMove()

	got := map[string]any{}
	for _, p := range c.mustStatus(http.StatusOK, "GET", "/api/places", nil).Body["places"].([]any) {
		p := p.(map[string]any)
		got[p["path"].(string)] = p["apartmentId"]
	}
	for path, want := range map[string]int64{
		"Altbau › Arbeitszimmer":                                  m.apartment,
		"Altbau › Arbeitszimmer › Schreibtisch › obere Schublade": m.apartment,
		"Neue Wohnung › Küche":                                    m.newApartment,
	} {
		if got[path] != float64(want) {
			t.Errorf("%s: apartment %v, want %d", path, got[path], want)
		}
	}
}
