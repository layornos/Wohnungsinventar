CREATE TABLE wohnung (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    erstellt_am TEXT NOT NULL
);

CREATE TABLE raum (
    id          INTEGER PRIMARY KEY,
    wohnung_id  INTEGER NOT NULL REFERENCES wohnung (id),
    name        TEXT NOT NULL,
    erstellt_am TEXT NOT NULL
);

CREATE INDEX raum_wohnung ON raum (wohnung_id);

-- Ein Lagerort liegt entweder direkt in einem Raum oder in einem anderen Lagerort.
-- Den Raum eines verschachtelten Lagerorts ergibt die Kette der Eltern.
CREATE TABLE lagerort (
    id             INTEGER PRIMARY KEY,
    name           TEXT NOT NULL,
    raum_id        INTEGER REFERENCES raum (id),
    eltern_id      INTEGER REFERENCES lagerort (id),
    moebel_item_id INTEGER REFERENCES item (id),
    erstellt_am    TEXT NOT NULL,
    CHECK ((raum_id IS NULL) <> (eltern_id IS NULL))
);

CREATE INDEX lagerort_raum ON lagerort (raum_id);
CREATE INDEX lagerort_eltern ON lagerort (eltern_id);
CREATE INDEX lagerort_moebel ON lagerort (moebel_item_id);

-- Platzierung: ein Lagerort oder ein Raum ohne Lagerort, höchstens eins von beidem.
-- SQLite kann CHECK nicht nachträglich hinzufügen; der Store setzt immer beide Spalten zusammen.
ALTER TABLE item ADD COLUMN lagerort_id INTEGER REFERENCES lagerort (id);
ALTER TABLE item ADD COLUMN raum_id INTEGER REFERENCES raum (id);

CREATE INDEX item_lagerort ON item (lagerort_id);
CREATE INDEX item_raum ON item (raum_id);
