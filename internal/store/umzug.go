package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrKeinUmzug          = errors.New("kein laufender Umzug")
	ErrUmzugLaeuft        = errors.New("es läuft schon ein Umzug")
	ErrUmzugAbgeschlossen = errors.New("Umzug ist abgeschlossen")
	ErrGleicheWohnung     = errors.New("alte und neue Wohnung sind gleich")
	ErrUnbekannteWohnung  = errors.New("unbekannte Wohnung")
	ErrWohnungImUmzug     = errors.New("Wohnung gehört zu einem Umzug")
	ErrRaumImUmzug        = errors.New("Raum kommt in einem Umzug vor")
	ErrBoxNichtLeer       = errors.New("Box ist nicht leer")
	ErrUnbekannteBox      = errors.New("Box gehört nicht zum laufenden Umzug")
	ErrZielraumFalsch     = errors.New("Zielraum liegt nicht in der neuen Wohnung")
	ErrItemsOhneZielraum  = errors.New("Items der alten Wohnung ohne Zielraum")
	ErrUnbekanntesItem    = errors.New("unbekanntes Item")
)

type Umzug struct {
	ID              int64
	AlteWohnung     Benannt
	NeueWohnung     Benannt
	Laufend         bool
	ErstelltAm      time.Time
	AbgeschlossenAm *time.Time
	Boxen           []Box
}

type Box struct {
	ID          int64
	UmzugID     int64
	Nummer      int
	Bezeichnung string
	// Anzahl der Items in der Box.
	Anzahl int
}

const umzugSelect = `
	SELECT u.id, a.id, a.name, n.id, n.name, u.laufend = 1, u.erstellt_am, u.abgeschlossen_am
	FROM umzug u JOIN wohnung a ON a.id = u.alte_wohnung_id JOIN wohnung n ON n.id = u.neue_wohnung_id`

func umzugScannen(z zeile) (Umzug, error) {
	var (
		u             Umzug
		erstellt      string
		abgeschlossen sql.NullString
	)
	if err := z.Scan(&u.ID, &u.AlteWohnung.ID, &u.AlteWohnung.Name, &u.NeueWohnung.ID, &u.NeueWohnung.Name,
		&u.Laufend, &erstellt, &abgeschlossen); err != nil {
		return Umzug{}, err
	}
	var err error
	if u.ErstelltAm, err = zeitLesen(erstellt); err != nil {
		return Umzug{}, err
	}
	if abgeschlossen.Valid {
		t, err := zeitLesen(abgeschlossen.String)
		if err != nil {
			return Umzug{}, err
		}
		u.AbgeschlossenAm = &t
	}
	return u, nil
}

// umzugLesen liest einen Umzug samt Boxen; ohne ID den laufenden.
func umzugLesen(ctx context.Context, q abfrager, id int64) (Umzug, error) {
	abfrage, args := umzugSelect+` WHERE u.laufend = 1`, []any{}
	if id != 0 {
		abfrage, args = umzugSelect+` WHERE u.id = ?`, []any{id}
	}
	u, err := umzugScannen(q.QueryRowContext(ctx, abfrage, args...))
	if errors.Is(err, sql.ErrNoRows) {
		if id == 0 {
			return Umzug{}, ErrKeinUmzug
		}
		return Umzug{}, ErrNichtGefunden
	}
	if err != nil {
		return Umzug{}, err
	}
	u.Boxen, err = boxenLesen(q.QueryContext(ctx, boxSelect+` WHERE b.umzug_id = ? GROUP BY b.id ORDER BY b.nummer`, u.ID))
	return u, err
}

func (s *Store) LaufenderUmzug(ctx context.Context) (Umzug, error) { return umzugLesen(ctx, s.db, 0) }

func (s *Store) UmzugAbrufen(ctx context.Context, id int64) (Umzug, error) {
	return umzugLesen(ctx, s.db, id)
}

// UmzuegeAuflisten liefert die abgeschlossenen Umzüge, die jüngsten zuerst.
func (s *Store) UmzuegeAuflisten(ctx context.Context) ([]Umzug, error) {
	rows, err := s.db.QueryContext(ctx, umzugSelect+` WHERE u.laufend = 0 ORDER BY u.abgeschlossen_am DESC, u.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	umzuege := []Umzug{}
	for rows.Next() {
		u, err := umzugScannen(rows)
		if err != nil {
			return nil, err
		}
		umzuege = append(umzuege, u)
	}
	return umzuege, rows.Err()
}

func (s *Store) UmzugAnlegen(ctx context.Context, alteWohnungID, neueWohnungID int64, jetzt time.Time) (Umzug, error) {
	var u Umzug
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		if alteWohnungID == neueWohnungID {
			return ErrGleicheWohnung
		}
		var beide, laeuft bool
		err := tx.QueryRowContext(ctx, `
			SELECT (SELECT COUNT(*) FROM wohnung WHERE id IN (?, ?)) = 2, EXISTS (SELECT 1 FROM umzug WHERE laufend = 1)`,
			alteWohnungID, neueWohnungID).Scan(&beide, &laeuft)
		switch {
		case err != nil:
			return err
		case !beide:
			return ErrUnbekannteWohnung
		case laeuft:
			return ErrUmzugLaeuft
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO umzug (alte_wohnung_id, neue_wohnung_id, erstellt_am) VALUES (?, ?, ?) RETURNING id`,
			alteWohnungID, neueWohnungID, zeitText(jetzt)).Scan(&id); err != nil {
			return err
		}
		u, err = umzugLesen(ctx, tx, id)
		return err
	})
	return u, err
}

const boxSelect = `
	SELECT b.id, b.umzug_id, b.nummer, b.bezeichnung, COUNT(ua.item_id)
	FROM box b LEFT JOIN umzugsangabe ua ON ua.box_id = b.id`

func boxenLesen(rows *sql.Rows, err error) ([]Box, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	boxen := []Box{}
	for rows.Next() {
		var b Box
		if err := rows.Scan(&b.ID, &b.UmzugID, &b.Nummer, &b.Bezeichnung, &b.Anzahl); err != nil {
			return nil, err
		}
		boxen = append(boxen, b)
	}
	return boxen, rows.Err()
}

func boxLesen(ctx context.Context, q abfrager, bedingung string, args ...any) (Box, error) {
	boxen, err := boxenLesen(q.QueryContext(ctx, boxSelect+` WHERE `+bedingung+` GROUP BY b.id`, args...))
	if err != nil {
		return Box{}, err
	}
	if len(boxen) == 0 {
		return Box{}, ErrNichtGefunden
	}
	return boxen[0], nil
}

func (s *Store) BoxAbrufen(ctx context.Context, id int64) (Box, error) {
	return boxLesen(ctx, s.db, `b.id = ?`, id)
}

// BoxPerNummer liefert die Box mit der Nummer im laufenden Umzug.
func (s *Store) BoxPerNummer(ctx context.Context, nummer int) (Box, error) {
	if _, err := umzugLesen(ctx, s.db, 0); err != nil {
		return Box{}, err
	}
	return boxLesen(ctx, s.db, `b.nummer = ? AND b.umzug_id = (SELECT id FROM umzug WHERE laufend = 1)`, nummer)
}

// BoxAnlegen legt im laufenden Umzug eine Box mit der nächsten, nie vergebenen Nummer an.
func (s *Store) BoxAnlegen(ctx context.Context, bezeichnung string, jetzt time.Time) (Box, error) {
	var b Box
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		var umzugID int64
		var nummer int
		err := tx.QueryRowContext(ctx, `
			UPDATE umzug SET letzte_boxnummer = letzte_boxnummer + 1 WHERE laufend = 1 RETURNING id, letzte_boxnummer`).
			Scan(&umzugID, &nummer)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrKeinUmzug
		}
		if err != nil {
			return err
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO box (umzug_id, nummer, bezeichnung, erstellt_am) VALUES (?, ?, ?, ?) RETURNING id`,
			umzugID, nummer, bezeichnung, zeitText(jetzt)).Scan(&id); err != nil {
			return err
		}
		b, err = boxLesen(ctx, tx, `b.id = ?`, id)
		return err
	})
	return b, err
}

// boxAendern führt eine Änderung an einer Box des laufenden Umzugs aus.
func (s *Store) boxAendern(ctx context.Context, id int64, aendern func(tx *sql.Tx, b Box) error) (Box, error) {
	var b Box
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		if err := umzugLaeuft(ctx, tx, `SELECT umzug_id FROM box WHERE id = ?`, id); err != nil {
			return err
		}
		vorher, err := boxLesen(ctx, tx, `b.id = ?`, id)
		if err != nil {
			return err
		}
		if err := aendern(tx, vorher); err != nil {
			return err
		}
		b, err = boxLesen(ctx, tx, `b.id = ?`, id)
		if errors.Is(err, ErrNichtGefunden) {
			err = nil
		}
		return err
	})
	return b, err
}

// umzugLaeuft prüft, dass der Umzug, den die Abfrage liefert, noch läuft.
func umzugLaeuft(ctx context.Context, q abfrager, umzugAbfrage string, args ...any) error {
	var laufend bool
	err := q.QueryRowContext(ctx, `SELECT laufend = 1 FROM umzug WHERE id = (`+umzugAbfrage+`)`, args...).Scan(&laufend)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNichtGefunden
	}
	if err != nil {
		return err
	}
	if !laufend {
		return ErrUmzugAbgeschlossen
	}
	return nil
}

func (s *Store) BoxUmbenennen(ctx context.Context, id int64, bezeichnung string) (Box, error) {
	return s.boxAendern(ctx, id, func(tx *sql.Tx, _ Box) error {
		_, err := tx.ExecContext(ctx, `UPDATE box SET bezeichnung = ? WHERE id = ?`, bezeichnung, id)
		return err
	})
}

func (s *Store) BoxLoeschen(ctx context.Context, id int64) error {
	_, err := s.boxAendern(ctx, id, func(tx *sql.Tx, b Box) error {
		if b.Anzahl > 0 {
			return ErrBoxNichtLeer
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM box WHERE id = ?`, id)
		return err
	})
	return err
}

// Umzugsangaben sind Herkunftsraum, Zielraum und Box eines Items im laufenden Umzug.
type Umzugsangaben struct {
	Herkunftsraum *Benannt
	Zielraum      *Benannt
	Box           *Box
}

// AngabenAenderung ändert Umzugsangaben: nil lässt eine Angabe, wie sie ist, ein Zeiger auf 0 setzt sie zurück.
type AngabenAenderung struct {
	HerkunftsraumID *int64
	ZielraumID      *int64
	BoxID           *int64
}

func (s *Store) UmzugsangabenSetzen(ctx context.Context, itemID int64, a AngabenAenderung, jetzt time.Time) (Item, error) {
	return s.itemAendern(ctx, itemID, jetzt, func(tx *sql.Tx) error {
		return angabenSetzen(ctx, tx, itemID, a)
	})
}

// UmzugsangabenSammeln ändert die Angaben mehrerer Items, ganz oder gar nicht.
func (s *Store) UmzugsangabenSammeln(ctx context.Context, itemIDs []int64, a AngabenAenderung, jetzt time.Time) error {
	return s.inTransaktion(ctx, func(tx *sql.Tx) error {
		for _, id := range itemIDs {
			res, err := tx.ExecContext(ctx, `UPDATE item SET geaendert_am = ? WHERE id = ?`, zeitText(jetzt), id)
			if err != nil {
				return err
			}
			if n, err := res.RowsAffected(); err != nil || n == 0 {
				return errors.Join(err, ErrUnbekanntesItem)
			}
			if err := angabenSetzen(ctx, tx, id, a); err != nil {
				return err
			}
		}
		return nil
	})
}

// angabenSetzen legt die Umzugsangaben bei Bedarf an, mit dem aktuellen Raum des Items als
// Herkunftsraum, und ändert dann die angegebenen Felder.
func angabenSetzen(ctx context.Context, q abfrager, itemID int64, a AngabenAenderung) error {
	var umzugID, neueWohnungID int64
	err := q.QueryRowContext(ctx, `SELECT id, neue_wohnung_id FROM umzug WHERE laufend = 1`).Scan(&umzugID, &neueWohnungID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrKeinUmzug
	}
	if err != nil {
		return err
	}
	if a.ZielraumID != nil && *a.ZielraumID != 0 {
		var wohnungID int64
		err := q.QueryRowContext(ctx, `SELECT wohnung_id FROM raum WHERE id = ?`, *a.ZielraumID).Scan(&wohnungID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUnbekannterPlatz
		}
		if err != nil {
			return err
		}
		if wohnungID != neueWohnungID {
			return ErrZielraumFalsch
		}
	}
	if a.HerkunftsraumID != nil && *a.HerkunftsraumID != 0 {
		if err := platzPruefen(ctx, q, *a.HerkunftsraumID, 0); err != nil {
			return err
		}
	}
	if a.BoxID != nil && *a.BoxID != 0 {
		var passt bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM box WHERE id = ? AND umzug_id = ?)`, *a.BoxID, umzugID).Scan(&passt); err != nil {
			return err
		}
		if !passt {
			return ErrUnbekannteBox
		}
	}

	raum, err := aktuellerRaum(ctx, q, itemID)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `
		INSERT INTO umzugsangabe (item_id, umzug_id, herkunftsraum_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		itemID, umzugID, nullID(raum)); err != nil {
		return err
	}
	for spalte, wert := range map[string]*int64{"herkunftsraum_id": a.HerkunftsraumID, "zielraum_id": a.ZielraumID, "box_id": a.BoxID} {
		if wert == nil {
			continue
		}
		if _, err := q.ExecContext(ctx, `UPDATE umzugsangabe SET `+spalte+` = ? WHERE item_id = ? AND umzug_id = ?`,
			nullID(*wert), itemID, umzugID); err != nil {
			return err
		}
	}
	return nil
}

// aktuellerRaum ist der Raum, in dem ein Item gerade platziert ist, über seinen Lagerort oder direkt; 0 ohne Platzierung.
func aktuellerRaum(ctx context.Context, q abfrager, itemID int64) (int64, error) {
	var lagerortID, raumID sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT lagerort_id, raum_id FROM item WHERE id = ?`, itemID).Scan(&lagerortID, &raumID); err != nil {
		return 0, err
	}
	p, err := platzierungLesen(ctx, q, lagerortID.Int64, raumID.Int64)
	if err != nil || p == nil {
		return 0, err
	}
	return p.Raum.ID, nil
}

// angabenLesen liest die Umzugsangaben eines Items im laufenden Umzug; nil, wenn es keine gibt.
func angabenLesen(ctx context.Context, q abfrager, itemID int64) (*Umzugsangaben, error) {
	var (
		herkunftID, zielID, boxID sql.NullInt64
		herkunftName, zielName    sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT h.id, h.name, z.id, z.name, ua.box_id
		FROM umzugsangabe ua JOIN umzug u ON u.id = ua.umzug_id AND u.laufend = 1
		LEFT JOIN raum h ON h.id = ua.herkunftsraum_id LEFT JOIN raum z ON z.id = ua.zielraum_id
		WHERE ua.item_id = ?`, itemID).Scan(&herkunftID, &herkunftName, &zielID, &zielName, &boxID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := &Umzugsangaben{Herkunftsraum: benanntAus(herkunftID, herkunftName), Zielraum: benanntAus(zielID, zielName)}
	if boxID.Valid {
		b, err := boxLesen(ctx, q, `b.id = ?`, boxID.Int64)
		if err != nil {
			return nil, err
		}
		a.Box = &b
	}
	return a, nil
}

func benanntAus(id sql.NullInt64, name sql.NullString) *Benannt {
	if !id.Valid {
		return nil
	}
	return &Benannt{ID: id.Int64, Name: name.String}
}

// ZielraumEintrag ist ein Raum der neuen Wohnung mit den Boxen und Items, die dorthin gehören.
type ZielraumEintrag struct {
	Raum  Benannt
	Boxen []Box
	Items []ItemListenEintrag
}

func (s *Store) Zielraumuebersicht(ctx context.Context) ([]ZielraumEintrag, error) {
	u, err := umzugLesen(ctx, s.db, 0)
	if err != nil {
		return nil, err
	}
	raeume, err := benannteLesen(s.db.QueryContext(ctx, `SELECT id, name FROM raum WHERE wohnung_id = ? ORDER BY name, id`, u.NeueWohnung.ID))
	if err != nil {
		return nil, err
	}
	uebersicht := []ZielraumEintrag{}
	for _, r := range raeume {
		boxen, err := boxenLesen(s.db.QueryContext(ctx, boxSelect+`
			WHERE b.id IN (SELECT box_id FROM umzugsangabe WHERE umzug_id = ? AND zielraum_id = ?)
			GROUP BY b.id ORDER BY b.nummer`, u.ID, r.ID))
		if err != nil {
			return nil, err
		}
		items, _, err := s.ItemsAuflisten(ctx, ItemFilter{ZielraumID: r.ID, Anzahl: 10_000})
		if err != nil {
			return nil, err
		}
		uebersicht = append(uebersicht, ZielraumEintrag{Raum: r, Boxen: boxen, Items: items})
	}
	return uebersicht, nil
}

// UmzugAbschliessen stellt alle Items mit Zielraum direkt in ihren Zielraum, ohne Lagerort, und schließt
// den Umzug ab. Items ohne Zielraum behalten ihre Platzierung. Die Umzugsangaben bleiben als Verlauf.
func (s *Store) UmzugAbschliessen(ctx context.Context, jetzt time.Time) (Umzug, error) {
	var u Umzug
	err := s.inTransaktion(ctx, func(tx *sql.Tx) error {
		laufend, err := umzugLesen(ctx, tx, 0)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE item SET lagerort_id = NULL, geaendert_am = ?,
				raum_id = (SELECT zielraum_id FROM umzugsangabe WHERE item_id = item.id AND umzug_id = ?)
			WHERE id IN (SELECT item_id FROM umzugsangabe WHERE umzug_id = ? AND zielraum_id IS NOT NULL)`,
			zeitText(jetzt), laufend.ID, laufend.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE umzug SET laufend = 0, abgeschlossen_am = ? WHERE id = ?`,
			zeitText(jetzt), laufend.ID); err != nil {
			return err
		}
		u, err = umzugLesen(ctx, tx, laufend.ID)
		return err
	})
	return u, err
}
