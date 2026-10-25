# Handover

Stand: 2026-10-24 abends. `main` = `ee338b8` plus dieser Commit, gepusht nach `github.com/layornos/Wohnungsinventar`, deployt auf dem Pi.

## Was heute passiert ist

- **Feedback-Runde 1** (Tickets 01–04, 06) umgesetzt, vom Nutzer getestet und erledigt. Das Seitenmenü öffnet nur noch über ☰ in der unteren Leiste; der Rand-Wisch kollidierte mit der Zurück-Geste von iOS.
- **Drei Designs** Minimal, Neon und Weich & farbig, gewählt unter „Einstellungen“ und am Nutzer gespeichert. Dazu eine Desktop-Ansicht mit Seitenleiste und eine feste untere Leiste auf dem Handy, auf der Startseite mit ☰, Suche und +.
- **Code komplett auf Englisch** (Regel in `CLAUDE.md`, Code-Namen in `CONTEXT.md`, Zuordnung in `.scratch/englischer-code/mapping.md`). Die Oberfläche bleibt deutsch.
- **Betrieb auf dem Pi** (ADR-0005): App als systemd-Dienst hinter einem gemeinsamen Caddy, nächtliches Backup aufs NAS.
- **Spec 09 Packen und Auspacken:** Box mit Zielraum, „Alles einpacken“ je Lagerort, Möbel nehmen ihren Inhalt mit, Auspack-Modus mit „Alle hier einräumen“, Etiketten mit QR-Code und NFC.
- **Feedback-Runde 2:**
  - Serienmodus beim Erfassen und kleinere Fotos
  - Mehrfachauswahl auf der Startseite
  - Hinweis bei neuer Version
  - eingebettete Schriften
  - Etiketten für Lagerorte mit „Hier einräumen“
  - KI-Vorschläge für Name und Beschreibung (Claude Haiku 5.5, ADR-0006)

## Läuft noch / offen

1. **Browser-Tests und CI** (`.scratch/feedback-2/issues/05-browser-tests-und-ci.md`): Ein Subagent arbeitet daran (Playwright-Tests unter `e2e/` und `.github/workflows/ci.yml`). Noch nicht gemergt. Nach dem Mergen die Tests an die zuletzt gemergten Änderungen anpassen und prüfen, ob die Action auf GitHub grün ist.
2. **Vom Nutzer auf dem iPhone zu testen** (alles ist deployt):
   - Serienmodus: Nach „Speichern“ ein Tipp bis zum nächsten Foto. Kommt ein echtes Handyfoto mit weniger als 1 MB an?
   - KI-Vorschlag beim ersten Foto. Mit einem echten Gegenstand noch nie probiert, die Pipeline funktioniert aber (3,4 s, Haiku 5.5). Taugt die Qualität nicht, auf dem Pi in `~/inventar/inventar.env` `INVENTAR_ANTHROPIC_MODEL=claude-sonnet-5-5` setzen und `sudo systemctl restart inventar`.
   - Mehrfachauswahl per langem Drücken.
   - Erscheint „Neue Version verfügbar“ nach dem nächsten Deploy?
   - Packen, Etikett drucken, QR scannen, Auspacken. QR-Code und NFC öffnen unter iOS Safari, nicht die installierte App; dort einmal separat anmelden.
   - Lagerort-Etikett und „Hier einräumen“.
3. **Ticket 07 Styles aufräumen** (`.scratch/feedback-1/issues/07-styles-und-designliste-aufraeumen.md`): `app.css` hat inzwischen etwa 2000 Zeilen. Erst angehen, wenn keine anderen Subagenten mehr an CSS arbeiten. Dazu gehören:
   - Klasse `box-label-sheet` in `label-sheet` umbenennen
   - die Etikettseite der Box auf `LabelActions.svelte` umstellen
   - doppelter Knopf „Foto aufnehmen“ auf dem Handy
   - Leerzeichen vor „›“ in `PathView`
   - doppelte Rahmen und Trennlinien in den Listen beim Packen und Auspacken
   - `tmp/` in `.gitignore`
4. **Erfassen im Funkloch** (Offline-Warteschlange für neue Items): bewusst zurückgestellt, noch kein Ticket.
5. **Specs 06 Verkauf, 07 Erinnerungen und 08 Preiserinnerung:** am 2026-10-25 umgesetzt und deployt (Migrationen 0010 bis 0012, jeweils Backup vorher; `INVENTAR_PUBLIC_URL` auf dem Pi gesetzt). 0012 hat auch die Termine für Items nachgeholt, die vor Spec 07 schon Behalten, Später verkaufen, Verkaufen (mit Preis) oder Verschenken hatten. Monatsmail am 1., über dasselbe Gmail-Konto wie die NAS-Backup-Info (Pi-Ticket 06).
6. **Pi-Tickets:** 04 (WireGuard) und 06 (Mailversand) erledigt.
7. **Feedback 05** (Verbleib nicht übernommen): beobachten, bis es noch einmal auftritt.

## Betrieb

- **Pi:** `ssh layornos@kodi` (192.168.178.87, Debian 12 arm64), `sudo` ohne Passwort.
- **App:**
  - Adresse: `https://mosiminventar.duckdns.org`, nur im Heimnetz und über WireGuard erreichbar.
  - Programm: `~/inventar/bin/inventar`
  - Daten: `~/inventar/data`
  - Einstellungen: `~/inventar/inventar.env` (Rechte 600; darin der API-Schlüssel, nie ausgeben)
- **Dienste:**
  - `inventar.service`
  - `inventar-backup.timer`: nachts nach `/mnt/NAS/Backup/inventar`, zusätzlich sonntags das System-Backup der NVMe.
- **Deploy:** `deploy/deploy.sh` (baut, lädt hoch, startet neu). Vor Deploys mit Migration `sudo systemctl start inventar-backup.service`.
- **Proxy:** `~/Projects/pi-proxy` (lokales Git-Repo, Caddy im Host-Netz, DuckDNS DNS-01). Er bedient auch `mosimfilme.duckdns.org` (Filmdatenbank, App auf `127.0.0.1:3000`).
- Die Docker-Volumes `docker_caddy_data` und `docker_caddy_config` auf dem Pi sind verwaist, aber nicht gelöscht. Erst nach Rückfrage löschen.

## Arbeitsweise, die sich bewährt hat

- **Subagenten:** Unabhängiges parallel bauen, jede Aufgabe in einem eigenen Worktree. Vorher den API-Vertrag zwischen Server und Client in die Spec schreiben, dann können beide Seiten gleichzeitig loslegen.
- **Konflikte:** Neue CSS-Blöcke kommen jeweils als eigener Block ans Dateiende. Konflikte in `app.css` lassen sich dann einfach durch „beide behalten“ lösen.
- **Vor jedem Deploy:** `make check`.
- **Größere UI-Änderungen:** Bildschirmfotos über Headless Chrome machen. Skripte liegen im Scratchpad der Sitzung; künftig die Playwright-Tests nutzen.
