package store

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"
	"time"
)

// germanData fills a database at migration level 0007 the way the German schema stored it.
const germanData = `
INSERT INTO person VALUES (1, 'Alex', '2026-01-01T00:00:00.000000000Z'), (2, 'Kim', '2026-01-01T00:00:00.000000000Z');
INSERT INTO nutzer VALUES (1, 'alex@example.org', 'hash', 'weich'), (2, 'kim@example.org', 'hash', NULL);
INSERT INTO sitzung VALUES ('token', 1, '2026-01-01T00:00:00.000000000Z', '2099-01-01T00:00:00.000000000Z');
INSERT INTO wohnung VALUES (1, 'Altbau', 'x'), (2, 'Neubau', 'x');
INSERT INTO raum VALUES (1, 1, 'Küche', 'x'), (2, 2, 'Wohnzimmer', 'x');
INSERT INTO item (id, name, name_klein, beschreibung, verbleib, erstellt_am, geaendert_am) VALUES
	(1, 'Regal', 'regal', '', NULL, '2026-01-01T00:00:00.000000000Z', '2026-01-01T00:00:01.000000000Z'),
	(2, 'Topf', 'topf', 'Kupfer', 'behalten', '2026-01-01T00:00:00.000000000Z', '2026-01-01T00:00:02.000000000Z'),
	(3, 'Lampe', 'lampe', '', 'verkaufen', '2026-01-01T00:00:00.000000000Z', '2026-01-01T00:00:03.000000000Z'),
	(4, 'Bild', 'bild', '', 'spaeter_verkaufen', '2026-01-01T00:00:00.000000000Z', '2026-01-01T00:00:04.000000000Z'),
	(5, 'Buch', 'buch', '', 'verschenken', '2026-01-01T00:00:00.000000000Z', '2026-01-01T00:00:05.000000000Z');
INSERT INTO lagerort VALUES (1, 'Fach', 1, NULL, 1, 'x'), (2, 'Kiste', NULL, 1, NULL, 'x');
UPDATE item SET lagerort_id = 2 WHERE id = 2;
UPDATE item SET raum_id = 1 WHERE id = 3;
INSERT INTO foto (item_id, datei, typ, hauptfoto, reihenfolge) VALUES
	(1, 'a.jpg', 'image/jpeg', 1, 1), (2, 'b.jpg', 'image/jpeg', 1, 1), (3, 'c.jpg', 'image/jpeg', 1, 1),
	(4, 'd.jpg', 'image/jpeg', 1, 1), (5, 'e.jpg', 'image/jpeg', 1, 1), (5, 'f.png', 'image/png', 0, 2), (5, 'g.png', 'image/png', 0, 3);
DELETE FROM foto WHERE datei = 'g.png';
INSERT INTO besitz VALUES (1, 1), (2, 1), (2, 2), (3, 2), (4, 1), (5, 1);
INSERT INTO umzug VALUES (1, 1, 2, 1, 1, '2026-01-01T00:00:00.000000000Z', NULL);
INSERT INTO box VALUES (1, 1, 1, 'Küche', 'x');
INSERT INTO umzugsangabe VALUES (2, 1, 1, 2, 1);
INSERT INTO einladung VALUES (1, 'th', NULL, 'Oma', 1, '2026-01-01T00:00:00.000000000Z', '2099-01-01T00:00:00.000000000Z', NULL, NULL);
`

func TestEnglishNamesMigrationKeepsData(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+url.PathEscape(filepath.Join(dir, "inventar.db"))+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migration (version TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"0001_person_nutzer_sitzung", "0002_item_foto_besitz", "0003_wohnung_raum_lagerort",
		"0004_umzug_box", "0005_einladung", "0006_foto_ids_nicht_wiederverwenden", "0007_nutzer_design"} {
		content, err := migrations.ReadFile("migrations/" + version + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migration (version) VALUES (?)`, version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(germanData); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer s.Close()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	u, err := s.SessionUser(ctx, "token", now)
	if err != nil || u.Name != "Alex" || u.Theme != ThemeSoft {
		t.Fatalf("session user: %+v, %v", u, err)
	}
	if kim, err := s.UserByEmail(ctx, "kim@example.org"); err != nil || kim.Theme != NoTheme {
		t.Fatalf("kim: %+v, %v", kim, err)
	}

	want := map[int64]Disposition{1: Undecided, 2: Keep, 3: Sell, 4: SellLater, 5: GiveAway}
	for id, d := range want {
		item, err := s.GetItem(ctx, id)
		if err != nil {
			t.Fatalf("item %d: %v", id, err)
		}
		if item.Disposition != d {
			t.Errorf("item %d: disposition %q, want %q", id, item.Disposition, d)
		}
	}
	pot, _ := s.GetItem(ctx, 2)
	if pot.Description != "Kupfer" || len(pot.Owners) != 2 || pot.Placement == nil ||
		len(pot.Placement.StorageLocations) != 2 || pot.Placement.Room.Name != "Küche" {
		t.Errorf("pot: %+v", pot)
	}
	if pot.Move == nil || pot.Move.TargetRoom == nil || pot.Move.TargetRoom.Name != "Wohnzimmer" || pot.Move.Box == nil || pot.Move.Box.Label != "Küche" {
		t.Errorf("pot move details: %+v", pot.Move)
	}
	shelf, _ := s.GetItem(ctx, 1)
	if len(shelf.FurnitureStorageLocations) != 1 || shelf.FurnitureStorageLocations[0].Name != "Fach" {
		t.Errorf("furniture: %+v", shelf.FurnitureStorageLocations)
	}
	book, _ := s.GetItem(ctx, 5)
	if len(book.Photos) != 2 || !book.Photos[0].MainPhoto || book.Photos[1].File != "f.png" || book.Photos[1].Type != "image/png" {
		t.Errorf("photos: %+v", book.Photos)
	}
	if invitations, err := s.OpenInvitations(ctx, now); err != nil || len(invitations) != 1 || invitations[0].CreatedBy != "Alex" {
		t.Errorf("invitations: %+v, %v", invitations, err)
	}

	// Photo IDs are still not reused after the rename (AUTOINCREMENT sequence kept).
	if _, err := s.AddPhotos(ctx, 5, []NewPhoto{{File: "h.png", Type: "image/png"}}, now); err != nil {
		t.Fatal(err)
	}
	book, _ = s.GetItem(ctx, 5)
	if id := book.Photos[2].ID; id != 8 {
		t.Errorf("new photo ID %d, want 8", id)
	}

	// The CHECK on the disposition knows the English values only.
	if _, err := s.EditItem(ctx, 1, ItemEdit{Name: "Regal", Disposition: Disposition("verkaufen")}, now); err == nil {
		t.Error("German disposition was accepted by the database")
	}

	// Foreign keys are on again after migrating.
	if err := s.DeletePerson(ctx, 2); err != ErrHasUserAccess {
		t.Errorf("delete user: %v", err)
	}
	var fk int
	if err := s.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d, %v", fk, err)
	}
	rows, err := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Error("foreign key violations after migrating")
	}
	rows.Close()

	// References in other tables point to the renamed tables, not to the German names.
	var german int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_schema
		WHERE sql LIKE '%nutzer%' OR sql LIKE '%wohnung%' OR sql LIKE '%raum%' OR sql LIKE '%lagerort%' OR sql LIKE '%umzug%'`).Scan(&german); err != nil || german != 0 {
		t.Errorf("%d schema entries still use German names, %v", german, err)
	}
}
