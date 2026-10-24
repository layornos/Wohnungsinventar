-- English names for all tables, columns and indexes, and English values for stored enums
-- (disposition, theme). Existing data is kept; the migration runs with foreign keys off
-- and is checked with foreign_key_check before commit (see Store.migrate).

-- Indexes cannot be renamed; they are dropped here and recreated under English names below.
DROP INDEX sitzung_person;
DROP INDEX item_geaendert;
DROP INDEX item_lagerort;
DROP INDEX item_raum;
DROP INDEX foto_item;
DROP INDEX foto_ein_hauptfoto;
DROP INDEX besitz_person;
DROP INDEX raum_wohnung;
DROP INDEX lagerort_raum;
DROP INDEX lagerort_eltern;
DROP INDEX lagerort_moebel;
DROP INDEX umzug_ein_laufender;
DROP INDEX umzugsangabe_box;
DROP INDEX umzugsangabe_umzug;
DROP INDEX einladung_person;

ALTER TABLE person RENAME COLUMN erstellt_am TO created_at;

-- "user" is reserved in PostgreSQL, hence app_user.
ALTER TABLE nutzer RENAME TO app_user;
ALTER TABLE app_user RENAME COLUMN passwort_hash TO password_hash;
ALTER TABLE app_user RENAME COLUMN design TO theme;
UPDATE app_user SET theme = 'soft' WHERE theme = 'weich';

ALTER TABLE sitzung RENAME TO session;
ALTER TABLE session RENAME COLUMN erstellt_am TO created_at;
ALTER TABLE session RENAME COLUMN laeuft_ab_am TO expires_at;

-- The CHECK on the disposition lists the stored values, so the item table is rebuilt.
-- Tables referencing item keep pointing to it by name.
CREATE TABLE item_new (
    id                  INTEGER PRIMARY KEY,
    name                TEXT NOT NULL,
    name_lower          TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    disposition         TEXT CHECK (disposition IN ('keep', 'sell', 'sell_later', 'give_away')),
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    storage_location_id INTEGER REFERENCES storage_location (id),
    room_id             INTEGER REFERENCES room (id)
);

INSERT INTO item_new (id, name, name_lower, description, disposition, created_at, updated_at, storage_location_id, room_id)
    SELECT id, name, name_klein, beschreibung,
        CASE verbleib
            WHEN 'behalten' THEN 'keep'
            WHEN 'verkaufen' THEN 'sell'
            WHEN 'spaeter_verkaufen' THEN 'sell_later'
            WHEN 'verschenken' THEN 'give_away'
        END,
        erstellt_am, geaendert_am, lagerort_id, raum_id
    FROM item;

DROP TABLE item;
ALTER TABLE item_new RENAME TO item;

ALTER TABLE foto RENAME TO photo;
ALTER TABLE photo RENAME COLUMN datei TO file;
ALTER TABLE photo RENAME COLUMN typ TO content_type;
ALTER TABLE photo RENAME COLUMN hauptfoto TO main_photo;
ALTER TABLE photo RENAME COLUMN reihenfolge TO sort_order;

ALTER TABLE besitz RENAME TO ownership;

ALTER TABLE wohnung RENAME TO apartment;
ALTER TABLE apartment RENAME COLUMN erstellt_am TO created_at;

ALTER TABLE raum RENAME TO room;
ALTER TABLE room RENAME COLUMN wohnung_id TO apartment_id;
ALTER TABLE room RENAME COLUMN erstellt_am TO created_at;

ALTER TABLE lagerort RENAME TO storage_location;
ALTER TABLE storage_location RENAME COLUMN raum_id TO room_id;
ALTER TABLE storage_location RENAME COLUMN eltern_id TO parent_id;
ALTER TABLE storage_location RENAME COLUMN moebel_item_id TO furniture_item_id;
ALTER TABLE storage_location RENAME COLUMN erstellt_am TO created_at;

ALTER TABLE umzug RENAME TO move;
ALTER TABLE move RENAME COLUMN alte_wohnung_id TO old_apartment_id;
ALTER TABLE move RENAME COLUMN neue_wohnung_id TO new_apartment_id;
ALTER TABLE move RENAME COLUMN laufend TO active;
ALTER TABLE move RENAME COLUMN letzte_boxnummer TO last_box_number;
ALTER TABLE move RENAME COLUMN erstellt_am TO created_at;
ALTER TABLE move RENAME COLUMN abgeschlossen_am TO completed_at;

ALTER TABLE box RENAME COLUMN umzug_id TO move_id;
ALTER TABLE box RENAME COLUMN nummer TO number;
ALTER TABLE box RENAME COLUMN bezeichnung TO label;
ALTER TABLE box RENAME COLUMN erstellt_am TO created_at;

-- The move details of an item (origin room, target room, box) in one move.
ALTER TABLE umzugsangabe RENAME TO item_move;
ALTER TABLE item_move RENAME COLUMN umzug_id TO move_id;
ALTER TABLE item_move RENAME COLUMN herkunftsraum_id TO origin_room_id;
ALTER TABLE item_move RENAME COLUMN zielraum_id TO target_room_id;

ALTER TABLE einladung RENAME TO invitation;
ALTER TABLE invitation RENAME COLUMN ersteller_id TO created_by_id;
ALTER TABLE invitation RENAME COLUMN erstellt_am TO created_at;
ALTER TABLE invitation RENAME COLUMN laeuft_ab_am TO expires_at;
ALTER TABLE invitation RENAME COLUMN angenommen_am TO accepted_at;
ALTER TABLE invitation RENAME COLUMN zurueckgezogen_am TO revoked_at;

CREATE INDEX session_person ON session (person_id);
CREATE INDEX item_updated ON item (updated_at);
CREATE INDEX item_storage_location ON item (storage_location_id);
CREATE INDEX item_room ON item (room_id);
CREATE INDEX photo_item ON photo (item_id, sort_order);
CREATE UNIQUE INDEX photo_one_main_photo ON photo (item_id) WHERE main_photo = 1;
CREATE INDEX ownership_person ON ownership (person_id);
CREATE INDEX room_apartment ON room (apartment_id);
CREATE INDEX storage_location_room ON storage_location (room_id);
CREATE INDEX storage_location_parent ON storage_location (parent_id);
CREATE INDEX storage_location_furniture ON storage_location (furniture_item_id);
CREATE UNIQUE INDEX move_one_active ON move (active) WHERE active = 1;
CREATE INDEX item_move_box ON item_move (box_id);
CREATE INDEX item_move_move ON item_move (move_id, target_room_id);
CREATE INDEX invitation_person ON invitation (person_id);
