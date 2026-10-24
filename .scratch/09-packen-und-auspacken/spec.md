# Spec 09: Packen und Auspacken mit wenig Tipps

Status: ready-for-agent
Blocked by: 04-umzug-und-boxen

## Problem Statement

Beim Umzug kostet heute vor allem das Danach Zeit: Jedes Item bekommt einzeln einen Zielraum und landet beim Abschluss ohne Lagerort im Zielraum, sodass man es noch einmal öffnen und neu platzieren muss. Auch der Inhalt von Möbeln, die als Ganzes umziehen, muss so neu einsortiert werden. Packen, Beschriften und Einräumen sollen mit so wenigen Tipps wie möglich gehen.

## Solution

Eine Box bekommt beim Anlegen einen Zielraum; alles darin hat ihn automatisch. Im Pack-Modus lässt sich ein ganzer Lagerort mit einem Tipp einpacken. Möbel nehmen ihre Lagerorte samt Inhalt mit in die neue Wohnung. In der neuen Wohnung räumt man eine Box im Auspack-Modus ein: Lagerort wählen, „Alle hier einräumen“ oder einzelne Items antippen; jedes eingeräumte Item hat sofort seinen endgültigen Platz. Für jede Box gibt es ein Etikett zum Ausdrucken mit Nummer, Bezeichnung, Zielraum und QR-Code, der direkt zum Auspacken führt; derselbe Link lässt sich auf einen NFC-Tag schreiben.

## User Stories

1. Als Nutzer will ich beim Anlegen einer Box gleich ihren Zielraum wählen, damit ich ihn nicht für jedes Item einzeln setzen muss.
2. Als Nutzer will ich, dass jedes Item in einer Box deren Zielraum hat, solange es keinen eigenen hat.
3. Als Nutzer will ich einem einzelnen Item in einer Box auf seiner Detailseite trotzdem einen anderen Zielraum geben können.
4. Als Nutzer will ich den Zielraum und die Bezeichnung einer Box ändern können.
5. Als Nutzer will ich im Pack-Modus einen ganzen Lagerort (samt Unterlagerorten) mit einem Tipp in die aktive Box packen.
6. Als Nutzer will ich, dass der Pack-Modus sich den zuletzt gewählten Herkunftsraum merkt.
7. Als Nutzer will ich, dass ein Möbelstück mit Zielraum seine Lagerorte samt Inhalt mitnimmt, damit dieser Inhalt danach nicht neu platziert werden muss.
8. Als Nutzer will ich eine Box über ihre Nummer, den QR-Code oder einen NFC-Tag öffnen und auspacken.
9. Als Nutzer will ich beim Auspacken einmal einen Lagerort oder Raum der neuen Wohnung wählen und dann die ganze Box mit einem Tipp dort einräumen.
10. Als Nutzer will ich beim Auspacken einzelne Items antippen, die dann am gewählten Platz landen, und den Platz für den Rest wechseln.
11. Als Nutzer will ich beim Auspacken einen fehlenden Lagerort direkt anlegen (wie in der Platzauswahl, Feedback 03).
12. Als Nutzer will ich, dass ein eingeräumtes Item sofort seinen endgültigen Platz hat und nicht mehr in der Box steht, auch wenn der Umzug noch läuft.
13. Als Nutzer will ich für jede Box ein Etikett drucken: große Nummer, Bezeichnung (z. B. „Box A“), Zielraum und QR-Code.
14. Als Nutzer will ich den Auspack-Link einer Box kopieren, um ihn auf einen NFC-Tag zu schreiben.
15. Als Nutzer will ich, dass der Abschluss des Umzugs bereits eingeräumte Items und Möbelinhalte nicht anfasst und nur den Rest wie bisher in den Zielraum stellt.

## Implementation Decisions

- **Begriffe** wie in `CONTEXT.md`; Code-Namen siehe Abschnitt „Code names“.
- **Zielraum der Box:** optional, muss in der neuen Wohnung liegen. Der **wirksame Zielraum** eines Items ist sein eigener Zielraum, sonst der Zielraum seiner Box, sonst keiner. „Items der alten Wohnung ohne Zielraum“ und der Abschluss rechnen mit dem wirksamen Zielraum.
- **Lagerort einpacken:** packt alle Items, die in diesem Lagerort oder einem seiner Unterlagerorte platziert sind, in die Box. Das Möbelstück selbst, zu dem der Lagerort gehört, wird nicht mitgepackt.
- **Möbel nehmen ihren Inhalt mit:** Wird ein Möbelstück in die neue Wohnung gebracht (beim Auspacken/Einräumen in einen Raum der neuen Wohnung oder beim Abschluss über seinen wirksamen Zielraum), wandern seine Lagerorte in diesen Raum. Items darin, die weder einen eigenen Zielraum haben noch in einer Box stecken, behalten ihre Platzierung und zählen nicht als „ohne Zielraum“.
- **Einräumen (Auspacken):** setzt für die gewählten Items sofort die Platzierung auf einen Lagerort oder Raum der **neuen Wohnung**, nimmt sie aus der Box und setzt ihren Zielraum auf den Raum der neuen Platzierung. Nur während eines laufenden Umzugs; nur Items dieser Box.
- **Abschluss:** Items, deren Platzierung bereits in der neuen Wohnung liegt, bleiben unverändert. Möbel mit wirksamem Zielraum nehmen ihre Lagerorte mit (siehe oben). Alle übrigen Items mit wirksamem Zielraum stehen danach wie bisher ohne Lagerort im Zielraum.
- **Etikett und QR-Code:** Der Server erzeugt den QR-Code als SVG (Go-Bibliothek, keine Abhängigkeit im Client). Inhalt ist der Auspack-Link der Box, gebaut wie der Einladungslink aus der Adresse, unter der die App gerade aufgerufen ist (inkl. Proxy-Modus aus ADR-0005). Die Druckansicht ist eine eigene Seite mit Druck-CSS (A6/A5-tauglich, schwarz auf weiß, unabhängig vom Design).
- **NFC:** iPhones (ab XS) und Android lesen NFC-Tags mit einer URL ohne App. Beschreiben lässt sich ein Tag aus einer Web-App heraus nur unter Android (Chrome, Web NFC); unter iOS mit einer App wie „NFC Tools“. Die App bietet deshalb „Link kopieren“ und, wo `NDEFReader` verfügbar ist, „Auf NFC-Tag schreiben“.
- **Hinweis iOS:** QR-Code und NFC-Tag öffnen den Link in Safari, nicht in der installierten App; dort ist man einmal separat angemeldet.

### API (Vertrag zwischen Server und Client)

- `POST /api/move/boxes` Body `{ "label": string, "targetRoomId": number|null }` → Box.
- `PUT /api/boxes/{id}` Body `{ "label": string, "targetRoomId": number|null }` → Box. (Bisher nur `label`.)
- Box-JSON (`boxJSON`, auch in `GET /api/move`, `GET /api/boxes/{id}`, `GET /api/move/boxes/{number}`, Übersicht je Zielraum) und `boxSummaryJSON` bekommen `"targetRoom": {id,name}|null`.
- Umzugsangaben eines Items (`move` im Item und in Listeneinträgen): `targetRoom` ist der **wirksame** Zielraum; neu `"targetRoomFromBox": boolean` (true, wenn er von der Box kommt). Setzen per `PATCH /api/items/{id}/move` setzt wie bisher den eigenen Zielraum (`targetRoomId: null` entfernt ihn, dann gilt wieder der der Box).
- `POST /api/boxes/{id}/pack` Body `{ "storageLocationId": number }` → `{ "count": number, "box": Box }`. Fehler: `unknown_place`, `move_completed`, `no_move`.
- `POST /api/boxes/{id}/unpack` Body `{ "itemIds": number[] | null, "storageLocationId": number|null, "roomId": number|null }` – genau eins von `storageLocationId`/`roomId`; `itemIds: null` heißt alle Items der Box → `{ "count": number, "box": Box }`. Fehler: `place_not_in_new_apartment`, `invalid_place`, `unknown_item` (Item nicht in dieser Box), `no_move`.
- `GET /api/boxes/{id}/qr` → `image/svg+xml`, QR-Code des Auspack-Links `<scheme>://<host>/boxen/{id}/auspacken`.
- Die Packliste einer Box (`GET /api/boxes/{id}`) enthält je Item zusätzlich `"previousPlace": string` (Pfad der bisherigen Platzierung, z. B. „Arbeitszimmer › Schreibtisch › oben“), damit der Auspack-Modus anzeigen kann, woher es kam.

### Client

- **Pack-Modus** (`/umzug/packen`): „Neue Box“ fragt Bezeichnung (optional) und Zielraum in einem kleinen Dialog. Items sind nach Lagerort gruppiert; jede Gruppe hat „Alles einpacken“. Der zuletzt gewählte Herkunftsraum wird gemerkt (localStorage). Box-Auswahl zeigt „Nr · Bezeichnung → Zielraum“.
- **Box-Seite** (`/boxen/[id]`): Zielraum und Bezeichnung ändern, Knöpfe „Auspacken“ und „Etikett“.
- **Auspack-Modus** (`/boxen/[id]/auspacken`): oben der gewählte Platz (Platzauswahl mit Anlegen, vorbelegt mit dem Zielraum der Box, nur Plätze der neuen Wohnung), darunter die Items mit Foto und „kam aus …“; Antippen räumt ein Item ein; großer Knopf „Alle hier einräumen“. Leere Box: Hinweis „Box ist leer“ und Link zur nächsten Box.
- **Box per Nummer:** vorhandenes Nummernfeld führt zum Auspack-Modus, sobald der Umzug in der neuen Wohnung angekommen ist (einfach: Box-Seite mit großem „Auspacken“-Knopf).
- **Etikett** (`/boxen/[id]/etikett`): Druckansicht mit Nummer (sehr groß), Bezeichnung, „→ Zielraum“, QR-Code; Knopf „Drucken“ (`window.print()`), „Link kopieren“, „Auf NFC-Tag schreiben“ (nur wenn `NDEFReader` existiert). Optional „Alle Etiketten drucken“ auf der Umzugsseite (eine Seite pro Box).
- **Item-Detailseite:** zeigt bei einem Item in einer Box „Zielraum: Küche (von Box 7)“ und erlaubt, einen eigenen Zielraum zu setzen oder zu entfernen.

## Testing Decisions

- Test-Seam ist die HTTP-API (wie in Spec 04). Gute Tests prüfen beobachtbares Verhalten, z. B.:
  - Box mit Zielraum Küche: ein hineingepacktes Item hat den wirksamen Zielraum Küche und `targetRoomFromBox: true`; mit eigenem Zielraum Bad gilt Bad.
  - Lagerort „Schreibtisch › oben“ mit Unterlagerort einpacken packt alle Items darin und darunter, nicht den Schreibtisch selbst.
  - Schreibtisch mit Zielraum Arbeitszimmer (neu): Nach dem Abschluss liegen seine Lagerorte im neuen Arbeitszimmer, und ein Stift in der Schublade ist weiter in der Schublade platziert; er taucht nicht unter „ohne Zielraum“ auf.
  - Einräumen der ganzen Box in „Küche › Oberschrank“: alle Items sind dort platziert, die Box ist leer, der Abschluss ändert ihre Platzierung nicht.
  - Einräumen an einen Platz der alten Wohnung wird mit `place_not_in_new_apartment` abgelehnt.
  - QR-Endpunkt liefert SVG; im Proxy-Modus enthält der kodierte Link `https`.
- Client: `make check`; zusätzlich ein Durchlauf im Browser für Packen → Etikett → Auspacken.

## Out of Scope

- Vorschlag „wie vorher“ (alte Lagerortstruktur in der neuen Wohnung nachbauen).
- Transport-Status einzelner Boxen.
- Web NFC unter iOS (vom Browser nicht unterstützt).
