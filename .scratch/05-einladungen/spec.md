# Spec 05: Einladungen

Status: ready-for-agent
Blocked by: 01-grundgeruest-und-anmeldung

## Problem Statement

Mehrere Personen im Haushalt sollen die App nutzen. Bisher können neue Nutzer nur per Kommandozeile auf dem Pi angelegt werden. Ein Nutzer soll andere selbst einladen können, ohne dass Mailversand eingerichtet ist und ohne offene Registrierung. Wer schon als Besitzer von Items geführt wird, soll beim Beitreten mit dieser Person verknüpft werden.

## Solution

Ein Nutzer erzeugt eine Einladung, optional für eine bestehende Person, und bekommt einen Link, den er selbst weiterschickt, z. B. per Messenger. Über den Link legt die eingeladene Person E-Mail und Passwort fest und ist damit Nutzer. Einladungen sind nur einmal verwendbar, laufen nach 7 Tagen ab und lassen sich zurückziehen.

## User Stories

1. Als Nutzer will ich eine weitere Person einladen und einen Einladungslink bekommen, den ich ihr selbst schicke.
2. Als Nutzer will ich den Link mit einem Tippen kopieren oder über das Teilen-Menü des Handys verschicken.
3. Als Nutzer will ich beim Einladen eine bestehende Person ohne Nutzerzugang auswählen, damit ihre Items ihr zugeordnet bleiben.
4. Als Nutzer will ich alternativ eine neue Person einladen, indem ich ihren Namen angebe.
5. Als eingeladene Person will ich über den Link sehen, wer mich eingeladen hat, und E-Mail und Passwort festlegen.
6. Als eingeladene Person will ich nach dem Annehmen direkt angemeldet sein.
7. Als Nutzer will ich offene Einladungen sehen, mit Name, Ersteller und Ablaufdatum.
8. Als Nutzer will ich eine offene Einladung zurückziehen, damit ein verlorener Link nicht mehr funktioniert.
9. Als Nutzer will ich, dass ein Einladungslink nach 7 Tagen abläuft.
10. Als Nutzer will ich, dass ein Einladungslink nur einmal verwendet werden kann.
11. Als eingeladene Person mit abgelaufenem, verwendetem oder zurückgezogenem Link will ich eine verständliche Meldung sehen, damit ich um einen neuen Link bitten kann.
12. Als Nutzer will ich, dass eine Person, die schon Nutzer ist, nicht erneut eingeladen werden kann.
13. Als Nutzer will ich, dass eine E-Mail-Adresse, die schon vergeben ist, beim Annehmen abgelehnt wird.
14. Als Nutzer will ich die Liste aller Nutzer des Haushalts sehen.

## Implementation Decisions

- **Datenmodell:** **Einladung** mit:
  - Token-Hash (der Token selbst wird nie gespeichert)
  - Person: entweder eine bestehende Person ohne Nutzerzugang, oder der Name einer anzulegenden Person
  - Ersteller (Nutzer)
  - angelegt am, läuft ab am (+7 Tage), angenommen am, zurückgezogen am
- **Zustand** wird abgeleitet: offen, angenommen, zurückgezogen oder abgelaufen. Nur offene Einladungen sind annehmbar.
- **Annehmen in einer Transaktion:**
  - Mit verknüpfter Person wird diese Person zum Nutzer. Sonst wird eine neue Person mit dem Namen aus der Einladung angelegt und zum Nutzer gemacht.
  - E-Mail eindeutig (Groß- und Kleinschreibung egal), Passwort mit argon2id wie in Spec 01.
  - Die Einladung wird als angenommen markiert, und es entsteht eine Sitzung wie bei der Anmeldung.
  - Hat eine verknüpfte Person inzwischen einen Nutzerzugang, wird das Annehmen abgelehnt.
- **Link:** Er wird aus der Adresse gebildet, unter der der Nutzer die App gerade aufruft, damit er im Heimnetz und über das VPN gleich ist. Der Token ist zufällig mit mindestens 128 Bit und URL-sicher kodiert.
- **Uhr als Abhängigkeit des Servers**, damit Tests den Ablauf prüfen können, ohne zu warten.
- **API:**
  - geschützt: Einladung anlegen (liefert den Link einmalig), offene Einladungen auflisten, zurückziehen, Nutzer auflisten
  - öffentlich: Einladung per Token prüfen (liefert Name und Ersteller, sonst den Zustand als fachlichen Fehler), Einladung annehmen (E-Mail, Passwort)
  - Fehler mit fachlichen Codes: „Einladung abgelaufen“, „Einladung verwendet“, „Einladung zurückgezogen“, „E-Mail vergeben“, „Person ist bereits Nutzer“
- **Client:**
  - Bereich „Nutzer“ mit Nutzerliste, offenen Einladungen und neuer Einladung (bestehende Person auswählen oder Namen eingeben).
  - Danach den Link anzeigen, mit Kopieren und Teilen-Menü (Web Share API, wo verfügbar).
  - Öffentliche Annehmen-Seite mit verständlichen Meldungen je Zustand.
- **Kein Mailversand** in dieser Spec. Der Mailer aus Spec 01 bleibt ungenutzt, damit Einladungen ohne SMTP funktionieren.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01. Die Uhr wird im Test-Server durch eine steuerbare ersetzt.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Nach dem Annehmen kann sich die neue Person anmelden.
  - Eine Einladung, deren Uhr um 7 Tage und 1 Minute vorgestellt wurde, wird mit „Einladung abgelaufen“ abgelehnt.
  - Eine verknüpfte Person bleibt nach dem Annehmen Besitzer ihrer Items.
- **Schwerpunkte:**
  - alle Zustände (offen, angenommen, zurückgezogen, abgelaufen)
  - einmalige Verwendung
  - Verknüpfung mit bestehender Person vs. neue Person
  - E-Mail-Eindeutigkeit
  - bereits Nutzer
  - öffentliche Endpunkte geben ohne gültigen Token nichts preis
  - geschützte Endpunkte ohne Sitzung 401
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die Anmelde-Tests aus Spec 01.

## Out of Scope

- Einladungen und Passwort-Zurücksetzen per Mail
- Rollen oder unterschiedliche Rechte. Alle Nutzer dürfen alles (ADR-0003).
- Nutzer entfernen oder deaktivieren

## Further Notes

- Diese Spec hängt nur von Spec 01 ab und kann jederzeit zwischen den anderen umgesetzt werden. Bis dahin legt man weitere Nutzer per Kommandozeile an.
- Die Verknüpfung mit einer bestehenden Person ist erst sinnvoll nutzbar, wenn Spec 02 Personen und Besitzer eingeführt hat. Personen existieren aber schon ab Spec 01, deshalb gibt es keine harte Abhängigkeit.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Uhr als Abhängigkeit des Servers einführen und in den Test-Helfern steuerbar machen.
- [ ] R002: Migration für Einladung (Token-Hash, verknüpfte Person oder Name, Ersteller, angelegt, läuft ab, angenommen, zurückgezogen).
- [ ] R003: Einladung anlegen: zufälliger, URL-sicherer Token mit mindestens 128 Bit, nur der Hash wird gespeichert, Ablauf +7 Tage, Link aus der aufgerufenen Adresse. Abgelehnt, wenn die Person schon Nutzer ist.
- [ ] R004: Offene Einladungen auflisten (Name, Ersteller, Ablauf), Einladung zurückziehen, Nutzer auflisten.
- [ ] R005: Öffentlich: Einladung per Token prüfen, mit fachlichem Fehler je Zustand.
- [ ] R006: Öffentlich: Einladung annehmen in einer Transaktion. Verknüpfte Person wird Nutzer oder neue Person wird angelegt, E-Mail eindeutig, Passwort mit argon2id, Einladung angenommen, Sitzung angelegt.
- [ ] R007: Client: Bereich „Nutzer“ mit Nutzerliste, offenen Einladungen, Zurückziehen und neuer Einladung (Person wählen oder Name). Link mit Kopieren und Teilen-Menü.
- [ ] R008: Client: öffentliche Annehmen-Seite mit Ersteller-Hinweis, Formular für E-Mail und Passwort, verständlichen Meldungen je Zustand und Weiterleitung in die App nach dem Annehmen.
- [ ] R009: API-Tests für alle Zustände inkl. Ablauf per gesteuerter Uhr, einmalige Verwendung, Verknüpfung mit bestehender Person (bleibt Besitzer), neue Person, E-Mail vergeben, bereits Nutzer, Zugriffsschutz.
