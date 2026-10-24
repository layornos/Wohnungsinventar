-- AUTOINCREMENT, damit SQLite die ID eines gelöschten Fotos nicht erneut vergibt. Fotos werden
-- dauerhaft im Browser zwischengespeichert; die Adresse enthält zusätzlich den Dateinamen.
-- SQLite kann AUTOINCREMENT nicht nachträglich setzen, die Tabelle wird deshalb neu angelegt.
CREATE TABLE foto_neu (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id     INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    datei       TEXT NOT NULL UNIQUE,
    typ         TEXT NOT NULL,
    hauptfoto   INTEGER NOT NULL DEFAULT 0,
    reihenfolge INTEGER NOT NULL
);

INSERT INTO foto_neu (id, item_id, datei, typ, hauptfoto, reihenfolge)
    SELECT id, item_id, datei, typ, hauptfoto, reihenfolge FROM foto;

DROP TABLE foto;
ALTER TABLE foto_neu RENAME TO foto;

CREATE INDEX foto_item ON foto (item_id, reihenfolge);
CREATE UNIQUE INDEX foto_ein_hauptfoto ON foto (item_id) WHERE hauptfoto = 1;
