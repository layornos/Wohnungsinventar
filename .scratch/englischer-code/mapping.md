# Code auf Englisch umstellen: verbindliche Zuordnung

Status: ready-for-agent

Regel und Fachbegriffe: `CLAUDE.md` (Abschnitt „Code language“) und `CONTEXT.md` (Abschnitt „Code names“). Diese Datei legt den **Vertrag zwischen Server und Client** fest, damit beide Seiten parallel umbenannt werden können. Abweichungen nur nach Rücksprache.

Die Oberfläche bleibt deutsch. Der Binärname `inventar` und das Präfix `INVENTAR_` der Umgebungsvariablen bleiben (Produktname). Das Datenverzeichnis behält seine Struktur (Ordner `fotos` usw.), damit keine Dateien verschoben werden müssen; nur die Konstanten im Code heißen englisch.

## API-Pfade

| Alt | Neu |
|---|---|
| `POST /api/anmeldung` | `POST /api/login` |
| `POST /api/abmeldung` | `POST /api/logout` |
| `GET /api/ich` | `GET /api/me` |
| `PUT /api/ich/design` | `PUT /api/me/theme` |
| `/api/personen`, `/api/personen/{id}` | `/api/persons`, `/api/persons/{id}` |
| `/api/items/{id}/besitzer/{personId}` | `/api/items/{id}/owners/{personId}` |
| `POST /api/items/{id}/fotos` | `POST /api/items/{id}/photos` |
| `PUT /api/items/{id}/hauptfoto` | `PUT /api/items/{id}/main-photo` |
| `GET /api/fotos/{id}/{datei}` | `GET /api/photos/{id}/{file}` |
| `GET /api/fotos/{id}/{datei}/vorschau` | `GET /api/photos/{id}/{file}/thumbnail` |
| `PUT/DELETE /api/fotos/{id}` | `PUT/DELETE /api/photos/{id}` |
| `/api/wohnungen`, `/api/wohnungen/{id}` | `/api/apartments`, `/api/apartments/{id}` |
| `POST /api/wohnungen/{id}/raeume` | `POST /api/apartments/{id}/rooms` |
| `/api/raeume/{id}` | `/api/rooms/{id}` |
| `GET /api/plaetze` | `GET /api/places` |
| `/api/lagerorte`, `/api/lagerorte/{id}` | `/api/storage-locations`, `/api/storage-locations/{id}` |
| `PUT /api/lagerorte/{id}/eltern` | `PUT /api/storage-locations/{id}/parent` |
| `GET /api/lagerorte/{id}/wegfall` | `GET /api/storage-locations/{id}/removal` |
| `PUT /api/lagerorte/{id}/moebel` | `PUT /api/storage-locations/{id}/furniture` |
| `GET /api/items/{id}/wegfall` | `GET /api/items/{id}/removal` |
| `PUT /api/items/{id}/platzierung` | `PUT /api/items/{id}/placement` |
| `PATCH /api/items/{id}/umzug` | `PATCH /api/items/{id}/move` |
| `GET/POST /api/umzug` | `GET/POST /api/move` |
| `POST /api/umzug/boxen` | `POST /api/move/boxes` |
| `GET /api/umzug/boxen/{nummer}` | `GET /api/move/boxes/{number}` |
| `GET /api/umzug/zielraeume` | `GET /api/move/target-rooms` |
| `POST /api/umzug/abschluss` | `POST /api/move/completion` |
| `POST /api/umzug/sammeln` | `POST /api/move/bulk` |
| `/api/boxen/{id}` | `/api/boxes/{id}` |
| `/api/umzuege`, `/api/umzuege/{id}` | `/api/moves`, `/api/moves/{id}` |
| `GET /api/nutzer` | `GET /api/users` |
| `/api/einladungen`, `/api/einladungen/{id}` | `/api/invitations`, `/api/invitations/{id}` |
| `POST /api/einladungen/pruefen` | `POST /api/invitations/check` |
| `POST /api/einladungen/annehmen` | `POST /api/invitations/accept` |

Unveränderte Pfade (`/api/items`, `/api/items/{id}`) bleiben.

## Query-Parameter und Formularfelder

- Item-Liste: `q` bleibt; `besitzer` → `owner`, `verbleib` → `disposition` (Wert `offen` → `undecided`), `raum` → `room`, `lagerort` → `storageLocation`, `box` bleibt, `seite` → `page`.
- Multipart-Feld `foto`/`fotos` → `photo`/`photos` (gleiche Ein-/Mehrzahl wie bisher).
- Weitere Query-Parameter nach demselben Muster in camelCase; bitte im Bericht auflisten.

## JSON-Felder

| Alt | Neu | | Alt | Neu |
|---|---|---|---|---|
| `abgeschlossenAm` | `completedAt` | | `meldung` | `message` |
| `alteWohnung` | `oldApartment` | | `moebel` | `furniture` |
| `alteWohnungId` | `oldApartmentId` | | `moebelLagerorte` | `furnitureStorageLocations` |
| `anzahl` | `count` | | `neueWohnung` | `newApartment` |
| `art` | `kind` | | `neueWohnungId` | `newApartmentId` |
| `beschreibung` | `description` | | `nummer` | `number` |
| `besitzer` | `owners` | | `nutzer` (Person, bool) | `isUser` |
| `bestaetigt` | `confirmed` | | `nutzer` (sonst) | `user` / `users` |
| `bezeichnung` | `label` | | `passwort` | `password` |
| `boxen` | `boxes` | | `pfad` | `path` |
| `design` | `theme` | | `platz` | `place` |
| `elternId` | `parentId` | | `platzierung` | `placement` |
| `ersteller` | `createdBy` | | `raeume` | `rooms` |
| `erstelltAm` | `createdAt` | | `raum` | `room` |
| `fotoId` | `photoId` | | `raumId` | `roomId` |
| `fotos` | `photos` | | `umzug` | `move` |
| `geaendertAm` | `updatedAt` | | `unter` | `children` |
| `hauptfoto` | `mainPhoto` | | `unterbaum` | `subtree` |
| `herkunftsraum` | `originRoom` | | `verbleib` | `disposition` |
| `laeuftAbAm` | `expiresAt` | | `vorschauUrl` | `thumbnailUrl` |
| `lagerorte` | `storageLocations` | | `wohnung` | `apartment` |
| `lagerortId` | `storageLocationId` | | `zielraum` | `targetRoom` |
| `laufend` | `active` | | | |

Unverändert: `id`, `itemId`, `items`, `personId`, `name`, `email`, `token`, `link`, `url`, `code`, `box`.

## Werte

- Verbleib: `behalten` → `keep`, `verkaufen` → `sell`, `spaeter_verkaufen` → `sell_later`, `verschenken` → `give_away`.
- Platz-Art (`kind`): `raum` → `room`, `lagerort` → `storageLocation`.
- Design (`theme`): `minimal`, `neon`, `weich` → `soft`.
- Fehlercodes:

| Alt | Neu | | Alt | Neu |
|---|---|---|---|---|
| `anmeldung_fehlgeschlagen` | `login_failed` | | `name_fehlt` | `name_missing` |
| `bereits_nutzer` | `already_user` | | `nicht_angemeldet` | `not_logged_in` |
| `besitzer_fehlt` | `owner_missing` | | `nicht_gefunden` | `not_found` |
| `box_nicht_leer` | `box_not_empty` | | `passwort_zu_kurz` | `password_too_short` |
| `einladung_abgelaufen` | `invitation_expired` | | `raum_im_umzug` | `room_in_move` |
| `einladung_ungueltig` | `invitation_invalid` | | `raum_nicht_leer` | `room_not_empty` |
| `einladung_verwendet` | `invitation_used` | | `umzug_abgeschlossen` | `move_completed` |
| `einladung_zurueckgezogen` | `invitation_revoked` | | `umzug_laeuft` | `move_in_progress` |
| `email_ungueltig` | `email_invalid` | | `unbekannte_box` | `unknown_box` |
| `email_vergeben` | `email_taken` | | `unbekannte_person` | `unknown_person` |
| `falscher_dateityp` | `wrong_file_type` | | `unbekannte_wohnung` | `unknown_apartment` |
| `foto_fehlt` | `photo_missing` | | `unbekannter_platz` | `unknown_place` |
| `gleiche_wohnung` | `same_apartment` | | `unbekanntes_design` | `unknown_theme` |
| `hat_nutzerzugang` | `has_user_access` | | `unbekanntes_item` | `unknown_item` |
| `items_ohne_zielraum` | `items_without_target_room` | | `ungueltige_anfrage` | `invalid_request` |
| `kein_umzug` | `no_move` | | `ungueltiger_platz` | `invalid_place` |
| `letzter_besitzer` | `last_owner` | | `ungueltiger_verbleib` | `invalid_disposition` |
| `letztes_foto` | `last_photo` | | `wohnung_im_umzug` | `apartment_in_move` |
| `moebel_in_sich_selbst` | `furniture_inside_itself` | | `wohnung_nicht_leer` | `apartment_not_empty` |
| `zielraum_nicht_in_neuer_wohnung` | `target_room_not_in_new_apartment` | | `zu_gross` | `too_large` |
| `zu_viele_fotos` | `too_many_photos` | | | |

Die deutschen Fehlermeldungen (`message`) bleiben deutsch, sie werden angezeigt.

## Sonstiges

- Sitzungs-Cookie `sitzung` → `session` (alle sind danach einmal abgemeldet).
- CLI: `starten` → `serve`, `nutzer-anlegen` → `create-user`, `passwort-zuruecksetzen` → `reset-password`, `hilfe` entfällt (`help`, `-h`, `--help` bleiben). Flags `--name`, `--email` bleiben.
- Umgebungsvariable `INVENTAR_MAX_FOTO_MB` → `INVENTAR_MAX_PHOTO_MB`.
- Datenbank: neue Migration benennt Tabellen und Spalten um (SQLite `ALTER TABLE … RENAME`) und setzt gespeicherte Werte um (Verbleib, Design `weich` → `soft`). Bestehende Daten bleiben erhalten. Bereits angewandte Migrationen werden inhaltlich nicht geändert.
- Client: `<html data-design>` → `data-theme`, localStorage-Schlüssel `inventar.design` → `inventar.theme` (alten Wert einmal übernehmen), Routen der Oberfläche (z. B. `/einstellungen`, `/wohnungen`) bleiben deutsch, weil Nutzer sie sehen.
