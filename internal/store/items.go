package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrLetztesFoto = errors.New("letztes Foto eines Items")

// Verbleib sagt, was mit einem Item geschehen soll; Offen heißt, das ist noch nicht festgelegt.
type Verbleib string

const (
	Offen            Verbleib = ""
	Behalten         Verbleib = "behalten"
	Verkaufen        Verbleib = "verkaufen"
	SpaeterVerkaufen Verbleib = "spaeter_verkaufen"
	Verschenken      Verbleib = "verschenken"
)

func (v Verbleib) Gueltig() bool {
	switch v {
	case Offen, Behalten, Verkaufen, SpaeterVerkaufen, Verschenken:
		return true
	}
	return false
}

type Item struct {
	ID           int64
	Name         string
	Beschreibung string
	Verbleib     Verbleib
	ErstelltAm   time.Time
	GeaendertAm  time.Time
	Besitzer     []Person
	// Fotos in ihrer Reihenfolge; genau eines ist Hauptfoto.
	Fotos []Foto
	// Platzierung ist nil, solange das Item nicht platziert ist.
	Platzierung *Pfad
	// MoebelLagerorte sind die Lagerorte, die zu diesem Item als Möbelstück gehören.
	MoebelLagerorte []Benannt
}

type Foto struct {
	ID        int64
	ItemID    int64
	Datei     string
	Typ       string
	Hauptfoto bool
}

// NeuesFoto ist eine bereits gespeicherte Datei, die einem Item zugeordnet wird.
type NeuesFoto struct {
	Datei string
	Typ   string
}

type NeuesItem struct {
	Name         string
	Beschreibung string
	Verbleib     Verbleib
	BesitzerIDs  []int64
	Fotos        []NeuesFoto
	Platz        Platz
}

// abfrager ist, was *sql.DB und *sql.Tx gemeinsam haben.
type abfrager interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// verbleibAus liest den gespeicherten Verbleib; NULL heißt offen.
func verbleibAus(n sql.NullString) Verbleib { return Verbleib(n.String) }

func verbleibWert(v Verbleib) any {
	if v == Offen {
		return nil
	}
	return string(v)
}

// ItemAnlegen legt ein Item mit seinen Besitzern und Fotos an; das erste Foto wird Hauptfoto.
// Der Aufrufer stellt sicher, dass es mindestens einen Besitzer und ein Foto gibt.
func (s *Store) ItemAnlegen(ctx context.Context, neu NeuesItem, jetzt time.Time) (Item, error) {
	var item Item
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO item (name, name_klein, beschreibung, verbleib, erstellt_am, geaendert_am)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
			neu.Name, strings.ToLower(neu.Name), neu.Beschreibung, verbleibWert(neu.Verbleib), zeitText(jetzt), zeitText(jetzt)).Scan(&id)
		if err != nil {
			return err
		}
		for _, p := range neu.BesitzerIDs {
			if err := besitzEintragen(ctx, tx, id, p); err != nil {
				return err
			}
		}
		if err := fotosEintragen(ctx, tx, id, neu.Fotos); err != nil {
			return err
		}
		if err := platzSetzen(ctx, tx, id, neu.Platz); err != nil {
			return err
		}
		item, err = itemLesen(ctx, tx, id)
		return err
	})
	return item, err
}

func besitzEintragen(ctx context.Context, q abfrager, itemID, personID int64) error {
	var existiert bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM person WHERE id = ?)`, personID).Scan(&existiert); err != nil {
		return err
	}
	if !existiert {
		return ErrUnbekanntePerson
	}
	_, err := q.ExecContext(ctx, `INSERT INTO besitz (item_id, person_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, itemID, personID)
	return err
}

// fotosEintragen hängt Fotos hinten an; hat das Item noch kein Hauptfoto, wird das erste neue es.
func fotosEintragen(ctx context.Context, q abfrager, itemID int64, fotos []NeuesFoto) error {
	for _, f := range fotos {
		_, err := q.ExecContext(ctx, `
			INSERT INTO foto (item_id, datei, typ, hauptfoto, reihenfolge)
			SELECT ?, ?, ?,
				CASE WHEN EXISTS (SELECT 1 FROM foto WHERE item_id = ? AND hauptfoto = 1) THEN 0 ELSE 1 END,
				COALESCE((SELECT MAX(reihenfolge) FROM foto WHERE item_id = ?), 0) + 1`,
			itemID, f.Datei, f.Typ, itemID, itemID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ItemAbrufen(ctx context.Context, id int64) (Item, error) {
	return itemLesen(ctx, s.db, id)
}

func itemLesen(ctx context.Context, q abfrager, id int64) (Item, error) {
	var (
		item                    Item
		verbleib                sql.NullString
		erstelltAm, geaendertAm string
		lagerortID, raumID      sql.NullInt64
	)
	err := q.QueryRowContext(ctx, `
		SELECT id, name, beschreibung, verbleib, erstellt_am, geaendert_am, lagerort_id, raum_id FROM item WHERE id = ?`, id).
		Scan(&item.ID, &item.Name, &item.Beschreibung, &verbleib, &erstelltAm, &geaendertAm, &lagerortID, &raumID)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNichtGefunden
	}
	if err != nil {
		return Item{}, err
	}
	item.Verbleib = verbleibAus(verbleib)
	if item.ErstelltAm, err = zeitLesen(erstelltAm); err != nil {
		return Item{}, err
	}
	if item.GeaendertAm, err = zeitLesen(geaendertAm); err != nil {
		return Item{}, err
	}
	if item.Besitzer, err = besitzerLesen(ctx, q, id); err != nil {
		return Item{}, err
	}
	if item.Platzierung, err = platzierungLesen(ctx, q, lagerortID.Int64, raumID.Int64); err != nil {
		return Item{}, err
	}
	if item.MoebelLagerorte, err = benannteLesen(q.QueryContext(ctx,
		`SELECT id, name FROM lagerort WHERE moebel_item_id = ? ORDER BY name, id`, id)); err != nil {
		return Item{}, err
	}
	item.Fotos, err = fotosLesen(ctx, q, id)
	return item, err
}

func platzierungLesen(ctx context.Context, q abfrager, lagerortID, raumID int64) (*Pfad, error) {
	var (
		p   Pfad
		err error
	)
	switch {
	case lagerortID != 0:
		p, err = lagerortPfad(ctx, q, lagerortID)
	case raumID != 0:
		p, err = raumPfad(ctx, q, raumID)
	default:
		return nil, nil
	}
	return &p, err
}

func besitzerLesen(ctx context.Context, q abfrager, itemID int64) ([]Person, error) {
	return personenLesen(q.QueryContext(ctx,
		personSelect+` JOIN besitz b ON b.person_id = p.id WHERE b.item_id = ? ORDER BY p.name, p.id`, itemID))
}

const fotoSelect = `SELECT id, item_id, datei, typ, hauptfoto = 1 FROM foto`

// zeile ist, was *sql.Row und *sql.Rows zum Auslesen gemeinsam haben.
type zeile interface{ Scan(dest ...any) error }

func fotoScannen(z zeile) (Foto, error) {
	var f Foto
	err := z.Scan(&f.ID, &f.ItemID, &f.Datei, &f.Typ, &f.Hauptfoto)
	return f, err
}

func fotosLesen(ctx context.Context, q abfrager, itemID int64) ([]Foto, error) {
	rows, err := q.QueryContext(ctx, fotoSelect+` WHERE item_id = ? ORDER BY reihenfolge, id`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fotos []Foto
	for rows.Next() {
		f, err := fotoScannen(rows)
		if err != nil {
			return nil, err
		}
		fotos = append(fotos, f)
	}
	return fotos, rows.Err()
}

// ItemListenEintrag ist ein Item in der Liste, mit seinem Hauptfoto für die Vorschau.
type ItemListenEintrag struct {
	ID          int64
	Name        string
	Verbleib    Verbleib
	HauptfotoID int64
}

type ItemFilter struct {
	// Text muss im Namen vorkommen, Groß- und Kleinschreibung egal.
	Text       string
	BesitzerID int64
	// Verbleib filtert, wenn gesetzt; ein Zeiger auf Offen liefert die Items ohne Verbleib.
	Verbleib *Verbleib
	// RaumID und LagerortID umfassen auch Items in verschachtelten Lagerorten.
	RaumID     int64
	LagerortID int64
	// MoebelID liefert die Items in den Lagerorten dieses Möbelstücks.
	MoebelID     int64
	OhneLagerort bool
	Anzahl       int
	Versatz      int
}

// teilbaum ist ein rekursiver Teilausdruck, der ab den Lagerorten, die start erfüllen,
// alle Unterlagerorte sammelt.
func teilbaum(name, start string) string {
	return name + ` (id) AS (
		SELECT id FROM lagerort WHERE ` + start + `
		UNION ALL
		SELECT l.id FROM lagerort l JOIN ` + name + ` t ON l.eltern_id = t.id)`
}

// ItemsAuflisten liefert die passenden Items, zuletzt geänderte zuerst, und ob es weitere gibt.
func (s *Store) ItemsAuflisten(ctx context.Context, f ItemFilter) ([]ItemListenEintrag, bool, error) {
	var (
		ctes, bedingungen []string
		cteArgs, args     []any
	)
	if f.Text != "" {
		bedingungen = append(bedingungen, `i.name_klein LIKE ? ESCAPE '\'`)
		args = append(args, "%"+likeMaskieren(strings.ToLower(f.Text))+"%")
	}
	if f.BesitzerID != 0 {
		bedingungen = append(bedingungen, `EXISTS (SELECT 1 FROM besitz b WHERE b.item_id = i.id AND b.person_id = ?)`)
		args = append(args, f.BesitzerID)
	}
	if f.Verbleib != nil {
		if *f.Verbleib == Offen {
			bedingungen = append(bedingungen, `i.verbleib IS NULL`)
		} else {
			bedingungen = append(bedingungen, `i.verbleib = ?`)
			args = append(args, string(*f.Verbleib))
		}
	}
	if f.RaumID != 0 {
		ctes = append(ctes, teilbaum("im_raum", "raum_id = ?"))
		cteArgs = append(cteArgs, f.RaumID)
		bedingungen = append(bedingungen, `(i.raum_id = ? OR i.lagerort_id IN (SELECT id FROM im_raum))`)
		args = append(args, f.RaumID)
	}
	if f.LagerortID != 0 {
		ctes = append(ctes, teilbaum("im_lagerort", "id = ?"))
		cteArgs = append(cteArgs, f.LagerortID)
		bedingungen = append(bedingungen, `i.lagerort_id IN (SELECT id FROM im_lagerort)`)
	}
	if f.MoebelID != 0 {
		ctes = append(ctes, teilbaum("im_moebel", "moebel_item_id = ?"))
		cteArgs = append(cteArgs, f.MoebelID)
		bedingungen = append(bedingungen, `i.lagerort_id IN (SELECT id FROM im_moebel)`)
	}
	if f.OhneLagerort {
		bedingungen = append(bedingungen, `i.lagerort_id IS NULL`)
	}

	abfrage := ""
	if len(ctes) > 0 {
		abfrage = "WITH RECURSIVE " + strings.Join(ctes, ", ") + " "
	}
	abfrage += `
		SELECT i.id, i.name, i.verbleib, f.id
		FROM item i JOIN foto f ON f.item_id = i.id AND f.hauptfoto = 1`
	if len(bedingungen) > 0 {
		abfrage += " WHERE " + strings.Join(bedingungen, " AND ")
	}
	// Einer mehr als verlangt, um zu erkennen, ob es weitere gibt.
	abfrage += ` ORDER BY i.geaendert_am DESC, i.id DESC LIMIT ? OFFSET ?`
	args = append(append(cteArgs, args...), f.Anzahl+1, f.Versatz)

	rows, err := s.db.QueryContext(ctx, abfrage, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	eintraege := []ItemListenEintrag{}
	for rows.Next() {
		var (
			e        ItemListenEintrag
			verbleib sql.NullString
		)
		if err := rows.Scan(&e.ID, &e.Name, &verbleib, &e.HauptfotoID); err != nil {
			return nil, false, err
		}
		e.Verbleib = verbleibAus(verbleib)
		eintraege = append(eintraege, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(eintraege) > f.Anzahl {
		return eintraege[:f.Anzahl], true, nil
	}
	return eintraege, false, nil
}

func likeMaskieren(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Store) FotoAbrufen(ctx context.Context, id int64) (Foto, error) {
	f, err := fotoScannen(s.db.QueryRowContext(ctx, fotoSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Foto{}, ErrNichtGefunden
	}
	return f, err
}

// itemAendern führt eine Änderung an einem bestehenden Item in einer eigenen Transaktion aus.
func (s *Store) itemAendern(ctx context.Context, id int64, jetzt time.Time, aendern func(*sql.Tx) error) (Item, error) {
	var item Item
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var err error
		item, err = itemAendernIn(ctx, tx, id, jetzt, aendern)
		return err
	})
	return item, err
}

// itemAendernIn führt die Änderung aus, setzt geaendert_am und liefert das Item danach.
func itemAendernIn(ctx context.Context, tx *sql.Tx, id int64, jetzt time.Time, aendern func(*sql.Tx) error) (Item, error) {
	res, err := tx.ExecContext(ctx, `UPDATE item SET geaendert_am = ? WHERE id = ?`, zeitText(jetzt), id)
	if err != nil {
		return Item{}, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return Item{}, errors.Join(err, ErrNichtGefunden)
	}
	if err := aendern(tx); err != nil {
		return Item{}, err
	}
	return itemLesen(ctx, tx, id)
}

func (s *Store) ItemBearbeiten(ctx context.Context, id int64, name, beschreibung string, verbleib Verbleib, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, id, jetzt, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE item SET name = ?, name_klein = ?, beschreibung = ?, verbleib = ? WHERE id = ?`,
			name, strings.ToLower(name), beschreibung, verbleibWert(verbleib), id)
		return err
	})
}

// ItemLoeschen löscht das Item samt Fotos und Besitz und liefert die Namen der Foto-Dateien,
// die der Aufrufer danach entfernt. Lagerorte des Items als Möbelstück fallen weg.
func (s *Store) ItemLoeschen(ctx context.Context, id int64, jetzt time.Time) ([]string, error) {
	var dateien []string
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		fotos, err := fotosLesen(ctx, tx, id)
		if err != nil {
			return err
		}
		eigene, err := benannteLesen(tx.QueryContext(ctx, `SELECT id, name FROM lagerort WHERE moebel_item_id = ?`, id))
		if err != nil {
			return err
		}
		for _, l := range eigene {
			// Ein eigener Lagerort kann schon mit einem anderen weggefallen sein, wenn er darin lag.
			if err := lagerortWegfall(ctx, tx, l.ID, jetzt); err != nil && !errors.Is(err, ErrNichtGefunden) {
				return err
			}
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM item WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			return errors.Join(err, ErrNichtGefunden)
		}
		for _, f := range fotos {
			dateien = append(dateien, f.Datei)
		}
		return nil
	})
	return dateien, err
}

func (s *Store) BesitzerHinzufuegen(ctx context.Context, itemID, personID int64, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		return besitzEintragen(ctx, tx, itemID, personID)
	})
}

// BesitzerEntfernen entfernt eine Person als Besitzer; der letzte Besitzer bleibt.
func (s *Store) BesitzerEntfernen(ctx context.Context, itemID, personID int64, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		var andere bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM besitz WHERE item_id = ? AND person_id <> ?)`,
			itemID, personID).Scan(&andere)
		if err != nil {
			return err
		}
		if !andere {
			return ErrLetzterBesitzer
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM besitz WHERE item_id = ? AND person_id = ?`, itemID, personID)
		return err
	})
}

func (s *Store) FotosHinzufuegen(ctx context.Context, itemID int64, fotos []NeuesFoto, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		return fotosEintragen(ctx, tx, itemID, fotos)
	})
}

// HauptfotoSetzen macht ein Foto des Items zum Hauptfoto.
func (s *Store) HauptfotoSetzen(ctx context.Context, itemID, fotoID int64, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		var gehoertDazu bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM foto WHERE id = ? AND item_id = ?)`, fotoID, itemID).Scan(&gehoertDazu)
		if err != nil {
			return err
		}
		if !gehoertDazu {
			return ErrNichtGefunden
		}
		// Erst das alte Hauptfoto zurücksetzen, sonst greift der Index „genau ein Hauptfoto“.
		if _, err := tx.ExecContext(ctx, `UPDATE foto SET hauptfoto = 0 WHERE item_id = ?`, itemID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE foto SET hauptfoto = 1 WHERE id = ?`, fotoID)
		return err
	})
}

// FotoLoeschen löscht ein Foto, außer es ist das letzte seines Items. War es das Hauptfoto,
// wird das nächste in der Reihenfolge Hauptfoto, nach dem letzten das erste. Geliefert werden
// der Dateiname, den der Aufrufer danach entfernt, und das Item.
func (s *Store) FotoLoeschen(ctx context.Context, fotoID int64, jetzt time.Time) (string, Item, error) {
	var (
		datei string
		item  Item
	)
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var itemID int64
		var reihenfolge, anzahl int
		err := tx.QueryRowContext(ctx, `
			SELECT f.item_id, f.datei, f.reihenfolge, (SELECT COUNT(*) FROM foto a WHERE a.item_id = f.item_id)
			FROM foto f WHERE f.id = ?`, fotoID).Scan(&itemID, &datei, &reihenfolge, &anzahl)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNichtGefunden
		}
		if err != nil {
			return err
		}
		if anzahl <= 1 {
			return ErrLetztesFoto
		}
		item, err = itemAendernIn(ctx, tx, itemID, jetzt, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, `DELETE FROM foto WHERE id = ?`, fotoID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `
				UPDATE foto SET hauptfoto = 1
				WHERE id = (
					SELECT id FROM foto WHERE item_id = ?
					ORDER BY CASE WHEN reihenfolge > ? THEN 0 ELSE 1 END, reihenfolge, id
					LIMIT 1)
				AND NOT EXISTS (SELECT 1 FROM foto WHERE item_id = ? AND hauptfoto = 1)`,
				itemID, reihenfolge, itemID)
			return err
		})
		return err
	})
	if err != nil {
		return "", Item{}, err
	}
	return datei, item, nil
}
