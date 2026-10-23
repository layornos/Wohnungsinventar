-- Höchstens ein Umzug läuft; letzte_boxnummer sorgt dafür, dass Nummern nie wiederverwendet werden.
CREATE TABLE umzug (
    id               INTEGER PRIMARY KEY,
    alte_wohnung_id  INTEGER NOT NULL REFERENCES wohnung (id),
    neue_wohnung_id  INTEGER NOT NULL REFERENCES wohnung (id),
    laufend          INTEGER NOT NULL DEFAULT 1,
    letzte_boxnummer INTEGER NOT NULL DEFAULT 0,
    erstellt_am      TEXT NOT NULL,
    abgeschlossen_am TEXT,
    CHECK (alte_wohnung_id <> neue_wohnung_id)
);

CREATE UNIQUE INDEX umzug_ein_laufender ON umzug (laufend) WHERE laufend = 1;

CREATE TABLE box (
    id          INTEGER PRIMARY KEY,
    umzug_id    INTEGER NOT NULL REFERENCES umzug (id),
    nummer      INTEGER NOT NULL,
    bezeichnung TEXT NOT NULL DEFAULT '',
    erstellt_am TEXT NOT NULL,
    UNIQUE (umzug_id, nummer)
);

-- Umzugsangaben eines Items, höchstens einmal je Umzug. Nach dem Abschluss bleiben sie als Verlauf.
CREATE TABLE umzugsangabe (
    item_id          INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    umzug_id         INTEGER NOT NULL REFERENCES umzug (id),
    herkunftsraum_id INTEGER REFERENCES raum (id),
    zielraum_id      INTEGER REFERENCES raum (id),
    box_id           INTEGER REFERENCES box (id),
    PRIMARY KEY (item_id, umzug_id)
);

CREATE INDEX umzugsangabe_box ON umzugsangabe (box_id);
CREATE INDEX umzugsangabe_umzug ON umzugsangabe (umzug_id, zielraum_id);
