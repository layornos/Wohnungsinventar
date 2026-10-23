# Spec 02: Items erfassen

Status: erledigt
Blocked by: 01-grundgeruest-und-anmeldung

## Problem Statement

Ich weiß nicht mehr genau, was ich alles besitze und wem im Haushalt es gehört. Vor dem Umzug will ich jedes Item schnell mit dem Handy fotografieren, den Besitzern zuordnen und beim Durchgehen gleich entscheiden, ob ich es behalte, verkaufe oder verschenke. Die Entscheidung soll mich aber beim Erfassen nicht aufhalten.

## Solution

In der App fotografiert man ein Item, gibt ihm einen Namen und hat sich selbst als Besitzer vorausgewählt. Weitere Besitzer, auch Personen ohne Nutzerzugang, lassen sich hinzufügen. Der Verbleib kann gesetzt oder offen gelassen werden. Die Item-Liste lässt sich nach Name durchsuchen und nach Besitzer und Verbleib filtern, auch nach „Verbleib offen“.

## User Stories

1. Als Nutzer will ich Personen anlegen, die keine Nutzer sind, z. B. ein Kind, damit ich ihnen Items zuordnen kann.
2. Als Nutzer will ich Personen umbenennen, damit Tippfehler korrigierbar sind.
3. Als Nutzer will ich eine Person nur löschen können, wenn sie bei keinem Item der letzte Besitzer ist, damit kein Item besitzerlos wird.
4. Als Nutzer will ich, dass Personen mit Nutzerzugang nicht gelöscht werden können, damit niemand versehentlich ausgesperrt wird.
5. Als Nutzer will ich ein Item anlegen, indem ich es mit der Handykamera fotografiere, damit das Erfassen schnell geht.
6. Als Nutzer will ich alternativ ein vorhandenes Foto aus der Galerie hochladen.
7. Als Nutzer will ich beim Anlegen eines Items mich selbst als Besitzer vorausgewählt haben, damit das Anlegen schnell geht.
8. Als Nutzer will ich einem Item einen Namen und eine optionale Beschreibung geben.
9. Als Nutzer will ich einem Item einen oder mehrere Besitzer zuordnen, damit klar ist, wem es gehört.
10. Als Nutzer will ich daran gehindert werden, den letzten Besitzer eines Items zu entfernen, damit jedes Item mindestens einer Person gehört.
11. Als Nutzer will ich einem Item weitere Fotos hinzufügen, damit Details und Zustand sichtbar sind.
12. Als Nutzer will ich daran gehindert werden, das letzte Foto eines Items zu löschen, damit jedes Item mindestens ein Foto hat.
13. Als Nutzer will ich eines der Fotos als Hauptfoto festlegen, damit es in Listen angezeigt wird.
14. Als Nutzer will ich, dass Fotos richtig herum angezeigt werden, auch wenn das Handy sie gedreht gespeichert hat.
15. Als Nutzer will ich, dass Listen schnell laden, auch über das VPN, indem sie verkleinerte Vorschauen zeigen.
16. Als Nutzer will ich einem Item einen Verbleib geben (Behalten, Verkaufen, Später verkaufen, Verschenken) oder ihn offen lassen.
17. Als Nutzer will ich den Verbleib eines Items jederzeit ändern oder wieder auf offen setzen.
18. Als Nutzer will ich Items nach Name suchen.
19. Als Nutzer will ich Items nach Besitzer und nach Verbleib filtern, auch nach „Verbleib offen“, damit ich gezielt entrümpeln kann.
20. Als Nutzer will ich auf der Detailseite eines Items alle Fotos, die Besitzer und den Verbleib sehen.
21. Als Nutzer will ich Name, Beschreibung, Besitzer und Verbleib eines Items bearbeiten.
22. Als Nutzer will ich ein Item löschen, z. B. wenn ich es versehentlich angelegt habe, samt seinen Fotos.
23. Als Nutzer ohne Sitzung will ich keine Fotos abrufen können, auch nicht über ihre direkte Adresse.

## Implementation Decisions

- **Datenmodell:**
  - **Item:** Name (Pflicht), Beschreibung, Verbleib (optional: Behalten, Verkaufen, Später verkaufen, Verschenken; leer heißt offen), angelegt am, geändert am.
  - **Foto:** gehört zu einem Item, hat ein Hauptfoto-Kennzeichen (genau eins pro Item) und eine Reihenfolge.
  - **Besitz:** Zuordnung zwischen Item und Person.
  - Personen und Nutzer stammen aus Spec 01.
- **Ein Item entsteht in einer einzigen Anfrage** (multipart) mit mindestens einem Foto und mindestens einem Besitzer. So gibt es nie ein Item, das die Invarianten verletzt. Das erste Foto wird Hauptfoto.
- **Invarianten, die der Server erzwingt:**
  - Mindestens ein Foto. Das Löschen des letzten Fotos wird mit dem Code „letztes Foto“ abgelehnt.
  - Mindestens ein Besitzer. Das Entfernen des letzten Besitzers wird mit „letzter Besitzer“ abgelehnt.
  - Genau ein Hauptfoto. Wird das Hauptfoto gelöscht, wird das nächste in der Reihenfolge Hauptfoto; war es das letzte, das erste.
  - Eine Person mit Nutzerzugang oder die letzter Besitzer eines Items ist, kann nicht gelöscht werden.
- **Fotos:**
  - Sie liegen als Dateien im Foto-Ordner des Datenverzeichnisses, nicht in der Datenbank (ADR-0002). Dateinamen sind zufällige IDs, nie vom Nutzer vorgegeben.
  - Angenommen werden JPEG, PNG und WebP bis zu einer konfigurierbaren Maximalgröße (Standard 20 MB). Der Inhaltstyp wird anhand der Datei geprüft, nicht anhand des Namens.
  - Beim Hochladen wird eine verkleinerte Vorschau (z. B. 512 px lange Kante, JPEG) erzeugt. Die EXIF-Ausrichtung wird dabei angewendet, das Original bleibt unverändert erhalten.
  - Fotos werden nur mit gültiger Sitzung ausgeliefert.
  - Beim Löschen eines Items oder Fotos werden auch die Dateien entfernt.
- **API:**
  - Personen: auflisten, anlegen, umbenennen, löschen.
  - Items: anlegen (multipart), abrufen, bearbeiten (Name, Beschreibung, Verbleib), löschen.
  - Besitzer: hinzufügen und entfernen.
  - Fotos: hinzufügen (multipart), als Hauptfoto setzen, löschen, Original und Vorschau abrufen.
  - Item-Liste: filterbar nach Text (Name enthält, Groß- und Kleinschreibung egal), Besitzer und Verbleib (inkl. „offen“), sortiert nach zuletzt geändert, mit Seitenweise-Abruf.
  - Fehler mit fachlichen Codes nach dem Format aus Spec 01.
- **Client:**
  - Schnell-Erfassen: Kamera-Eingabe des Browsers (`capture`), alternativ Galerie, Name, Besitzer mit dem angemeldeten Nutzer vorausgewählt, Verbleib optional.
  - Item-Liste mit Vorschaubild, Suche und Filtern.
  - Detailseite mit Fotos, Besitzern und Verbleib, Bearbeiten und Löschen.
  - Personenverwaltung.
  - Große Tippflächen, mobil zuerst.

## Testing Decisions

- **Test-Seam ist die HTTP-API**, mit den Test-Helfern aus Spec 01: echter Server, echte SQLite-Datei und echter Foto-Ordner im temporären Verzeichnis.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item ohne Foto lässt sich nicht anlegen.
  - Nach dem Löschen des Hauptfotos ist ein anderes Foto Hauptfoto.
  - Der Filter „Verbleib offen“ liefert genau die Items ohne Verbleib.
- **Schwerpunkte:**
  - alle Invarianten
  - Löschregeln für Personen
  - Filter und Suche
  - Ablehnung falscher Dateitypen
  - Vorschau entsteht und ist gemäß EXIF gedreht (mit einem kleinen Test-JPEG mit Ausrichtungs-Tag)
  - Dateien verschwinden nach dem Löschen
  - ohne Sitzung kein Zugriff auf Fotos
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 01.

## Out of Scope

- Wohnungen, Räume, Lagerorte und Platzierung von Items (Spec 03). Bis dahin haben Items keine Platzierung.
- Umzug und Boxen (Spec 04)
- Verkaufsablauf, Erinnerungen und Behalte-Prüfung, also alles, was am Verbleib hängt, außer dem Feld selbst
- Sachkategorien

## Further Notes

- Der Verbleib darf bewusst offen bleiben, damit das Erfassen beim Packen nicht blockiert. Das ist mit dem Nutzer abgestimmt.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [x] R001: Migration für Item (Name, Beschreibung, optionaler Verbleib, Zeitstempel), Foto (Item, Hauptfoto-Kennzeichen, Reihenfolge, Dateiverweise) und Besitz (Item–Person).
- [x] R002: Foto-Speicher im Datenverzeichnis: Prüfung von Inhaltstyp und Größe, zufällige Dateinamen, Original speichern, Vorschau mit angewendeter EXIF-Ausrichtung erzeugen, Dateien löschen.
- [x] R003: Personen-API: auflisten, anlegen, umbenennen, löschen. Löschen wird abgelehnt bei Nutzerzugang oder wenn die Person bei einem Item letzter Besitzer ist.
- [x] R004: Item anlegen in einer multipart-Anfrage mit mindestens einem Foto und mindestens einem Besitzer. Das erste Foto ist Hauptfoto.
- [x] R005: Item abrufen, bearbeiten (Name, Beschreibung, Verbleib inkl. Zurücksetzen auf offen) und löschen samt Foto-Dateien.
- [x] R006: Besitzer hinzufügen und entfernen. Das Entfernen des letzten Besitzers wird mit „letzter Besitzer“ abgelehnt.
- [x] R007: Fotos hinzufügen, Hauptfoto setzen und löschen. Das letzte Foto ist nicht löschbar („letztes Foto“), nach dem Löschen des Hauptfotos rückt das nächste nach. Original und Vorschau nur mit Sitzung abrufbar.
- [x] R008: Item-Liste mit Textsuche, Filter nach Besitzer und Verbleib (inkl. „offen“), Sortierung nach zuletzt geändert und seitenweisem Abruf. Jedes Listenelement enthält die Adresse der Vorschau des Hauptfotos.
- [x] R009: Client, Schnell-Erfassen: Kamera oder Galerie, Name, angemeldeter Nutzer als Besitzer vorausgewählt, Verbleib optional.
- [x] R010: Client: Item-Liste mit Vorschaubildern, Suche und Filtern, Detailseite mit Fotos, Besitzern und Verbleib, Bearbeiten und Löschen inkl. Foto- und Besitzerverwaltung, fachliche Fehlermeldungen.
- [x] R011: Client: Personenverwaltung (auflisten, anlegen, umbenennen, löschen mit verständlicher Ablehnung).
- [x] R012: API-Tests für alle Invarianten, Löschregeln, Filter, Dateityp-Prüfung, EXIF-Vorschau, Entfernen der Dateien und Zugriffsschutz für Fotos.
