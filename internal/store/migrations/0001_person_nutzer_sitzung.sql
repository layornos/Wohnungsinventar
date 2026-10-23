CREATE TABLE person (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    erstellt_am TEXT NOT NULL
);

-- Ein Nutzer ist eine Person mit Zugang. E-Mails werden kleingeschrieben gespeichert.
CREATE TABLE nutzer (
    person_id     INTEGER PRIMARY KEY REFERENCES person (id),
    email         TEXT NOT NULL UNIQUE,
    passwort_hash TEXT NOT NULL
);

-- Vom Sitzungs-Token wird nur der Hash gespeichert.
CREATE TABLE sitzung (
    token_hash   TEXT PRIMARY KEY,
    person_id    INTEGER NOT NULL REFERENCES nutzer (person_id) ON DELETE CASCADE,
    erstellt_am  TEXT NOT NULL,
    laeuft_ab_am TEXT NOT NULL
);

CREATE INDEX sitzung_person ON sitzung (person_id);
