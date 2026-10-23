# Spec 04: Umzug und Boxen

Status: ready-for-agent
Blocked by: 03-wohnungen-raeume-lagerorte

## Problem Statement

Beim Packen für den Umzug will ich festhalten, welches Item in welcher Box steckt, aus welchem Raum es kommt und in welchen Raum der neuen Wohnung es soll. Dann muss ich nach dem Umzug nicht jede Box aufreißen, um etwas zu finden, und die Umzugshelfer wissen, wohin jede Box gehört. Nach dem Umzug will ich die Items nach und nach einsortieren.

## Solution

Man legt einen Umzug von der alten in die neue Wohnung an. Boxen bekommen fortlaufende Nummern, die man auf den Karton schreibt. Im Pack-Modus wählt man eine Box und packt Items hinein. Jedes Item bekommt einen Zielraum der neuen Wohnung, und der Herkunftsraum ist vorbelegt. Pro Box gibt es eine Packliste, pro Zielraum eine Übersicht, und Items ohne Zielraum werden angezeigt. Beim Abschluss stehen alle Items mit Zielraum im Zielraum, ohne Lagerort. Items ohne Zielraum bleiben nach einer Bestätigung, wo sie waren.

## User Stories

1. Als Nutzer will ich einen Umzug von einer alten in eine neue Wohnung anlegen.
2. Als Nutzer will ich, dass immer höchstens ein Umzug läuft, damit Herkunfts- und Zielräume eindeutig sind.
3. Als Nutzer will ich, dass alte und neue Wohnung verschieden sein müssen.
4. Als Nutzer will ich Boxen mit fortlaufender Nummer und optionaler Bezeichnung anlegen, damit ich die Nummer auf den Karton schreiben kann.
5. Als Nutzer will ich die Bezeichnung einer Box ändern und eine leere Box löschen.
6. Als Nutzer will ich ein Item in eine Box packen, es daraus wieder herausnehmen oder in eine andere Box umpacken.
7. Als Nutzer will ich für ein Item einen Zielraum der neuen Wohnung festlegen.
8. Als Nutzer will ich, dass der Herkunftsraum aus dem bisherigen Raum des Items vorbelegt ist, ich ihn aber ändern kann.
9. Als Nutzer will ich mehrere Items auf einmal in eine Box packen oder ihnen einen Zielraum geben, damit das Packen schnell geht.
10. Als Nutzer will ich einen Pack-Modus, in dem ich eine Box wähle und dann Items antippe oder neu fotografiere, die direkt in diese Box kommen.
11. Als Nutzer will ich im Pack-Modus nach Herkunftsraum filtern, damit ich Zimmer für Zimmer packen kann.
12. Als Nutzer will ich eine Packliste pro Box sehen, mit Foto, Name, Herkunftsraum und Zielraum jedes Items.
13. Als Nutzer will ich eine Box nach Nummer aufrufen, damit ich beim Auspacken sofort ihren Inhalt sehe.
14. Als Nutzer will ich bei der Suche nach einem Item sehen, in welcher Box es steckt.
15. Als Nutzer will ich die Item-Liste nach Box filtern.
16. Als Nutzer will ich eine Übersicht pro Zielraum sehen, welche Boxen und Items dorthin gehören, damit die Umzugshelfer wissen, wohin.
17. Als Nutzer will ich sehen, welche Items der alten Wohnung noch keinen Zielraum haben, damit ich nichts vergesse.
18. Als Nutzer will ich auf der Detailseite eines Items Herkunftsraum, Zielraum und Box sehen und ändern, solange der Umzug läuft.
19. Als Nutzer will ich den Umzug abschließen, damit jedes Item mit Zielraum im Zielraum steht, ohne Lagerort, und die Boxzuordnungen enden.
20. Als Nutzer will ich beim Abschließen sehen, welche Items keinen Zielraum haben, und bestätigen, dass sie an ihrem bisherigen Platz bleiben.
21. Als Nutzer will ich nach dem Abschluss Items nach und nach einsortieren, indem ich die Liste „ohne Lagerort“ abarbeite.
22. Als Nutzer will ich abgeschlossene Umzüge weiterhin sehen, aber nicht mehr ändern können.
23. Als Nutzer will ich, dass eine Wohnung, die zu einem Umzug gehört, nicht gelöscht werden kann.

## Implementation Decisions

- **Datenmodell** (Begriffe wie in `CONTEXT.md`):
  - **Umzug:** alte Wohnung, neue Wohnung (verschieden), Zustand (laufend oder abgeschlossen), angelegt am, abgeschlossen am. Höchstens ein laufender Umzug.
  - **Box:** gehört zu einem Umzug. Hat eine fortlaufende Nummer, die pro Umzug ab 1 eindeutig ist und nach dem Löschen nicht wiederverwendet wird, und eine optionale Bezeichnung.
  - **Umzugsangaben eines Items:** gehören zu einem Item und einem Umzug, je Item höchstens einmal pro Umzug. Sie bestehen aus Herkunftsraum (optional), Zielraum (optional, muss in der neuen Wohnung liegen) und Box (optional, muss zum selben Umzug gehören).
- **Umzugsangaben entstehen bei Bedarf**, nämlich wenn einem Item im laufenden Umzug zum ersten Mal ein Zielraum, eine Box oder ein Herkunftsraum gegeben wird. Der Herkunftsraum wird dann mit dem aktuellen Raum des Items vorbelegt, abgeleitet aus Lagerort oder direkter Raum-Platzierung. Hat das Item keinen Raum, bleibt der Herkunftsraum leer. Die Platzierung des Items bleibt während des Umzugs unverändert.
- **„Items der alten Wohnung ohne Zielraum“** sind Items, deren aktueller Raum zur alten Wohnung gehört und die keinen Zielraum haben.
- **Invarianten, die der Server erzwingt:**
  - höchstens ein laufender Umzug
  - Zielraum liegt in der neuen Wohnung
  - Box gehört zum laufenden Umzug
  - nur leere Boxen sind löschbar
  - abgeschlossene Umzüge sind unveränderlich
  - eine Wohnung, die zu einem Umzug gehört, ist nicht löschbar (erweitert die Löschregel aus Spec 03)
  - beim Löschen eines Items verschwinden seine Umzugsangaben
- **Abschluss in einer Transaktion:**
  - Items mit Zielraum werden direkt im Zielraum platziert, ohne Lagerort.
  - Items ohne Zielraum behalten ihre Platzierung.
  - Der Umzug wird abgeschlossen. Umzugsangaben und Boxen bleiben als Verlauf am abgeschlossenen Umzug erhalten, wirken aber nicht mehr: Items gelten danach als in keiner Box.
  - Gibt es Items der alten Wohnung ohne Zielraum, verlangt der Abschluss eine ausdrückliche Bestätigung und lehnt sonst mit „Items ohne Zielraum“ ab. Die Antwort enthält die Liste dieser Items.
- **API:**
  - Umzug: laufenden Umzug abrufen, anlegen, abschließen (mit Bestätigungs-Kennzeichen), abgeschlossene auflisten.
  - Boxen: auflisten, anlegen, umbenennen, löschen, nach Nummer abrufen samt Packliste.
  - Umzugsangaben: für ein Item setzen (Herkunftsraum, Zielraum, Box, jeweils auch zurücksetzbar) und als Sammeloperation für mehrere Items (Box oder Zielraum setzen).
  - Übersichten: pro Zielraum (Boxen und Items) und Items der alten Wohnung ohne Zielraum.
  - Item-Liste: zusätzlicher Filter Box und Herkunftsraum. Listenelemente und Item-Abruf enthalten während eines laufenden Umzugs die Umzugsangaben.
  - Fehler mit fachlichen Codes.
- **Client:**
  - Umzugsbereich: anlegen, Status, Übersichten.
  - Box-Liste und Box-Seite mit Packliste.
  - Box nach Nummer aufrufen über ein großes Eingabefeld.
  - Pack-Modus: aktive Box wählen, Filter nach Herkunftsraum, Items antippen zum Hineinpacken, Schnell-Erfassen aus Spec 02 legt neue Items direkt in die aktive Box, Mehrfachauswahl für Zielraum.
  - Übersicht pro Zielraum und Liste „ohne Zielraum“.
  - Abschluss-Dialog mit Liste der Items ohne Zielraum und Bestätigung.
  - Item-Detailseite zeigt Herkunftsraum, Zielraum und Box und lässt sie ändern.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Nach dem Abschluss des Umzugs ist Box 7 leer, und ihre Items stehen ohne Lagerort in ihren Zielräumen.
  - Ein Item, dessen Raum zur alten Wohnung gehört und das keinen Zielraum hat, verhindert den Abschluss ohne Bestätigung.
  - Der Herkunftsraum eines Items aus Schreibtisch › obere Schublade ist das Arbeitszimmer.
- **Schwerpunkte:**
  - höchstens ein laufender Umzug
  - Zielraum-Prüfung
  - fortlaufende Boxnummern ohne Wiederverwendung
  - Vorbelegung des Herkunftsraums
  - Sammeloperationen
  - Packliste
  - Übersichten
  - Abschluss mit und ohne Bestätigung
  - Unveränderlichkeit nach dem Abschluss
  - Löschregeln für Boxen und Wohnungen
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 03.

## Out of Scope

- Abbildung des Transports. Das Item ist einfach in der Box.
- QR-Codes oder druckbare Etiketten für Boxen
- Mehrere gleichzeitige Umzüge
- Automatisches Anlegen der Räume der neuen Wohnung. Das geschieht manuell über Spec 03.

## Further Notes

- Dass Items ohne Zielraum nach einer Bestätigung an ihrem bisherigen Platz bleiben, ist mit dem Nutzer abgestimmt.
- Herkunftsraum und Zielraum gehören zum Umzug, nicht dauerhaft zum Item.
- Der Umzug steht bevor, deshalb sind Pack-Modus und Packliste der wichtigste Teil dieser Spec. Wenn die Zeit drängt, können die Übersicht pro Zielraum und der Verlauf abgeschlossener Umzüge zuletzt kommen.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration für Umzug (alte und neue Wohnung, Zustand, Zeitstempel; höchstens ein laufender), Box (Umzug, fortlaufende Nummer pro Umzug ohne Wiederverwendung, Bezeichnung) und Umzugsangaben (Item, Umzug, Herkunftsraum, Zielraum, Box).
- [ ] R002: Umzug-API: laufenden abrufen, anlegen (verschiedene Wohnungen, höchstens einer laufend), abgeschlossene auflisten. Eine Wohnung mit Umzug ist nicht löschbar.
- [ ] R003: Box-API: auflisten, anlegen mit nächster Nummer, umbenennen, nur leer löschen, nach Nummer abrufen samt Packliste (Foto, Name, Herkunftsraum, Zielraum).
- [ ] R004: Umzugsangaben für ein Item setzen und zurücksetzen. Sie entstehen bei Bedarf mit vorbelegtem Herkunftsraum aus dem abgeleiteten Raum. Prüfungen: Zielraum in der neuen Wohnung, Box im laufenden Umzug. Beim Löschen eines Items verschwinden seine Umzugsangaben.
- [ ] R005: Sammeloperation: Box oder Zielraum für mehrere Items in einer Anfrage, ganz oder gar nicht.
- [ ] R006: Item-Liste um Filter nach Box und Herkunftsraum erweitern. Umzugsangaben in Liste und Item-Abruf, solange ein Umzug läuft.
- [ ] R007: Übersichten: pro Zielraum (Boxen und Items) und Items der alten Wohnung ohne Zielraum.
- [ ] R008: Abschluss in einer Transaktion: Items mit Zielraum im Zielraum ohne Lagerort platzieren, Items ohne Zielraum unverändert lassen, ohne Bestätigung mit „Items ohne Zielraum“ samt Liste ablehnen. Danach ist der Umzug unveränderlich.
- [ ] R009: Client: Umzugsbereich (anlegen, Status), Box-Liste, Box-Seite mit Packliste, Box nach Nummer aufrufen.
- [ ] R010: Client: Pack-Modus mit aktiver Box, Filter nach Herkunftsraum, Antippen zum Hineinpacken, Schnell-Erfassen direkt in die aktive Box, Mehrfachauswahl für Zielraum.
- [ ] R011: Client: Übersicht pro Zielraum, Liste „ohne Zielraum“, Abschluss-Dialog mit Bestätigung. Umzugsangaben auf der Item-Detailseite anzeigen und ändern.
- [ ] R012: API-Tests für alle Invarianten, Vorbelegung des Herkunftsraums, Boxnummern, Sammeloperation, Packliste, Übersichten, Abschluss mit und ohne Bestätigung und Unveränderlichkeit danach.
