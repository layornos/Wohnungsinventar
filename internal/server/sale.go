package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"inventar/internal/euro"
	"inventar/internal/store"
)

const dateLayout = "2006-01-02"

// today is the current date in the household's time zone.
func (s *server) today() string { return store.Day(s.Now()) }

const (
	msgNoSale         = "Einen Angebotspreis gibt es nur bei Verkaufen und Später verkaufen"
	msgInvalidDate    = "Ungültiges Datum"
	msgDateInFuture   = "Das Datum liegt in der Zukunft"
	msgNoAskingPrice  = "Erst einen Preis eintragen"
	msgNoListing      = "Ein Inserat gibt es nur bei Verkaufen, Später verkaufen und Verschenken"
	maxTitleLength    = 65
	settingTemplate   = "listing_template"
	settingDisclaimer = "warranty_disclaimer"
)

// The defaults until the household changes them (spec 06).
const (
	defaultTemplate   = "{beschreibung}\n\nPreis: {preis}\n\nAbholung oder Versand nach Absprache."
	defaultDisclaimer = "Privatverkauf. Der Verkauf erfolgt unter Ausschluss jeglicher Gewährleistung, Garantie und Rücknahme."
)

// readCents reads an amount in cents (null allowed) and answers errors itself.
func readCents(w http.ResponseWriter, raw json.RawMessage, target **int64, code, message string) bool {
	var v *int64
	if err := json.Unmarshal(raw, &v); err != nil || (v != nil && *v < 0) {
		writeError(w, http.StatusBadRequest, code, message)
		return false
	}
	*target = v
	return true
}

// readDate checks a date (YYYY-MM-DD) and answers errors itself.
func readDate(w http.ResponseWriter, value string) bool {
	if _, err := time.Parse(dateLayout, value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", msgInvalidDate)
		return false
	}
	return true
}

type listingSettingsJSON struct {
	Template           string `json:"template"`
	WarrantyDisclaimer string `json:"warrantyDisclaimer"`
}

func (s *server) loadListingSettings(r *http.Request) (listingSettingsJSON, error) {
	values, err := s.Store.Settings(r.Context(), settingTemplate, settingDisclaimer)
	if err != nil {
		return listingSettingsJSON{}, err
	}
	out := listingSettingsJSON{Template: defaultTemplate, WarrantyDisclaimer: defaultDisclaimer}
	if v, ok := values[settingTemplate]; ok {
		out.Template = v
	}
	if v, ok := values[settingDisclaimer]; ok {
		out.WarrantyDisclaimer = v
	}
	return out, nil
}

func (s *server) listingSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.loadListingSettings(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// saveListingSettings stores the template and the warranty disclaimer; the disclaimer must not be empty.
func (s *server) saveListingSettings(w http.ResponseWriter, r *http.Request) {
	var req listingSettingsJSON
	if !readJSON(w, r, &req) {
		return
	}
	req.Template = strings.TrimSpace(req.Template)
	req.WarrantyDisclaimer = strings.TrimSpace(req.WarrantyDisclaimer)
	if req.WarrantyDisclaimer == "" {
		writeError(w, http.StatusBadRequest, "warranty_disclaimer_missing", "Der Gewährleistungsausschluss darf nicht leer sein")
		return
	}
	if err := s.Store.SaveSettings(r.Context(), map[string]string{settingTemplate: req.Template, settingDisclaimer: req.WarrantyDisclaimer}); err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// listing builds the listing text of an item from the template: the title is the name, cut to the limit of the
// strictest platform (then the full name opens the text); paragraphs left empty, e.g. without description, are
// dropped, and the warranty disclaimer always comes last.
func listing(item store.Item, settings listingSettingsJSON) (title, text string) {
	price := "zu verschenken"
	if item.Disposition != store.GiveAway {
		price = euro.Format(*item.AskingPrice)
	}
	var paragraphs []string
	title = item.Name
	if name := []rune(item.Name); len(name) > maxTitleLength {
		title = strings.TrimSpace(string(name[:maxTitleLength-1])) + "…"
		paragraphs = append(paragraphs, item.Name)
	}
	filled := strings.NewReplacer("{name}", item.Name, "{beschreibung}", item.Description, "{preis}", price).
		Replace(strings.ReplaceAll(settings.Template, "\r\n", "\n"))
	for _, p := range strings.Split(filled, "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	paragraphs = append(paragraphs, settings.WarrantyDisclaimer)
	return title, strings.Join(paragraphs, "\n\n")
}

func (s *server) listingText(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	item, err := s.Store.GetItem(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	switch {
	case item.Disposition != store.GiveAway && !item.Disposition.ForSale():
		writeError(w, http.StatusConflict, "no_sale", msgNoListing)
		return
	case item.Disposition != store.GiveAway && item.AskingPrice == nil:
		writeError(w, http.StatusConflict, "no_asking_price", msgNoAskingPrice)
		return
	}
	settings, err := s.loadListingSettings(r)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	title, text := listing(item, settings)
	writeJSON(w, http.StatusOK, map[string]string{"title": title, "text": text})
}

// handOver marks an item as sold or given away, or corrects that in the archive:
// {"outcome": "sold"|"givenAway", "saleProceeds": cents (sold only), "date"?: "YYYY-MM-DD", default today}.
func (s *server) handOver(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Outcome      string          `json:"outcome"`
		SaleProceeds json.RawMessage `json:"saleProceeds"`
		Date         string          `json:"date"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	h := store.Handover{Date: req.Date}
	switch req.Outcome {
	case "sold":
		h.Disposition = store.Sell
		var proceeds *int64
		if req.SaleProceeds != nil && !readCents(w, req.SaleProceeds, &proceeds, "invalid_sale_proceeds", "Ungültiger Verkaufserlös") {
			return
		}
		if proceeds == nil {
			writeError(w, http.StatusBadRequest, "sale_proceeds_missing", "Bitte den Verkaufserlös angeben")
			return
		}
		h.Proceeds = *proceeds
	case "givenAway":
		h.Disposition = store.GiveAway
	default:
		writeError(w, http.StatusBadRequest, "invalid_outcome", "Bitte verkauft oder verschenkt angeben")
		return
	}
	today := s.today()
	if h.Date == "" {
		h.Date = today
	} else if !readDate(w, h.Date) {
		return
	}
	if h.Date > today {
		writeError(w, http.StatusBadRequest, "date_in_future", msgDateInFuture)
		return
	}
	item, err := s.Store.HandOver(r.Context(), id, h, s.Now())
	s.writeItem(w, r, item, err)
}

func (s *server) takeBack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	item, err := s.Store.TakeBack(r.Context(), id, s.Now())
	s.writeItem(w, r, item, err)
}

// readPeriod reads from and to (YYYY-MM-DD, both optional) and answers errors itself.
func readPeriod(w http.ResponseWriter, r *http.Request) (from, to string, ok bool) {
	from, to = r.URL.Query().Get("from"), r.URL.Query().Get("to")
	for _, d := range []string{from, to} {
		if d != "" && !readDate(w, d) {
			return "", "", false
		}
	}
	return from, to, true
}

// listArchive lists sold and given away items, most recently handed over first, with the filters of the item list
// plus outcome ("sold" or "givenAway") and the period from/to.
func (s *server) listArchive(w http.ResponseWriter, r *http.Request) {
	filter, ok := readItemFilter(w, r)
	if !ok {
		return
	}
	filter.Archived = true
	if filter.HandedOverFrom, filter.HandedOverTo, ok = readPeriod(w, r); !ok {
		return
	}
	switch r.URL.Query().Get("outcome") {
	case "":
	case "sold":
		d := store.Sell
		filter.Disposition = &d
	case "givenAway":
		d := store.GiveAway
		filter.Disposition = &d
	default:
		writeError(w, http.StatusBadRequest, "invalid_outcome", "Bitte verkauft oder verschenkt angeben")
		return
	}
	s.writeItemList(w, r, filter)
}

type proceedsEntryJSON struct {
	itemListJSON
	// Share is the part of the proceeds counting for the owner in the filter, otherwise the whole proceeds.
	Share int64 `json:"share"`
}

// proceeds sums up the archive in the period from/to, optionally for one owner, whose items count with an equal
// share each.
func (s *server) proceeds(w http.ResponseWriter, r *http.Request) {
	from, to, ok := readPeriod(w, r)
	if !ok {
		return
	}
	filter, ok := readItemFilter(w, r)
	if !ok {
		return
	}
	p, err := s.Store.ProceedsSummary(r.Context(), from, to, filter.OwnerID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	entries := make([]proceedsEntryJSON, 0, len(p.Entries))
	for _, e := range p.Entries {
		entries = append(entries, proceedsEntryJSON{itemListJSON: toItemListJSON([]store.ItemListEntry{e.ItemListEntry})[0], Share: e.Share})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total": p.Total, "soldCount": p.SoldCount, "givenAwayCount": p.GivenAwayCount, "items": entries,
	})
}
