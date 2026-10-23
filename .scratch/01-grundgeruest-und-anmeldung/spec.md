# Spec 01: Grundgerüst und Anmeldung

Status: erledigt

## Problem Statement

Der Haushalt braucht eine Anwendung auf dem Raspberry Pi 5, die per Handy erreichbar ist und in der nur Nutzer des Haushalts etwas sehen oder ändern können. Bevor Items erfasst werden können, muss es einen laufenden Server mit Datenbank geben, eine installierbare App auf dem Handy und eine Anmeldung. Die Entwicklung darf dabei nicht davon abhängen, dass DuckDNS, Zertifikat oder Mailversand schon eingerichtet sind.

## Solution

Ein einzelnes Go-Programm, das eine JSON-API und eine eingebettete SvelteKit-PWA ausliefert und seine Daten in SQLite in einem konfigurierbaren Datenverzeichnis ablegt. Den ersten Nutzer legt man auf dem Pi per Kommandozeile an. Danach meldet er sich in der App mit E-Mail und Passwort an und bleibt angemeldet. Lokal läuft alles per HTTP auf `localhost`. Auf dem Pi holt der Server sich das Zertifikat über DuckDNS selbst, sobald Domain und Token konfiguriert sind.

## User Stories

1. Als erster Nutzer will ich auf dem Pi per Kommandozeile mein Konto anlegen, damit niemand anderes sich beim ersten Start als Nutzer registrieren kann.
2. Als Nutzer will ich mich mit E-Mail und Passwort anmelden, damit nur der Haushalt Zugriff hat.
3. Als Nutzer will ich angemeldet bleiben, damit ich mich auf dem Handy nicht jedes Mal neu anmelden muss.
4. Als Nutzer will ich mich abmelden, damit ein geteiltes Gerät nicht dauerhaft Zugriff hat.
5. Als Nutzer, der sein Passwort vergessen hat, will ich es auf dem Pi per Kommandozeile zurücksetzen lassen, damit ich ohne Mailversand wieder hineinkomme.
6. Als Betreiber will ich bis zur Einladungsfunktion weitere Nutzer per Kommandozeile anlegen können.
7. Als Nutzer will ich die App auf dem Startbildschirm meines Handys installieren.
8. Als Nutzer will ich die App über dieselbe Adresse im Heimnetz und über das WireGuard-VPN erreichen.
9. Als Betreiber will ich den Server als einzelnes Programm auf dem Pi starten, mit einem konfigurierbaren Datenverzeichnis auf der SSD.
10. Als Entwickler will ich den Server lokal ohne DuckDNS und ohne Zertifikat auf `localhost` starten.
11. Als Betreiber will ich, dass der Server das Zertifikat über DuckDNS selbst holt und erneuert, sobald Domain und Token konfiguriert sind.
12. Als Betreiber will ich, dass der Server ohne SMTP-Zugangsdaten startet und Mails dann nur ins Log schreibt.
13. Als Nutzer ohne Sitzung will ich auf die Anmeldeseite geleitet werden, statt eine Fehlermeldung zu sehen.

## Implementation Decisions

- **Ein Prozess:** Ein einzelnes Go-Binary liefert die JSON-API und den statisch gebauten SvelteKit-Client aus (ADR-0001). Der Client ist ins Binary eingebettet. Unbekannte Nicht-API-Pfade liefern die App-Hülle aus, damit Client-Routing funktioniert. Das Binary wird für linux/arm64 gebaut.
- **Unterbefehle des Binarys:** Server starten, Nutzer anlegen (Name, E-Mail, Passwort interaktiv oder per Eingabe) und Passwort zurücksetzen.
- **SQLite ohne cgo:** Als Treiber dient ein reiner Go-Treiber (z. B. modernc.org/sqlite), damit sich das Binary ohne C-Toolchain für den Pi bauen lässt (ADR-0002). WAL-Modus und Fremdschlüssel sind aktiviert. Schemamigrationen sind ins Binary eingebettet und laufen beim Start in aufsteigender Reihenfolge.
- **Datenzugriff hinter einer schmalen Schnittstelle**, mit Standard-SQL bevorzugt, damit ein späterer Wechsel zu PostgreSQL möglich bleibt (ADR-0002).
- **Datenverzeichnis:** konfigurierbar, enthält die SQLite-Datei und einen Ordner für Fotos, den spätere Specs nutzen.
- **Konfiguration über Umgebungsvariablen:** Datenverzeichnis, Adresse und Port, optional DuckDNS-Domain und -Token, optional SMTP-Host, -Port, -Nutzer, -Passwort und Absender. Unvollständige Gruppen, z. B. eine Domain ohne Token, führen beim Start zu einem klaren Fehler.
- **TLS:** Fehlen die DuckDNS-Angaben, läuft der Server per HTTP. Sind sie gesetzt, holt und erneuert er das Let's-Encrypt-Zertifikat per DNS-01 selbst, z. B. mit certmagic und dem libdns-DuckDNS-Provider, und lauscht per HTTPS (ADR-0004). Der Pi muss dafür von außen nicht erreichbar sein.
- **Mailer als Schnittstelle:** mit einer SMTP-Implementierung (Gmail mit App-Passwort) und einer Log-Implementierung, die genutzt wird, wenn SMTP nicht konfiguriert ist. In dieser Spec verschickt noch nichts Mails.
- **Datenmodell:** Eine **Person** hat einen Namen. Ein **Nutzer** ist eine Person mit eindeutiger E-Mail (Groß- und Kleinschreibung egal) und Passwort-Hash. Eine **Sitzung** gehört zu einem Nutzer und hat einen zufälligen Token, von dem nur der Hash gespeichert wird.
- **Authentifizierung:**
  - Passwörter werden mit argon2id gehasht.
  - Nach der Anmeldung gibt es ein HttpOnly-Cookie mit `SameSite=Strict`, das 90 Tage gilt (bei HTTPS mit `Secure`).
  - Es gibt keine offene Registrierung (ADR-0003).
  - Ohne gültige Sitzung antworten alle API-Endpunkte außer der Anmeldung mit 401.
  - Ein Passwort-Reset beendet alle Sitzungen des Nutzers.
- **API:** Endpunkte zum Anmelden, Abmelden und Abrufen des angemeldeten Nutzers. Fehler kommen als JSON mit fachlichem Code und lesbarer Meldung.
- **Client:** SvelteKit mit statischem Adapter, mobil zuerst. PWA mit Manifest (Name, Icons, `display: standalone`) und einem Service Worker, der die App-Hülle zwischenspeichert. Diese Spec enthält die Anmeldeseite, eine Startseite mit dem Namen des Nutzers und Abmelden. Der Client leitet bei 401 auf die Anmeldeseite.

## Testing Decisions

- **Eine Test-Seam, die HTTP-API:** Tests starten den echten Server-Handler mit `httptest`, mit echter SQLite-Datei in einem temporären Verzeichnis, und sprechen nur über HTTP. Interna werden weder gemockt noch direkt aufgerufen.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B. „mit falschem Passwort gibt es keine Sitzung“ oder „nach dem Abmelden liefert der Abruf des angemeldeten Nutzers 401“.
- **Test-Helfer:** Diese Spec legt das Muster für alle weiteren fest: einen Helfer, der einen Server im temporären Verzeichnis startet, und einen Helfer, der einen angemeldeten Nutzer samt Cookie-Client bereitstellt.
- **Weitere Seams:** Der Mailer wird in Tests durch eine aufzeichnende Implementierung ersetzt. Die Unterbefehle werden über ihre Funktion getestet, mit anschließender Anmeldung über die API.
- **Client:** nur Typprüfung (svelte-check) und Build.
- **Vorbild im Code:** keins, das Repo ist neu.

## Out of Scope

- Einladungen (Spec 05) und alles Fachliche wie Items, Räume und Umzug (Specs 02–04)
- Passwort-Zurücksetzen und Einladungen per Mail
- Offline-Nutzung über das Zwischenspeichern der App-Hülle hinaus
- Einrichtung von Pi, FritzBox, DuckDNS, SSD und Backups (siehe `.scratch/pi-betrieb/`), außerdem systemd-Dienst und Backup-Skript

## Further Notes

- Diese Spec ist Voraussetzung für alle weiteren. Die Pi-Tickets blockieren sie nicht.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [x] R001: Go-Modul und Projektgerüst anlegen. Ein Befehl startet den Server lokal, ein Befehl baut das Binary für linux/arm64 inklusive Client-Build.
- [x] R002: Konfiguration aus Umgebungsvariablen laden und prüfen: Datenverzeichnis, Adresse, optional DuckDNS (Domain und Token), optional SMTP. Unvollständige Gruppen führen zu einem klaren Startfehler.
- [x] R003: SQLite mit reinem Go-Treiber anbinden (WAL, Fremdschlüssel an). Eingebettete Migrationen beim Start ausführen. Erste Migration mit Person, Nutzer (eindeutige E-Mail ohne Beachtung der Groß- und Kleinschreibung) und Sitzung.
- [x] R004: Mailer-Schnittstelle mit SMTP-Implementierung und Log-Implementierung. Ist SMTP nicht konfiguriert, wird die Log-Implementierung gewählt.
- [x] R005: Unterbefehle für Server starten, Nutzer anlegen und Passwort zurücksetzen. Der Reset beendet alle Sitzungen des Nutzers.
- [x] R006: Anmeldung, Abmeldung und Abruf des angemeldeten Nutzers. Passwörter mit argon2id, Sitzungs-Cookie HttpOnly, `SameSite=Strict`, 90 Tage, `Secure` bei HTTPS, Token nur gehasht gespeichert. Middleware, die ohne Sitzung 401 liefert. JSON-Fehlerformat mit fachlichem Code.
- [x] R007: TLS: ohne DuckDNS-Angaben HTTP, mit Angaben automatisches Zertifikat per DNS-01 über DuckDNS und Betrieb per HTTPS.
- [x] R008: SvelteKit-Client mit statischem Adapter: Anmeldeseite, Startseite mit Nutzernamen und Abmelden, Weiterleitung bei 401. PWA-Manifest mit Icons und Service Worker für die App-Hülle.
- [x] R009: Client ins Binary einbetten und ausliefern. Nicht-API-Pfade liefern die App-Hülle.
- [x] R010: Test-Helfer (Server im temporären Verzeichnis, angemeldeter Nutzer, aufzeichnender Mailer) und API-Tests: erfolgreiche und fehlgeschlagene Anmeldung, Abmelden, 401 ohne Sitzung, Passwort-Reset macht alte Sitzungen ungültig, per Unterbefehl angelegter Nutzer kann sich anmelden.
- [x] R011: Ein Prüfbefehl führt Go-Tests, svelte-check und den Client-Build aus. Eine kurze Betriebsanleitung beschreibt die Umgebungsvariablen und den Start auf dem Pi.
