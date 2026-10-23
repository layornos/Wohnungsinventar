-- Vom Einladungs-Token wird nur der Hash gespeichert. Eine Einladung gilt entweder einer
-- bestehenden Person ohne Nutzerzugang (person_id) oder einer anzulegenden Person (name).
-- Der Zustand (offen, angenommen, zurückgezogen, abgelaufen) ergibt sich aus den Zeitpunkten.
CREATE TABLE einladung (
    id                INTEGER PRIMARY KEY,
    token_hash        TEXT NOT NULL UNIQUE,
    person_id         INTEGER REFERENCES person (id) ON DELETE CASCADE,
    name              TEXT,
    ersteller_id      INTEGER NOT NULL REFERENCES nutzer (person_id),
    erstellt_am       TEXT NOT NULL,
    laeuft_ab_am      TEXT NOT NULL,
    angenommen_am     TEXT,
    zurueckgezogen_am TEXT,
    CHECK ((person_id IS NULL) <> (name IS NULL))
);

CREATE INDEX einladung_person ON einladung (person_id);
