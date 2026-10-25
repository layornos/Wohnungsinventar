# Spec 06: Verkauf

Status: ready-for-agent
Blocked by: 02-items-erfassen, 03-wohnungen-raeume-lagerorte

## Problem Statement

Beim Entrümpeln markiere ich viele Items mit dem Verbleib Verkaufen oder Verschenken. Danach muss ich jedes Item auf einer Verkaufsplattform einstellen und dafür jedes Mal Titel, Beschreibung, Preis und den Hinweis tippen, dass es ein Privatverkauf ohne Gewährleistung ist. Wenn etwas weg ist, will ich festhalten, für wie viel es verkauft wurde, ohne dass das Item spurlos verschwindet. Und ich will sehen, wie viel das Entrümpeln in einem Zeitraum eingebracht hat.

## Solution

Ein Item mit Verbleib Verkaufen oder Später verkaufen bekommt einen Angebotspreis; Verschenken heißt im Inserat „zu verschenken“. Für jedes solche Item erzeugt die App einen plattformneutralen Inserattext aus einer Vorlage des Haushalts, der immer den Gewährleistungsausschluss enthält, und man kopiert ihn mit einem Tippen. Ist das Item weg, markiert man es als verkauft (mit Verkaufserlös) oder verschenkt, von jedem Verbleib aus. Es landet dann im Archiv: Es verschwindet aus der normalen Item-Liste, verliert seine Platzierung und bleibt mit Fotos, Besitzern und Verkaufserlös erhalten. Die Erlösübersicht summiert die Verkaufserlöse in einem einstellbaren Zeitraum.

## User Stories

1. Als Nutzer will ich einem Item mit Verbleib Verkaufen oder Später verkaufen einen Angebotspreis geben, damit klar ist, wofür ich es anbiete.
2. Als Nutzer will ich den Angebotspreis jederzeit ändern, z. B. wenn sich nichts tut.
3. Als Nutzer will ich, dass ein Item mit Verbleib Verschenken im Inserat „zu verschenken“ zeigt, damit Verschenken genauso abläuft wie Verkaufen.
4. Als Nutzer will ich den Angebotspreis in der Item-Liste und auf der Detailseite sehen.
5. Als Nutzer will ich die Item-Liste nach Verbleib Verkaufen filtern und sehen, welche Items noch keinen Angebotspreis haben.
6. Als Nutzer will ich für ein Item einen fertigen Inserattext mit Titel, Beschreibung, Preis und Gewährleistungsausschluss sehen.
7. Als Nutzer will ich Titel und Inserattext jeweils mit einem Tippen kopieren oder über das Teilen-Menü weitergeben, damit ich sie in die App der Verkaufsplattform einfügen kann.
8. Als Nutzer will ich die Fotos eines Items herunterladen oder teilen, damit ich sie beim Inserieren hochladen kann.
9. Als Nutzer will ich, dass der Inserattext immer den Gewährleistungsausschluss enthält, damit ich ihn nie vergesse.
10. Als Nutzer will ich die Vorlage für den Inserattext und den Wortlaut des Gewährleistungsausschlusses für den Haushalt einstellen.
11. Als Nutzer will ich ein Item von jedem Verbleib aus als verkauft markieren und dabei den Verkaufserlös und das Verkaufsdatum angeben.
12. Als Nutzer will ich ein Item als verschenkt markieren, mit Verkaufserlös 0 €.
13. Als Nutzer will ich, dass verkaufte und verschenkte Items ins Archiv wandern und aus der normalen Item-Liste verschwinden.
14. Als Nutzer will ich das Archiv durchsuchen und nach Besitzer, verkauft oder verschenkt und Zeitraum filtern.
15. Als Nutzer will ich auf der Detailseite eines archivierten Items Fotos, Besitzer, Verkaufserlös und Verkaufsdatum sehen.
16. Als Nutzer will ich einen falsch eingegebenen Verkaufserlös oder ein falsches Verkaufsdatum korrigieren.
17. Als Nutzer will ich einen Verkauf rückgängig machen, wenn der Käufer abgesprungen ist, damit das Item wieder im Bestand ist.
18. Als Nutzer will ich, dass ein verkauftes Möbelstück seine Lagerorte verliert und die Items darin ohne Lagerort im Raum stehen, damit sie neu einsortiert werden.
19. Als Nutzer will ich in der Erlösübersicht die Summe der Verkaufserlöse in einem Zeitraum sehen, z. B. dieser Monat, dieses Jahr, letzte 12 Monate, gesamt oder frei gewählt.
20. Als Nutzer will ich in der Erlösübersicht auch sehen, wie viele Items verkauft und wie viele verschenkt wurden, und die einzelnen Items dazu.
21. Als Nutzer will ich die Erlösübersicht nach Besitzer filtern, damit jeder sieht, was seine Items eingebracht haben.

## Implementation Decisions

Geklärt mit dem Nutzer am 2026-10-25 (Grilling); die früheren Annahmen sind damit entschieden. Leitlinie: schlicht und plattformneutral, „nicht zu fancy“.

- **Beträge** werden als ganze Cent gespeichert, nie als Gleitkommazahl. Es gibt nur Euro. Die Anzeige erfolgt deutsch formatiert, z. B. „45,50 €“, ganze Beträge ohne Nachkommastellen („45 €“).
- **Datenmodell** (Begriffe wie in `CONTEXT.md`), Erweiterung des Items:
  - **Angebotspreis:** optional, in Cent, ≥ 0.
  - **Angeboten seit:** Zeitpunkt, an dem das Item zuletzt den Verbleib Verkaufen oder Verschenken bekommen hat. Spec 08 rechnet Preiserinnerung und Verschenk-Erinnerung davon aus.
  - **Abgegeben am:** Datum des Verkaufs bzw. des Verschenkens. Ist es gesetzt, ist das Item im Archiv.
  - **Verkaufserlös:** in Cent, ≥ 0, gesetzt genau dann, wenn das Item im Archiv ist.
  - Der Verbleib eines archivierten Items ist Verkaufen oder Verschenken und unterscheidet so „verkauft“ von „verschenkt“.
- **Haushaltseinstellungen:** eine neue, kleine Schlüssel-Wert-Tabelle für Einstellungen des einen Haushalts. Diese Spec legt dort die Inserat-Vorlage und den Gewährleistungsausschluss ab. Spec 08 nutzt dieselbe Tabelle.
- **Regeln für den Angebotspreis:**
  - Optional, damit das Erfassen (Serienmodus) nicht aufhält. Die Item-Liste bekommt den Filter „ohne Angebotspreis“.
  - Setzen lässt er sich bei Verkaufen und Später verkaufen; bei anderem Verbleib wird mit „kein Verkauf“ abgelehnt.
  - Er wird nie automatisch gelöscht. Bei Verschenken zeigt die App „zu verschenken“, der gespeicherte Preis bleibt unangetastet. Bei Behalten und offen wird er nicht angezeigt, bleibt aber gespeichert und ist bei einem Wechsel zurück auf Verkaufen wieder da.
  - „Angeboten seit“ beginnt bei jedem Wechsel auf Verkaufen oder Verschenken neu.
- **Inserattext:**
  - Plattformneutral für Kleinanzeigen, Vinted, eBay und das Board im Firmen-Intranet: ohne Formatierung, ohne Links, keine Vorlagen je Plattform.
  - Er wird bei jedem Abruf aus der Vorlage erzeugt und nicht pro Item gespeichert. Kein eigener Text pro Item, keine Zusatzfelder (Zustand, Maße usw. gehören in die Beschreibung).
  - **Titel:** der Name des Items, höchstens 65 Zeichen (Grenze von Kleinanzeigen, der strengsten der genutzten Plattformen), sonst gekürzt mit „…“. Ist er gekürzt, steht der volle Name als erste Zeile im Text.
  - **Text:** aus der Vorlage mit den Platzhaltern `{name}`, `{beschreibung}` und `{preis}`. `{preis}` wird zum Betrag („15 €“, ohne VB oder Festpreis) bzw. bei Verschenken zu „zu verschenken“. Eine leere Beschreibung entfällt samt ihrer Leerzeile.
  - Ohne Angebotspreis (bei Verkaufen oder Später verkaufen) gibt es keinen Inserattext; der Client sperrt das Kopieren mit „Erst einen Preis eintragen“.
  - Standard-Vorlage: `{beschreibung}` + Leerzeile + `Preis: {preis}` + Leerzeile + „Abholung oder Versand nach Absprache.“
  - Der Gewährleistungsausschluss wird immer am Ende angehängt und ist nicht Teil der bearbeitbaren Vorlage. Ein leerer Gewährleistungsausschluss wird beim Speichern mit „Gewährleistungsausschluss fehlt“ abgelehnt.
  - Standard-Gewährleistungsausschluss (vom Nutzer gewählt, ohne Haftungsvorbehalt): „Privatverkauf. Der Verkauf erfolgt unter Ausschluss jeglicher Gewährleistung, Garantie und Rücknahme.“
- **Als verkauft oder verschenkt markieren** (in einer Transaktion):
  - Erlaubt bei jedem Verbleib. Der Server setzt den Verbleib dabei auf Verkaufen bzw. Verschenken. Verkauft: Verkaufserlös Pflicht, ≥ 0. Verschenkt: Verkaufserlös 0.
  - Das Datum ist standardmäßig heute in der Zeitzone des Haushalts und darf in der Vergangenheit liegen, nicht in der Zukunft („Datum in der Zukunft“).
  - Die Platzierung wird entfernt. Gehören dem Item Lagerorte (Möbelstück), entfallen sie wie in Spec 03 beim Löschen des Möbelstücks: Die Items darin stehen ohne Lagerort im Raum. Die Wegfall-Vorschau aus Spec 03 wird dafür vor dem Markieren genutzt.
  - Läuft ein Umzug, werden die Umzugsangaben des Items entfernt, es ist danach in keiner Box.
  - Der Verkaufserlös kann vom Angebotspreis abweichen. Der Angebotspreis bleibt zur Information erhalten.
- **Archiv:**
  - Archivierte Items erscheinen nicht in der normalen Item-Liste, in Raum- und Lagerortseiten, in „ohne Lagerort“, beim Packen und Auspacken, in Umzugsübersichten und nicht in der Platzauswahl als Möbelstück.
  - Findet die Suche auf der Startseite nichts, zeigt sie „Im Archiv suchen“ und nimmt den Suchbegriff mit.
  - Sie sind schreibgeschützt bis auf Verkaufserlös und Datum. Änderungen an Name, Verbleib, Besitzern, Fotos und Platzierung werden mit „Item archiviert“ abgelehnt.
  - Fotos bleiben erhalten (ADR-0002).
  - **Zurück in den Bestand:** Entfernt Datum und Verkaufserlös. Das Item ist wieder im Bestand, mit seinem Verbleib und seinem Angebotspreis, ohne Platzierung. Entfallene Lagerorte kommen nicht zurück.
  - Archivierte Items lassen sich weiterhin löschen, dann fehlen sie auch in der Erlösübersicht. Der Client fragt vorher nach und nennt dabei den Verkaufserlös, der danach fehlt.
- **Erlösübersicht:** Für einen Zeitraum (von–bis, Datumsgrenzen einschließlich, nach Datum der Abgabe in der Zeitzone des Haushalts) liefert sie die Summe der Verkaufserlöse, die Anzahl verkaufter und verschenkter Items und die Items selbst, das neueste zuerst. Optionaler Filter nach Besitzer. Bei mehreren Besitzern wird der Verkaufserlös gleichmäßig aufgeteilt; übrige Cents gehen der Reihe nach an die Besitzer in alphabetischer Reihenfolge (wie die Besitzer sortiert sind). Mit Filter zeigt jedes Item Gesamtbetrag und Anteil. Keine Aufteilung nach Monaten, keine Diagramme.
- **Zeitzone:** fest `Europe/Berlin`, ohne Umgebungsvariable. Die Zeitzonendaten werden ins Binary eingebettet (`time/tzdata`), damit es nicht auf die Pi-Installation ankommt. Spec 07 nutzt dieselbe Einstellung.
- **API:**
  - Item bearbeiten nimmt zusätzlich den Angebotspreis an. Item-Abruf und Liste liefern Angebotspreis, „angeboten seit“ und, im Archiv, Verkaufserlös und Datum.
  - Item-Liste: zusätzlicher Filter „ohne Angebotspreis“. Archivierte Items sind ausgeschlossen.
  - Inserattext eines Items abrufen (Titel und Text).
  - Item als verkauft oder verschenkt markieren, Verkaufserlös und Datum korrigieren, zurück in den Bestand.
  - Archiv auflisten: Textsuche, Filter nach Besitzer, verkauft oder verschenkt und Zeitraum, sortiert nach Abgabedatum absteigend, seitenweise.
  - Erlösübersicht für einen Zeitraum, optional je Besitzer.
  - Haushaltseinstellungen für Inserat-Vorlage und Gewährleistungsausschluss abrufen und speichern.
  - Fehler mit fachlichen Codes: „kein Verkauf“, „kein Angebotspreis“, „Item archiviert“, „Gewährleistungsausschluss fehlt“, „Datum in der Zukunft“.
- **Client:**
  - Angebotspreis auf der Detailseite und beim Bearbeiten, Euro-Eingabe mit Komma.
  - Bereich „Inserat“ auf der Detailseite bei Verkaufen, Später verkaufen und Verschenken: Titel, Text, Kopieren-Knöpfe, Teilen-Menü (Web Share API, wo verfügbar, sonst Kopieren) und Herunterladen der Fotos; ohne Angebotspreis gesperrt mit Hinweis.
  - Knöpfe „Verkauft“ und „Verschenkt“ immer unten auf der Detailseite (bei „Item löschen“), bei Verkaufs-Items zusätzlich im Bereich „Inserat“. Dialog für Verkaufserlös (vorbelegt mit dem Angebotspreis) und Datum, mit Wegfall-Warnung bei Möbelstücken.
  - Archiv als eigene Seite im Seitenmenü mit Suche und Filtern; Detailseite schreibgeschützt mit Korrektur und „Zurück in den Bestand“.
  - Erlösübersicht mit Zeitraum-Schnellwahl (dieser Monat, dieses Jahr, letzte 12 Monate, gesamt) und freiem Zeitraum, Besitzer-Filter.
  - Einstellungsseite für Inserat-Vorlage und Gewährleistungsausschluss mit Vorschau.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01. Die Uhr des Servers (`Jetzt`) wird für Datumsprüfungen gesteuert.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Inserattext eines Items mit Verbleib Verschenken enthält „zu verschenken“ und den Gewährleistungsausschluss; ohne Angebotspreis gibt es für ein Verkaufs-Item keinen Inserattext.
  - Ein Item mit 300 € Verkaufserlös und zwei Besitzern zählt im Besitzer-Filter je 150 €.
  - Nach dem Verkauf des Schreibtisch-Items ist es nicht mehr in der Item-Liste, aber im Archiv, und die Schere aus seiner oberen Schublade steht ohne Lagerort im Arbeitszimmer.
  - Die Erlösübersicht für März enthält einen Verkauf vom 31. März um 23:30 Uhr deutscher Zeit, aber keinen vom 1. April.
- **Schwerpunkte:**
  - Regeln für den Angebotspreis je Verbleib, Preis bleibt beim Wechsel erhalten
  - Gewährleistungsausschluss ist immer im Inserattext und lässt sich nicht leer speichern
  - Platzhalter, Preisformatierung, Titelkürzung auf 65 Zeichen
  - Markieren als verkauft oder verschenkt mit Wegfall der Lagerorte
  - Schreibschutz im Archiv, Korrektur und Rückgängig
  - archivierte Items fehlen in allen normalen Listen
  - Zeitraumgrenzen und Zeitzone der Erlösübersicht, Filter nach Besitzer mit Aufteilung
  - Beträge in Cent ohne Rundungsfehler
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 03.

## Out of Scope

- Direktes Veröffentlichen auf Verkaufsplattformen über deren Schnittstellen und Links zu Inseraten
- Preiserinnerung und Verschenk-Erinnerung (Spec 08) sowie alle Erinnerungen (Spec 07)
- Käufer, Versand, Bezahlung und Verhandlungsverlauf
- Steuerliche Auswertungen und Export der Erlösübersicht
- Mehrere Währungen
- Mehrere Vorlagen oder Vorlagen je Plattform, eigener Inserattext pro Item, Zusatzfelder wie Zustand oder Maße
- VB oder Festpreis als Auswahl
- Aufteilung der Erlösübersicht nach Monaten, Diagramme

## Further Notes

- Dass ein verkauftes Möbelstück seine Lagerorte verliert, steht so in `CONTEXT.md` unter „Ohne Lagerort“.
- „Angeboten seit“ wird hier eingeführt, damit Spec 08 einen Bezugszeitpunkt hat, ohne das Datenmodell erneut anzufassen.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration: Item um Angebotspreis (Cent), „angeboten seit“, Abgabedatum und Verkaufserlös (Cent) erweitern. Tabelle für Haushaltseinstellungen mit Standardwerten für Inserat-Vorlage und Gewährleistungsausschluss.
- [ ] R002: Zeitzone des Haushalts fest `Europe/Berlin`, Zeitzonendaten eingebettet.
- [ ] R003: Angebotspreis über Item bearbeiten: setzbar bei Verkaufen und Später verkaufen, nie automatisch gelöscht, „angeboten seit“ beim Wechsel auf Verkaufen oder Verschenken neu gesetzt. Angebotspreis in Item-Abruf und Liste, Filter „ohne Angebotspreis“.
- [ ] R004: Haushaltseinstellungen abrufen und speichern. Ein leerer Gewährleistungsausschluss wird abgelehnt.
- [ ] R005: Inserattext abrufen: Titel (höchstens 65 Zeichen, voller Name dann im Text) und Text aus der Vorlage mit Platzhaltern, deutsch formatierter Preis bzw. „zu verschenken“, leere Beschreibung entfällt, Gewährleistungsausschluss immer am Ende; ohne Angebotspreis „kein Angebotspreis“.
- [ ] R006: Als verkauft oder verschenkt markieren in einer Transaktion, von jedem Verbleib aus: Verbleib passend setzen, Verkaufserlös und Datum (nicht in der Zukunft) speichern, Platzierung entfernen, Lagerorte eines Möbelstücks wie in Spec 03 entfallen lassen, Umzugsangaben eines laufenden Umzugs entfernen.
- [ ] R007: Archiv: archivierte Items aus allen normalen Listen und Übersichten ausschließen, Schreibschutz mit „Item archiviert“, Korrektur von Verkaufserlös und Datum, zurück in den Bestand. Archiv-Liste mit Suche, Filtern und seitenweisem Abruf.
- [ ] R008: Erlösübersicht: Summe, Anzahl verkauft und verschenkt und Items in einem Zeitraum (Datumsgrenzen in der Zeitzone des Haushalts), optional je Besitzer mit gleichmäßiger Aufteilung.
- [ ] R009: Client: Angebotspreis anzeigen und bearbeiten, Bereich „Inserat“ mit Kopieren, Teilen und Foto-Download.
- [ ] R010: Client: Dialog „Verkauft“ bzw. „Verschenkt“ mit Wegfall-Warnung, Archiv-Seite im Seitenmenü, Hinweis „Im Archiv suchen“ auf der Startseite, schreibgeschützte Detailseite mit Korrektur und „Zurück in den Bestand“.
- [ ] R011: Client: Erlösübersicht mit Zeitraum-Schnellwahl und Besitzer-Filter, Einstellungsseite für Vorlage und Gewährleistungsausschluss mit Vorschau.
- [ ] R012: API-Tests für die Regeln zum Angebotspreis, Inserattext mit Gewährleistungsausschluss, Markieren mit Wegfall, Archiv-Ausschluss und Schreibschutz, zurück in den Bestand, Erlösübersicht mit Zeitzonengrenzen und Besitzer-Aufteilung.
