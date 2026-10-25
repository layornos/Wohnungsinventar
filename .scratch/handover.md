# Handover

Stand: 2026-10-24 abends, Abschnitt „Läuft noch / offen“ am 2026-10-25 aktualisiert. `main` = `ee338b8` plus dieser Commit, gepusht nach `github.com/layornos/Wohnungsinventar`, deployt auf dem Pi.

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

Stand 2026-10-25 mittags: Alle Specs und Tickets sind umgesetzt und deployt.

1. **Erledigt am 2026-10-25:**
   - Browser-Tests und CI (feedback-2/05): Playwright unter `e2e/`, GitHub Action grün.
   - Ticket 07 Styles aufräumen: je Design eine Datei unter `web/src/themes/`, Designliste nur in `web/src/lib/themes.ts`.
   - Specs 06 Verkauf, 07 Erinnerungen und 08 Preiserinnerung (Migrationen 0010 bis 0012, jeweils Backup vorher; `INVENTAR_PUBLIC_URL` auf dem Pi gesetzt). 0012 hat auch die Termine für Items nachgeholt, die vor Spec 07 schon Behalten, Später verkaufen, Verkaufen (mit Preis) oder Verschenken hatten.
   - Pi-Tickets 04 (WireGuard von unterwegs) und 06 (Mailversand über dasselbe Gmail-Konto wie die NAS-Backup-Info, Test-Mail kam an).
   - Die Tests auf dem iPhone aus der vorigen Übergabe hat der Nutzer bestätigt.
2. **Zu beobachten:**
   - Erste Monatsmail am 1. eines Monats, sobald Erinnerungen offen sind (frühestens 30 Tage nach dem ersten Verkaufs- oder Verschenk-Item).
   - Feedback 05 (Verbleib nicht übernommen): beobachten, bis es noch einmal auftritt.
3. **Ideen ohne Ticket:** Erfassen im Funkloch (Offline-Warteschlange für neue Items), bewusst zurückgestellt.

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
