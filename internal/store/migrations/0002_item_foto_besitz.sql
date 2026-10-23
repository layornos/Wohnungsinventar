-- Verbleib NULL heißt offen. name_klein ist der kleingeschriebene Name für die Suche,
-- weil LOWER in SQLite nur ASCII kennt.
CREATE TABLE item (
    id           INTEGER PRIMARY KEY,
    name         TEXT NOT NULL,
    name_klein   TEXT NOT NULL,
    beschreibung TEXT NOT NULL DEFAULT '',
    verbleib     TEXT CHECK (verbleib IN ('behalten', 'verkaufen', 'spaeter_verkaufen', 'verschenken')),
    erstellt_am  TEXT NOT NULL,
    geaendert_am TEXT NOT NULL
);

CREATE INDEX item_geaendert ON item (geaendert_am);

-- Die Datei selbst liegt im Foto-Ordner (ADR-0002); datei ist ihr zufälliger Name.
CREATE TABLE foto (
    id          INTEGER PRIMARY KEY,
    item_id     INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    datei       TEXT NOT NULL UNIQUE,
    typ         TEXT NOT NULL,
    hauptfoto   INTEGER NOT NULL DEFAULT 0,
    reihenfolge INTEGER NOT NULL
);

CREATE INDEX foto_item ON foto (item_id, reihenfolge);
CREATE UNIQUE INDEX foto_ein_hauptfoto ON foto (item_id) WHERE hauptfoto = 1;

CREATE TABLE besitz (
    item_id   INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    person_id INTEGER NOT NULL REFERENCES person (id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, person_id)
);

CREATE INDEX besitz_person ON besitz (person_id);
