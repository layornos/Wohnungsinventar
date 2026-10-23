# Spec 06: Verkauf

Status: needs-info
Blocked by: 02-items-erfassen, 03-wohnungen-raeume-lagerorte

## Problem Statement

Beim Entrümpeln markiere ich viele Items mit dem Verbleib Verkaufen oder Verschenken. Danach muss ich jedes Item auf einer Verkaufsplattform einstellen und dafür jedes Mal Titel, Beschreibung, Preis und den Hinweis tippen, dass es ein Privatverkauf ohne Gewährleistung ist. Wenn etwas weg ist, will ich festhalten, für wie viel es verkauft wurde, ohne dass das Item spurlos verschwindet. Und ich will sehen, wie viel das Entrümpeln in einem Zeitraum eingebracht hat.

## Solution

Ein Item mit Verbleib Verkaufen bekommt einen Angebotspreis. Für Verschenken ist der Angebotspreis immer 0 €. Für jedes solche Item erzeugt die App einen Inserattext aus einer Vorlage des Haushalts, der immer den Gewährleistungsausschluss enthält, und man kopiert ihn mit einem Tippen. Ist das Item weg, markiert man es als verkauft (mit Verkaufserlös) oder verschenkt. Es landet dann im Archiv: Es verschwindet aus der normalen Item-Liste, verliert seine Platzierung und bleibt mit Fotos, Besitzern und Verkaufserlös erhalten. Die Erlösübersicht summiert die Verkaufserlöse in einem einstellbaren Zeitraum.

## User Stories

1. Als Nutzer will ich einem Item mit Verbleib Verkaufen einen Angebotspreis geben, damit klar ist, wofür ich es anbiete.
2. Als Nutzer will ich den Angebotspreis jederzeit ändern, z. B. wenn sich nichts tut.
3. Als Nutzer will ich, dass ein Item mit Verbleib Verschenken automatisch den Angebotspreis 0 € hat, damit Verschenken genauso abläuft wie Verkaufen.
4. Als Nutzer will ich den Angebotspreis in der Item-Liste und auf der Detailseite sehen.
5. Als Nutzer will ich die Item-Liste nach Verbleib Verkaufen filtern und sehen, welche Items noch keinen Angebotspreis haben.
6. Als Nutzer will ich für ein Item einen fertigen Inserattext mit Titel, Beschreibung, Preis und Gewährleistungsausschluss sehen.
7. Als Nutzer will ich Titel und Inserattext jeweils mit einem Tippen kopieren oder über das Teilen-Menü weitergeben, damit ich sie in die App der Verkaufsplattform einfügen kann.
8. Als Nutzer will ich die Fotos eines Items herunterladen oder teilen, damit ich sie beim Inserieren hochladen kann.
9. Als Nutzer will ich, dass der Inserattext immer den Gewährleistungsausschluss enthält, damit ich ihn nie vergesse.
10. Als Nutzer will ich die Vorlage für den Inserattext und den Wortlaut des Gewährleistungsausschlusses für den Haushalt einstellen.
11. Als Nutzer will ich ein Item als verkauft markieren und dabei den Verkaufserlös und das Verkaufsdatum angeben.
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

- **Beträge** werden als ganze Cent gespeichert, nie als Gleitkommazahl. Es gibt nur Euro. Die Anzeige erfolgt deutsch formatiert, z. B. „45,50 €“.
- **Datenmodell** (Begriffe wie in `CONTEXT.md`), Erweiterung des Items aus Spec 02:
  - **Angebotspreis:** optional, in Cent, ≥ 0.
  - **Angeboten seit:** Zeitpunkt, an dem das Item zuletzt den Verbleib Verkaufen oder Verschenken bekommen hat. Spec 08 rechnet Preiserinnerung und Preisplan davon aus.
  - **Abgegeben am:** Datum des Verkaufs bzw. des Verschenkens. Ist es gesetzt, ist das Item im Archiv.
  - **Verkaufserlös:** in Cent, ≥ 0, gesetzt genau dann, wenn das Item im Archiv ist.
  - Der Verbleib eines archivierten Items bleibt Verkaufen oder Verschenken und unterscheidet so „verkauft“ von „verschenkt“.
- **Haushaltseinstellungen:** eine neue, kleine Schlüssel-Wert-Tabelle für Einstellungen des einen Haushalts. Diese Spec legt dort die Inserat-Vorlage und den Gewährleistungsausschluss ab. Spec 08 nutzt dieselbe Tabelle.
- **Regeln für den Angebotspreis:**
  - Ein Angebotspreis lässt sich nur bei Verbleib Verkaufen setzen. Sonst wird mit „kein Verkauf“ abgelehnt.
  - Beim Wechsel auf Verschenken setzt der Server den Angebotspreis auf 0. Ein anderer Preis wird mit „Verschenken kostet nichts“ abgelehnt.
  - Der Angebotspreis ist bei Verkaufen optional, damit das Erfassen nicht aufhält. *(Annahme)*
  - Wechselt der Verbleib weg von Verkaufen oder Verschenken, wird der Angebotspreis entfernt. Bei einem erneuten Wechsel auf Verkaufen beginnt „Angeboten seit“ neu. *(Annahme)*
- **Inserattext:**
  - Er wird bei jedem Abruf aus der Vorlage erzeugt und nicht pro Item gespeichert. *(Annahme)*
  - Er besteht aus einem **Titel** (Name des Items) und einem **Text**. Die Vorlage hat die Platzhalter `{name}`, `{beschreibung}` und `{preis}`. `{preis}` wird zu „45 € VB“ bzw. bei Verschenken zu „zu verschenken“, ohne Angebotspreis zu „Preis auf Anfrage“. *(Annahme, auch das „VB“)*
  - Der Gewährleistungsausschluss wird immer am Ende angehängt und ist nicht Teil der bearbeitbaren Vorlage. So kann er nicht versehentlich herausgelöscht werden. Ein leerer Gewährleistungsausschluss wird beim Speichern mit „Gewährleistungsausschluss fehlt“ abgelehnt.
  - Standard-Vorlage *(Annahme)*: `{name}` + Leerzeile + `{beschreibung}` + Leerzeile + `Preis: {preis}` + Leerzeile + „Abholung oder Versand nach Absprache.“
  - Standard-Gewährleistungsausschluss *(Annahme, Wortlaut vom Nutzer zu bestätigen, keine Rechtsberatung)*: „Privatverkauf. Der Verkauf erfolgt unter Ausschluss jeglicher Gewährleistung, Garantie und Rücknahme. Die Haftung für Schäden aus der Verletzung von Leben, Körper oder Gesundheit sowie für Vorsatz und grobe Fahrlässigkeit bleibt unberührt.“
  - Der Text ist plattformneutral, ohne Formatierung und ohne Links, damit er sich in jede Plattform-App einfügen lässt. Plattformspezifische Vorlagen gibt es nicht. *(Annahme)*
- **Als verkauft oder verschenkt markieren** (in einer Transaktion):
  - Erlaubt nur bei Verbleib Verkaufen (Verkaufserlös Pflicht, ≥ 0) oder Verschenken (Verkaufserlös 0). Bei anderem Verbleib wird mit „kein Verkauf“ abgelehnt. *(Annahme: Später verkaufen muss erst auf Verkaufen gestellt werden)*
  - Das Datum ist standardmäßig heute in der Zeitzone des Haushalts und darf in der Vergangenheit liegen, nicht in der Zukunft. *(Annahme)*
  - Die Platzierung wird entfernt. Gehören dem Item Lagerorte (Möbelstück), entfallen sie wie in Spec 03 beim Löschen des Möbelstücks: Die Items darin stehen ohne Lagerort im Raum. Die Wegfall-Vorschau aus Spec 03 wird dafür vor dem Markieren genutzt.
  - Läuft ein Umzug (Spec 04), werden die Umzugsangaben des Items entfernt, es ist danach in keiner Box. *(Annahme; nur wenn Spec 04 umgesetzt ist)*
  - Der Verkaufserlös kann vom Angebotspreis abweichen. Der Angebotspreis bleibt zur Information erhalten.
- **Archiv:**
  - Archivierte Items erscheinen nicht in der normalen Item-Liste, in Raum- und Lagerortseiten, in „ohne Lagerort“, in Umzugsübersichten und nicht in der Platzierungsauswahl als Möbelstück.
  - Sie sind schreibgeschützt bis auf Verkaufserlös und Datum. Änderungen an Name, Verbleib, Besitzern, Fotos und Platzierung werden mit „Item archiviert“ abgelehnt. *(Annahme)*
  - Fotos bleiben erhalten (ADR-0002, Dateien im Datenverzeichnis).
  - **Verkauf rückgängig:** Entfernt Datum und Verkaufserlös. Das Item ist wieder im Bestand, mit seinem Verbleib, ohne Platzierung. Entfallene Lagerorte kommen nicht zurück. *(Annahme)*
  - Archivierte Items lassen sich weiterhin löschen, dann fehlen sie auch in der Erlösübersicht. Der Client fragt vorher nach. *(Annahme)*
- **Erlösübersicht:** Für einen Zeitraum (von–bis, Datumsgrenzen einschließlich, nach Datum der Abgabe in der Zeitzone des Haushalts) liefert sie die Summe der Verkaufserlöse, die Anzahl verkaufter und verschenkter Items und die Items selbst. Optionaler Filter nach Besitzer. Bei mehreren Besitzern zählt der volle Verkaufserlös für jeden gefilterten Besitzer, er wird nicht aufgeteilt. *(Annahme)*
- **Zeitzone:** `Europe/Berlin`, per Umgebungsvariable änderbar. Die Zeitzonendaten werden ins Binary eingebettet (`time/tzdata`), damit es nicht auf die Pi-Installation ankommt. *(Annahme; Spec 07 braucht dasselbe)*
- **API:**
  - Item bearbeiten aus Spec 02 nimmt zusätzlich den Angebotspreis an. Item-Abruf und Liste liefern Angebotspreis, „angeboten seit“ und, im Archiv, Verkaufserlös und Datum.
  - Item-Liste: zusätzlicher Filter „ohne Angebotspreis“. Archivierte Items sind standardmäßig ausgeschlossen.
  - Inserattext eines Items abrufen (Titel und Text).
  - Item als verkauft oder verschenkt markieren, Verkaufserlös und Datum korrigieren, Verkauf rückgängig machen.
  - Archiv auflisten: Textsuche, Filter nach Besitzer, verkauft oder verschenkt und Zeitraum, sortiert nach Abgabedatum absteigend, seitenweise.
  - Erlösübersicht für einen Zeitraum, optional je Besitzer.
  - Haushaltseinstellungen für Inserat-Vorlage und Gewährleistungsausschluss abrufen und speichern.
  - Fehler mit fachlichen Codes: „kein Verkauf“, „Verschenken kostet nichts“, „Item archiviert“, „Gewährleistungsausschluss fehlt“, „Datum in der Zukunft“.
- **Client:**
  - Angebotspreis auf der Detailseite und beim Bearbeiten, Euro-Eingabe mit Komma.
  - Bereich „Inserat“ auf der Detailseite mit Titel, Text, Kopieren-Knöpfen, Teilen-Menü (Web Share API, wo verfügbar, sonst Kopieren) und Herunterladen der Fotos.
  - Aktion „Verkauft“ bzw. „Verschenkt“ mit Dialog für Verkaufserlös (vorbelegt mit dem Angebotspreis) und Datum, mit Wegfall-Warnung bei Möbelstücken.
  - Archiv-Seite mit Suche und Filtern, Detailseite schreibgeschützt mit „Verkauf rückgängig“ und Korrektur.
  - Erlösübersicht mit Zeitraum-Schnellwahl (dieser Monat, dieses Jahr, letzte 12 Monate, gesamt) und freiem Zeitraum, Besitzer-Filter.
  - Einstellungsseite für Inserat-Vorlage und Gewährleistungsausschluss mit Vorschau.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01. Die Uhr des Servers (`Jetzt`) wird für Datumsprüfungen gesteuert.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item mit Verbleib Verschenken hat den Angebotspreis 0 €, und ein Inserattext dazu enthält „zu verschenken“ und den Gewährleistungsausschluss.
  - Nach dem Verkauf des Schreibtisch-Items ist es nicht mehr in der Item-Liste, aber im Archiv, und die Schere aus seiner oberen Schublade steht ohne Lagerort im Arbeitszimmer.
  - Die Erlösübersicht für März enthält einen Verkauf vom 31. März um 23:30 Uhr deutscher Zeit, aber keinen vom 1. April.
- **Schwerpunkte:**
  - Regeln für den Angebotspreis je Verbleib
  - Gewährleistungsausschluss ist immer im Inserattext und lässt sich nicht leer speichern
  - Platzhalter und Preisformatierung
  - Markieren als verkauft oder verschenkt mit Wegfall der Lagerorte
  - Schreibschutz im Archiv, Korrektur und Rückgängig
  - archivierte Items fehlen in allen normalen Listen
  - Zeitraumgrenzen und Zeitzone der Erlösübersicht, Filter nach Besitzer
  - Beträge in Cent ohne Rundungsfehler
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 03.

## Out of Scope

- Direktes Veröffentlichen auf Verkaufsplattformen über deren Schnittstellen und Links zu Inseraten
- Preiserinnerung und Preisplan (Spec 08) sowie alle Erinnerungen (Spec 07)
- Käufer, Versand, Bezahlung und Verhandlungsverlauf
- Steuerliche Auswertungen und Export der Erlösübersicht
- Mehrere Währungen
- Mehrere Vorlagen oder Vorlagen je Plattform

## Further Notes

- Dass ein verkauftes Möbelstück seine Lagerorte verliert, steht so in `CONTEXT.md` unter „Ohne Lagerort“.
- „Angeboten seit“ wird hier eingeführt, damit Spec 08 einen Bezugszeitpunkt hat, ohne das Datenmodell erneut anzufassen.
- Alle mit *(Annahme)* markierten Punkte hat Claude ohne Rückfrage festgelegt. Sie stehen unten unter „Offene Fragen“.

## Offene Fragen

1. **Wortlaut des Gewährleistungsausschlusses:** Passt der vorgeschlagene Text, oder gibt es einen eigenen? Soll die Haftungs-Einschränkung (Leben, Körper, Gesundheit, Vorsatz, grobe Fahrlässigkeit) drin sein?
2. **Plattformen:** Welche Plattformen werden genutzt (z. B. Kleinanzeigen, Facebook Marketplace, Vinted)? Reicht ein plattformneutraler Text, oder braucht eine Plattform eigene Längen oder Felder, z. B. eine Titel-Höchstlänge?
3. **Preisangabe:** Immer „VB“, immer Festpreis, oder pro Item wählbar? Was steht im Text, wenn kein Angebotspreis gesetzt ist?
4. **Inserattext pro Item:** Reicht der aus der Vorlage erzeugte Text, oder soll man ihn pro Item anpassen und speichern können, z. B. für Zustand oder Maße?
5. **Weitere Felder:** Sollen Zustand, Maße, Abholort oder Versand als eigene Felder in den Inserattext?
6. **Angebotspreis Pflicht?** Soll der Verbleib Verkaufen ohne Angebotspreis erlaubt sein?
7. **Wechsel des Verbleibs:** Soll der Angebotspreis beim Wechsel weg von Verkaufen verworfen oder aufbewahrt werden?
8. **Von welchem Verbleib aus verkaufen?** Darf man ein Item auch mit Verbleib Später verkaufen, Behalten oder offen direkt als verkauft markieren?
9. **Archiv in Listen:** Sollen archivierte Items in der Suche ganz fehlen oder z. B. über einen Schalter „Archiv einbeziehen“ auffindbar sein?
10. **Verkauf rückgängig:** Ist das gewünscht? Wenn ja, mit welchem Verbleib kommt das Item zurück?
11. **Platzierung beim Verkauf:** Ist es richtig, dass die Platzierung entfernt wird und Lagerorte eines Möbelstücks entfallen? Oder sollen die Lagerorte bleiben, bis man sie selbst löscht?
12. **Archivierte Items löschen:** Erlaubt, oder sollen sie für die Erlösübersicht unlöschbar sein?
13. **Erlösübersicht bei mehreren Besitzern:** Voller Betrag je Besitzer, gleichmäßig aufgeteilt, oder gar keine Auswertung je Besitzer?
14. **Erlösübersicht:** Reichen Summe und Anzahl, oder braucht es eine Aufteilung nach Monat?
15. **Zeitzone:** Ist `Europe/Berlin` richtig?

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration: Item um Angebotspreis (Cent), „angeboten seit“, Abgabedatum und Verkaufserlös (Cent) erweitern. Tabelle für Haushaltseinstellungen mit Standardwerten für Inserat-Vorlage und Gewährleistungsausschluss.
- [ ] R002: Zeitzone des Haushalts aus der Konfiguration (Standard `Europe/Berlin`), Zeitzonendaten eingebettet.
- [ ] R003: Angebotspreis über Item bearbeiten: nur bei Verkaufen setzbar, bei Verschenken immer 0, beim Wechsel weg entfernt, „angeboten seit“ beim Wechsel auf Verkaufen oder Verschenken gesetzt. Angebotspreis in Item-Abruf und Liste, Filter „ohne Angebotspreis“.
- [ ] R004: Haushaltseinstellungen abrufen und speichern. Ein leerer Gewährleistungsausschluss wird abgelehnt.
- [ ] R005: Inserattext abrufen: Titel und Text aus der Vorlage mit Platzhaltern, deutsch formatierter Preis bzw. „zu verschenken“, Gewährleistungsausschluss immer am Ende.
- [ ] R006: Als verkauft oder verschenkt markieren in einer Transaktion: Verbleib prüfen, Verkaufserlös und Datum (nicht in der Zukunft) speichern, Platzierung entfernen, Lagerorte eines Möbelstücks wie in Spec 03 entfallen lassen, Umzugsangaben eines laufenden Umzugs entfernen.
- [ ] R007: Archiv: archivierte Items aus allen normalen Listen und Übersichten ausschließen, Schreibschutz mit „Item archiviert“, Korrektur von Verkaufserlös und Datum, Verkauf rückgängig. Archiv-Liste mit Suche, Filtern und seitenweisem Abruf.
- [ ] R008: Erlösübersicht: Summe, Anzahl verkauft und verschenkt und Items in einem Zeitraum (Datumsgrenzen in der Zeitzone des Haushalts), optional je Besitzer.
- [ ] R009: Client: Angebotspreis anzeigen und bearbeiten, Bereich „Inserat“ mit Kopieren, Teilen und Foto-Download.
- [ ] R010: Client: Dialog „Verkauft“ bzw. „Verschenkt“ mit Wegfall-Warnung, Archiv-Seite und schreibgeschützte Detailseite mit Korrektur und Rückgängig.
- [ ] R011: Client: Erlösübersicht mit Zeitraum-Schnellwahl und Besitzer-Filter, Einstellungsseite für Vorlage und Gewährleistungsausschluss mit Vorschau.
- [ ] R012: API-Tests für die Regeln zum Angebotspreis, Inserattext mit Gewährleistungsausschluss, Markieren mit Wegfall, Archiv-Ausschluss und Schreibschutz, Rückgängig, Erlösübersicht mit Zeitzonengrenzen und Besitzer-Filter.
