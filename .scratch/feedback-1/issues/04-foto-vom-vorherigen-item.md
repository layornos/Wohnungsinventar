# Zweites Item zeigt das Foto des ersten

Status: ready-for-agent
Typ: Fehler

## Beobachtet

Item angelegt, Foto aufgenommen. Dann ein zweites Item angelegt und ein neues Foto aufgenommen – angezeigt wird trotzdem das Foto des ersten.

## Ursache

- Fotos werden über ihre Datenbank-ID ausgeliefert (`/api/fotos/{id}`) und dürfen ein Jahr lang unverändert im Browser-Cache bleiben (`Cache-Control: … immutable`, `internal/server/fotos.go`).
- Die Tabelle `foto` nutzt `INTEGER PRIMARY KEY` ohne `AUTOINCREMENT` (Migration 0002). SQLite vergibt die ID des zuletzt gelöschten Fotos dann erneut. Ebenso beginnen die IDs bei 1, wenn das Datenverzeichnis neu angelegt wurde.
- Bekommt ein neues Foto so eine schon einmal benutzte ID, zeigt der Browser das alte Bild aus seinem Cache.

Bestätigt durch den Nutzer: Vorher war ein Foto gelöscht worden. Das passt zur Wiederverwendung der ID.

## Lösungsidee

Die Adresse eines Fotos eindeutig machen, z. B. den zufälligen Dateinamen in die URL aufnehmen (`/api/fotos/{id}/{datei}` oder `?v=<datei>`). Zusätzlich IDs nicht wiederverwenden. Test: Foto löschen, neues anlegen, die URL muss sich unterscheiden.

## Fertig, wenn

- [ ] Neues Foto erscheint immer korrekt, Regressionstest an der HTTP-API.
