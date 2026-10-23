package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrWohnungNichtLeer   = errors.New("Wohnung hat noch Räume")
	ErrRaumNichtLeer      = errors.New("Raum hat noch Lagerorte oder Items")
	ErrZyklus             = errors.New("Lagerort läge in sich selbst")
	ErrMoebelInSichSelbst = errors.New("Möbelstück läge in einem eigenen Lagerort")
	ErrUnbekannterPlatz   = errors.New("unbekannter Raum oder Lagerort")
)

// Ort ist ein benanntes Element eines Pfads: Wohnung, Raum oder Lagerort.
type Ort struct {
	ID   int64
	Name string
}

type Wohnung struct {
	ID     int64
	Name   string
	Raeume []Ort
}

func (s *Store) WohnungenAuflisten(ctx context.Context) ([]Wohnung, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.name, r.id, r.name
		FROM wohnung w LEFT JOIN raum r ON r.wohnung_id = w.id
		ORDER BY w.name, w.id, r.name, r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	wohnungen := []Wohnung{}
	for rows.Next() {
		var (
			w        Wohnung
			raumID   sql.NullInt64
			raumName sql.NullString
		)
		if err := rows.Scan(&w.ID, &w.Name, &raumID, &raumName); err != nil {
			return nil, err
		}
		if n := len(wohnungen); n == 0 || wohnungen[n-1].ID != w.ID {
			w.Raeume = []Ort{}
			wohnungen = append(wohnungen, w)
		}
		if raumID.Valid {
			letzte := &wohnungen[len(wohnungen)-1]
			letzte.Raeume = append(letzte.Raeume, Ort{ID: raumID.Int64, Name: raumName.String})
		}
	}
	return wohnungen, rows.Err()
}

func (s *Store) WohnungAnlegen(ctx context.Context, name string, jetzt time.Time) (Ort, error) {
	o := Ort{Name: name}
	err := s.db.QueryRowContext(ctx, `INSERT INTO wohnung (name, erstellt_am) VALUES (?, ?) RETURNING id`, name, zeitText(jetzt)).Scan(&o.ID)
	return o, err
}

func (s *Store) WohnungUmbenennen(ctx context.Context, id int64, name string) (Ort, error) {
	return umbenennen(ctx, s.db, "wohnung", id, name)
}

// umbenennen setzt den Namen einer Zeile in einer der Tabellen wohnung, raum oder lagerort.
func umbenennen(ctx context.Context, q abfrager, tabelle string, id int64, name string) (Ort, error) {
	res, err := q.ExecContext(ctx, `UPDATE `+tabelle+` SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return Ort{}, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return Ort{}, errors.Join(err, ErrNichtGefunden)
	}
	return Ort{ID: id, Name: name}, nil
}

func (s *Store) WohnungLoeschen(ctx context.Context, id int64) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var existiert, raeume bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM wohnung WHERE id = ?), EXISTS (SELECT 1 FROM raum WHERE wohnung_id = ?)`,
			id, id).Scan(&existiert, &raeume)
		switch {
		case err != nil:
			return err
		case !existiert:
			return ErrNichtGefunden
		case raeume:
			return ErrWohnungNichtLeer
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM wohnung WHERE id = ?`, id)
		return err
	})
}

func (s *Store) RaumAnlegen(ctx context.Context, wohnungID int64, name string, jetzt time.Time) (Ort, error) {
	o := Ort{Name: name}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO raum (wohnung_id, name, erstellt_am)
		SELECT id, ?, ? FROM wohnung WHERE id = ? RETURNING id`, name, zeitText(jetzt), wohnungID).Scan(&o.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return Ort{}, ErrNichtGefunden
	}
	return o, err
}

func (s *Store) RaumUmbenennen(ctx context.Context, id int64, name string) (Ort, error) {
	return umbenennen(ctx, s.db, "raum", id, name)
}

// RaumLoeschen löscht einen leeren Raum: ohne Lagerorte und ohne direkt darin platzierte Items.
func (s *Store) RaumLoeschen(ctx context.Context, id int64) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var existiert, belegt bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM raum WHERE id = ?),
				EXISTS (SELECT 1 FROM lagerort WHERE raum_id = ?) OR EXISTS (SELECT 1 FROM item WHERE raum_id = ?)`,
			id, id, id).Scan(&existiert, &belegt)
		switch {
		case err != nil:
			return err
		case !existiert:
			return ErrNichtGefunden
		case belegt:
			return ErrRaumNichtLeer
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM raum WHERE id = ?`, id)
		return err
	})
}

// Raum ist ein Raum mit seiner Wohnung und dem Baum seiner Lagerorte.
type Raum struct {
	Ort
	Wohnung   Ort
	Lagerorte []Lagerort
}

// Lagerort ist ein Knoten im Lagerort-Baum.
type Lagerort struct {
	Ort
	Moebel *Ort
	Unter  []Lagerort
}

func (s *Store) RaumAbrufen(ctx context.Context, id int64) (Raum, error) {
	var r Raum
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.name, w.id, w.name FROM raum r JOIN wohnung w ON w.id = r.wohnung_id WHERE r.id = ?`, id).
		Scan(&r.ID, &r.Name, &r.Wohnung.ID, &r.Wohnung.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Raum{}, ErrNichtGefunden
	}
	if err != nil {
		return Raum{}, err
	}
	r.Lagerorte, err = lagerortBaum(ctx, s.db, id)
	return r, err
}

// lagerortBaum liest alle Lagerorte eines Raums samt Unterlagerorten als Baum.
func lagerortBaum(ctx context.Context, q abfrager, raumID int64) ([]Lagerort, error) {
	rows, err := q.QueryContext(ctx, `
		WITH RECURSIVE baum (id, eltern_id) AS (
			SELECT id, NULL FROM lagerort WHERE raum_id = ?
			UNION ALL
			SELECT l.id, l.eltern_id FROM lagerort l JOIN baum b ON l.eltern_id = b.id
		)
		SELECT l.id, l.name, b.eltern_id, i.id, i.name
		FROM baum b JOIN lagerort l ON l.id = b.id LEFT JOIN item i ON i.id = l.moebel_item_id
		ORDER BY l.name, l.id`, raumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type knoten struct {
		l      Lagerort
		eltern sql.NullInt64
	}
	var alle []knoten
	for rows.Next() {
		var (
			k          knoten
			moebelID   sql.NullInt64
			moebelName sql.NullString
		)
		if err := rows.Scan(&k.l.ID, &k.l.Name, &k.eltern, &moebelID, &moebelName); err != nil {
			return nil, err
		}
		if moebelID.Valid {
			k.l.Moebel = &Ort{ID: moebelID.Int64, Name: moebelName.String}
		}
		alle = append(alle, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	kinder := map[int64][]int64{}
	var wurzeln []int64
	nachID := map[int64]Lagerort{}
	for _, k := range alle {
		nachID[k.l.ID] = k.l
		if k.eltern.Valid {
			kinder[k.eltern.Int64] = append(kinder[k.eltern.Int64], k.l.ID)
		} else {
			wurzeln = append(wurzeln, k.l.ID)
		}
	}
	var bauen func(ids []int64) []Lagerort
	bauen = func(ids []int64) []Lagerort {
		out := []Lagerort{}
		for _, id := range ids {
			l := nachID[id]
			l.Unter = bauen(kinder[id])
			out = append(out, l)
		}
		return out
	}
	return bauen(wurzeln), nil
}

// Pfad beschreibt, wo etwas liegt: Wohnung › Raum › Lagerorte von außen nach innen.
type Pfad struct {
	Wohnung   Ort
	Raum      Ort
	Lagerorte []Ort
}

// Platz ist die Platzierung eines Items: ein Lagerort, ein Raum ohne Lagerort oder (beide 0) nichts.
type Platz struct {
	LagerortID int64
	RaumID     int64
}

// raumPfad liefert den Pfad eines Raums.
func raumPfad(ctx context.Context, q abfrager, raumID int64) (Pfad, error) {
	var p Pfad
	err := q.QueryRowContext(ctx, `
		SELECT w.id, w.name, r.id, r.name FROM raum r JOIN wohnung w ON w.id = r.wohnung_id WHERE r.id = ?`, raumID).
		Scan(&p.Wohnung.ID, &p.Wohnung.Name, &p.Raum.ID, &p.Raum.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return Pfad{}, ErrNichtGefunden
	}
	p.Lagerorte = []Ort{}
	return p, err
}

// lagerortPfad liefert den Pfad bis einschließlich des Lagerorts, über die Kette seiner Eltern.
func lagerortPfad(ctx context.Context, q abfrager, lagerortID int64) (Pfad, error) {
	rows, err := q.QueryContext(ctx, `
		WITH RECURSIVE kette (id, name, eltern_id, raum_id, tiefe) AS (
			SELECT id, name, eltern_id, raum_id, 0 FROM lagerort WHERE id = ?
			UNION ALL
			SELECT l.id, l.name, l.eltern_id, l.raum_id, k.tiefe + 1 FROM lagerort l JOIN kette k ON l.id = k.eltern_id
		)
		SELECT id, name, raum_id FROM kette ORDER BY tiefe DESC`, lagerortID)
	if err != nil {
		return Pfad{}, err
	}
	var (
		lagerorte []Ort
		raumID    sql.NullInt64
	)
	for rows.Next() {
		var (
			o Ort
			r sql.NullInt64
		)
		if err := rows.Scan(&o.ID, &o.Name, &r); err != nil {
			rows.Close()
			return Pfad{}, err
		}
		if r.Valid {
			raumID = r
		}
		lagerorte = append(lagerorte, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Pfad{}, err
	}
	if len(lagerorte) == 0 {
		return Pfad{}, ErrNichtGefunden
	}
	p, err := raumPfad(ctx, q, raumID.Int64)
	p.Lagerorte = lagerorte
	return p, err
}

// LagerortDetail ist ein Lagerort mit Pfad, Möbelstück und direkten Unterlagerorten.
type LagerortDetail struct {
	Ort
	Pfad   Pfad
	Moebel *Ort
	Unter  []Ort
}

type NeuerLagerort struct {
	Name string
	// Genau eins von beiden: der Raum oder der Eltern-Lagerort.
	RaumID   int64
	ElternID int64
}

func (s *Store) LagerortAnlegen(ctx context.Context, neu NeuerLagerort, jetzt time.Time) (LagerortDetail, error) {
	var l LagerortDetail
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		if err := elternPruefen(ctx, tx, neu.RaumID, neu.ElternID); err != nil {
			return err
		}
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO lagerort (name, raum_id, eltern_id, erstellt_am) VALUES (?, ?, ?, ?) RETURNING id`,
			neu.Name, nullID(neu.RaumID), nullID(neu.ElternID), zeitText(jetzt)).Scan(&id)
		if err != nil {
			return err
		}
		l, err = lagerortLesen(ctx, tx, id)
		return err
	})
	return l, err
}

// elternPruefen prüft, dass es den Raum oder den Eltern-Lagerort gibt.
func elternPruefen(ctx context.Context, q abfrager, raumID, elternID int64) error {
	tabelle, id := "raum", raumID
	if elternID != 0 {
		tabelle, id = "lagerort", elternID
	}
	var existiert bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+tabelle+` WHERE id = ?)`, id).Scan(&existiert); err != nil {
		return err
	}
	if !existiert {
		return ErrUnbekannterPlatz
	}
	return nil
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *Store) LagerortAbrufen(ctx context.Context, id int64) (LagerortDetail, error) {
	return lagerortLesen(ctx, s.db, id)
}

func lagerortLesen(ctx context.Context, q abfrager, id int64) (LagerortDetail, error) {
	var (
		l          LagerortDetail
		moebelID   sql.NullInt64
		moebelName sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT l.id, l.name, i.id, i.name FROM lagerort l LEFT JOIN item i ON i.id = l.moebel_item_id WHERE l.id = ?`, id).
		Scan(&l.ID, &l.Name, &moebelID, &moebelName)
	if errors.Is(err, sql.ErrNoRows) {
		return LagerortDetail{}, ErrNichtGefunden
	}
	if err != nil {
		return LagerortDetail{}, err
	}
	if moebelID.Valid {
		l.Moebel = &Ort{ID: moebelID.Int64, Name: moebelName.String}
	}
	if l.Pfad, err = lagerortPfad(ctx, q, id); err != nil {
		return LagerortDetail{}, err
	}
	l.Unter, err = orteLesen(q.QueryContext(ctx, `SELECT id, name FROM lagerort WHERE eltern_id = ? ORDER BY name, id`, id))
	return l, err
}

func orteLesen(rows *sql.Rows, err error) ([]Ort, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orte := []Ort{}
	for rows.Next() {
		var o Ort
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		orte = append(orte, o)
	}
	return orte, rows.Err()
}

func (s *Store) LagerortUmbenennen(ctx context.Context, id int64, name string) (Ort, error) {
	return umbenennen(ctx, s.db, "lagerort", id, name)
}

// MoebelSetzen ordnet den Lagerort einem Item (dem Möbelstück) zu; itemID 0 löst die Zuordnung.
func (s *Store) MoebelSetzen(ctx context.Context, lagerortID, itemID int64) (LagerortDetail, error) {
	var l LagerortDetail
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		if itemID != 0 {
			var existiert bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM item WHERE id = ?)`, itemID).Scan(&existiert); err != nil {
				return err
			}
			if !existiert {
				return ErrNichtGefunden
			}
		}
		res, err := tx.ExecContext(ctx, `UPDATE lagerort SET moebel_item_id = ? WHERE id = ?`, nullID(itemID), lagerortID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			return errors.Join(err, ErrNichtGefunden)
		}
		if err := moebelPruefen(ctx, tx); err != nil {
			return err
		}
		l, err = lagerortLesen(ctx, tx, lagerortID)
		return err
	})
	return l, err
}

// moebelPruefen stellt sicher, dass kein Möbelstück in einem seiner eigenen Lagerorte
// oder deren Unterlagerorten platziert ist.
func moebelPruefen(ctx context.Context, q abfrager) error {
	var verletzt bool
	err := q.QueryRowContext(ctx, `
		WITH RECURSIVE eigene (moebel_item_id, id) AS (
			SELECT moebel_item_id, id FROM lagerort WHERE moebel_item_id IS NOT NULL
			UNION ALL
			SELECT e.moebel_item_id, l.id FROM lagerort l JOIN eigene e ON l.eltern_id = e.id
		)
		SELECT EXISTS (SELECT 1 FROM eigene e JOIN item i ON i.id = e.moebel_item_id AND i.lagerort_id = e.id)`).Scan(&verletzt)
	if err != nil {
		return err
	}
	if verletzt {
		return ErrMoebelInSichSelbst
	}
	return nil
}

// platzSetzen setzt die Platzierung eines Items; es bleibt höchstens Lagerort oder Raum gesetzt.
func platzSetzen(ctx context.Context, q abfrager, itemID int64, p Platz) error {
	if p.LagerortID != 0 || p.RaumID != 0 {
		if err := elternPruefen(ctx, q, p.RaumID, p.LagerortID); err != nil {
			return err
		}
	}
	if p.LagerortID != 0 {
		p.RaumID = 0
	}
	if _, err := q.ExecContext(ctx, `UPDATE item SET lagerort_id = ?, raum_id = ? WHERE id = ?`,
		nullID(p.LagerortID), nullID(p.RaumID), itemID); err != nil {
		return err
	}
	return moebelPruefen(ctx, q)
}

func (s *Store) PlatzierungSetzen(ctx context.Context, itemID int64, p Platz, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		return platzSetzen(ctx, tx, itemID, p)
	})
}

// unterbaumCTE liefert als „unterbaum“ den Lagerort mit dem Platzhalter-ID und alle seine Unterlagerorte.
const unterbaumCTE = `WITH RECURSIVE unterbaum (id) AS (
	SELECT id FROM lagerort WHERE id = ?
	UNION ALL
	SELECT l.id FROM lagerort l JOIN unterbaum u ON l.eltern_id = u.id
) `

// LagerortVerschieben hängt den Lagerort samt Inhalt in einen Raum oder unter einen anderen Lagerort.
func (s *Store) LagerortVerschieben(ctx context.Context, id, raumID, elternID int64) (LagerortDetail, error) {
	var l LagerortDetail
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var existiert bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM lagerort WHERE id = ?)`, id).Scan(&existiert); err != nil {
			return err
		}
		if !existiert {
			return ErrNichtGefunden
		}
		if err := elternPruefen(ctx, tx, raumID, elternID); err != nil {
			return err
		}
		if elternID != 0 {
			var zyklus bool
			if err := tx.QueryRowContext(ctx, unterbaumCTE+`SELECT EXISTS (SELECT 1 FROM unterbaum WHERE id = ?)`, id, elternID).Scan(&zyklus); err != nil {
				return err
			}
			if zyklus {
				return ErrZyklus
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE lagerort SET raum_id = ?, eltern_id = ? WHERE id = ?`, nullID(raumID), nullID(elternID), id); err != nil {
			return err
		}
		if err := moebelPruefen(ctx, tx); err != nil {
			return err
		}
		var err error
		l, err = lagerortLesen(ctx, tx, id)
		return err
	})
	return l, err
}

// LagerortLoeschen löscht den Lagerort samt Unterlagerorten; ihre Items stehen danach
// ohne Lagerort in dem Raum, in dem der Lagerort lag.
func (s *Store) LagerortLoeschen(ctx context.Context, id int64, jetzt time.Time) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		return lagerortWegfall(ctx, tx, id, jetzt)
	})
}

func lagerortWegfall(ctx context.Context, q abfrager, id int64, jetzt time.Time) error {
	pfad, err := lagerortPfad(ctx, q, id)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, unterbaumCTE+`
		UPDATE item SET lagerort_id = NULL, raum_id = ?, geaendert_am = ?
		WHERE lagerort_id IN (SELECT id FROM unterbaum)`, id, pfad.Raum.ID, zeitText(jetzt)); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, unterbaumCTE+`DELETE FROM lagerort WHERE id IN (SELECT id FROM unterbaum)`, id)
	return err
}

// PlatzEintrag ist ein Raum oder Lagerort mit seinem Pfad als Text, z. B. für eine Auswahlliste.
type PlatzEintrag struct {
	Art  string // "raum" oder "lagerort"
	ID   int64
	Pfad string
}

// PlaetzeAuflisten liefert alle Räume und Lagerorte, nach Pfad sortiert.
func (s *Store) PlaetzeAuflisten(ctx context.Context) ([]PlatzEintrag, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE lager (id, pfad) AS (
			SELECT l.id, w.name || ' › ' || r.name || ' › ' || l.name
			FROM lagerort l JOIN raum r ON r.id = l.raum_id JOIN wohnung w ON w.id = r.wohnung_id
			UNION ALL
			SELECT l.id, g.pfad || ' › ' || l.name FROM lagerort l JOIN lager g ON l.eltern_id = g.id
		)
		SELECT 'raum', r.id, w.name || ' › ' || r.name FROM raum r JOIN wohnung w ON w.id = r.wohnung_id
		UNION ALL
		SELECT 'lagerort', id, pfad FROM lager
		ORDER BY 3, 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plaetze := []PlatzEintrag{}
	for rows.Next() {
		var p PlatzEintrag
		if err := rows.Scan(&p.Art, &p.ID, &p.Pfad); err != nil {
			return nil, err
		}
		plaetze = append(plaetze, p)
	}
	return plaetze, rows.Err()
}
