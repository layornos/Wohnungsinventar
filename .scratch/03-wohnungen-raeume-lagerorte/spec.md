# Spec 03: Wohnungen, Räume, Lagerorte

Status: erledigt
Blocked by: 02-items-erfassen

## Problem Statement

Ich weiß oft nicht, wo etwas hingehört. Ich will für jedes Item festhalten, an welchem festen Platz es liegt, z. B. in der oberen Schublade des Schreibtischs im Arbeitszimmer. Möbel wie der Schreibtisch sind dabei selbst Items, die ich vielleicht verkaufe. Wo ein Item gerade herumliegt, will ich nicht verfolgen.

## Solution

Wohnungen bestehen aus Räumen, und in Räumen gibt es verschachtelte Lagerorte. Ein Lagerort kann zu einem Möbelstück-Item gehören. Jedes Item hat entweder einen Lagerort, oder es steht direkt in einem Raum (z. B. ein Sofa), oder es ist noch nicht platziert. Fällt ein Lagerort weg, weil er gelöscht wird oder weil sein Möbelstück gelöscht wird, stehen die Items darin ohne Lagerort im Raum. Eine eigene Liste zeigt alle Items ohne Lagerort zum Einsortieren. Filter nach Raum oder Lagerort finden auch Items in verschachtelten Lagerorten.

## User Stories

1. Als Nutzer will ich Wohnungen anlegen, umbenennen und löschen, z. B. „Altbau Weststadt“ und „Neue Wohnung“.
2. Als Nutzer will ich in einer Wohnung Räume anlegen, umbenennen und löschen.
3. Als Nutzer will ich einen Raum nur löschen können, wenn darin keine Lagerorte und keine Items mehr sind, damit nichts unbemerkt seinen Platz verliert.
4. Als Nutzer will ich eine Wohnung nur löschen können, wenn sie keine Räume mehr hat.
5. Als Nutzer will ich in einem Raum Lagerorte anlegen, z. B. „Regal links“.
6. Als Nutzer will ich Lagerorte verschachteln, z. B. Schreibtisch › obere Schublade.
7. Als Nutzer will ich einen Lagerort einem Item zuordnen (dem Möbelstück), damit klar ist, dass die Schublade zum Schreibtisch gehört.
8. Als Nutzer will ich den vollständigen Pfad eines Lagerorts sehen, z. B. „Arbeitszimmer › Schreibtisch › obere Schublade“.
9. Als Nutzer will ich Lagerorte umbenennen.
10. Als Nutzer will ich einen Lagerort samt Inhalt in einen anderen Raum oder Lagerort verschieben, damit sich alle Items darin mitbewegen, wenn ich ein Regal umstelle.
11. Als Nutzer will ich daran gehindert werden, einen Lagerort in sich selbst oder in einen seiner Unterlagerorte zu verschieben.
12. Als Nutzer will ich beim Löschen eines Lagerorts vorher sehen, welche Items dadurch ohne Lagerort sind.
13. Als Nutzer will ich, dass die Items eines gelöschten Lagerorts und seiner Unterlagerorte danach ohne Lagerort in dem Raum stehen, in dem der Lagerort lag.
14. Als Nutzer will ich beim Löschen eines Möbelstück-Items gewarnt werden, dass seine Lagerorte entfallen und deren Items ohne Lagerort in dem Raum stehen, in dem der jeweilige Lagerort lag.
15. Als Nutzer will ich einem Item einen Lagerort zuweisen.
16. Als Nutzer will ich ein Item direkt in einen Raum stellen, ohne Lagerort, z. B. ein Sofa.
17. Als Nutzer will ich die Platzierung eines Items entfernen können.
18. Als Nutzer will ich beim Erfassen eines Items gleich einen Raum oder Lagerort wählen können, wobei der zuletzt gewählte vorausgewählt ist, damit ich ein Regal schnell durcharbeiten kann.
19. Als Nutzer will ich Items ohne Lagerort in einer eigenen Liste sehen, damit ich weiß, was noch einsortiert werden muss.
20. Als Nutzer will ich Items nach Raum und nach Lagerort filtern, inklusive der Items in verschachtelten Lagerorten.
21. Als Nutzer will ich auf der Detailseite eines Items sehen, wo es hingehört, als Pfad mit Wohnung und Raum.
22. Als Nutzer will ich auf der Seite eines Raums oder Lagerorts seine Unterlagerorte und die Items darin sehen.

## Implementation Decisions

- **Datenmodell** (Begriffe wie in `CONTEXT.md`):
  - **Wohnung:** Name. **Raum:** Name, gehört zu genau einer Wohnung.
  - **Lagerort:** Name. Sein Elternelement ist entweder ein Raum oder ein anderer Lagerort, genau eins von beiden. Optional gehört er zu einem Item (dem Möbelstück).
  - **Platzierung eines Items:** entweder ein Lagerort, oder ein Raum ohne Lagerort, oder nichts. Gespeichert wird höchstens eins von beidem.
  - Der Raum eines Items mit Lagerort ergibt sich aus der Kette der Lagerorte und wird nicht doppelt gespeichert. „Ohne Lagerort“ heißt, das Item hat keinen Lagerort; einen Raum kann es trotzdem haben.
- **Invarianten, die der Server erzwingt:**
  - Keine Zyklen bei Lagerorten. Verschieben in sich selbst oder einen Unterlagerort wird mit „Zyklus“ abgelehnt.
  - Ein Möbelstück-Item darf nicht in einem seiner eigenen Lagerorte platziert werden.
  - Ein Raum ist nur leer löschbar, also ohne Lagerorte und ohne direkt platzierte Items.
  - Eine Wohnung ist nur ohne Räume löschbar.
- **Wegfall eines Lagerorts:** Wird ein Lagerort gelöscht, oder ein Möbelstück-Item, dem Lagerorte gehören, dann entfallen diese Lagerorte samt allen Unterlagerorten in einer Transaktion. Alle Items darin stehen danach ohne Lagerort in dem Raum, in dem der weggefallene Lagerort lag. Eine Vorschau-Abfrage liefert vorher die betroffenen Items. Das Löschen von Items aus Spec 02 wird um diesen Wegfall erweitert.
- **Abfragen über verschachtelte Lagerorte** mit rekursiven Abfragen (rekursive CTEs gibt es in SQLite und PostgreSQL). Die Filter Raum und Lagerort umfassen alle Unterlagerorte. Pfade werden für die Anzeige berechnet.
- **API:**
  - Wohnungen: auflisten, anlegen, umbenennen, löschen.
  - Räume: auflisten, anlegen, umbenennen, löschen.
  - Lagerorte: auflisten als Baum je Raum, anlegen, umbenennen, verschieben, Möbelstück zuordnen und lösen, Vorschau für das Löschen, löschen.
  - Items: Platzierung setzen (Lagerort, Raum oder keine), auch schon beim Anlegen.
  - Item-Liste: zusätzliche Filter „Raum“, „Lagerort“ und „ohne Lagerort“.
  - Item-Abruf: Platzierung als Pfad (Wohnung › Raum › Lagerort › …), dazu die Lagerorte, die zu dem Item als Möbelstück gehören.
  - Fehler mit fachlichen Codes.
- **Client:**
  - Verwaltung von Wohnungen und Räumen.
  - Raum- und Lagerortseiten mit Baum und Items.
  - Platzierungsauswahl als durchsuchbare Pfadliste, beim Erfassen mit dem zuletzt gewählten Platz vorausgewählt.
  - Liste „ohne Lagerort“ mit schnellem Einsortieren.
  - Warn-Dialoge mit der Liste der betroffenen Items vor dem Löschen eines Lagerorts oder Möbelstück-Items.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Nach dem Löschen des Schreibtisch-Items steht die Schere, die in seiner oberen Schublade lag, ohne Lagerort im Arbeitszimmer.
  - Der Filter „Arbeitszimmer“ findet die Schere in Schreibtisch › obere Schublade.
  - Ein Regal in ein anderes Zimmer zu verschieben ändert den Raum aller Items darin.
- **Schwerpunkte:**
  - Zyklusschutz
  - Wegfall über mehrere Verschachtelungsebenen
  - Löschregeln für Raum und Wohnung
  - Filter über verschachtelte Lagerorte
  - Liste „ohne Lagerort“
  - Pfadberechnung
  - Möbelstück darf nicht in sich selbst platziert werden
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 02.

## Out of Scope

- Umzug, Herkunftsraum, Zielraum und Boxen (Spec 04)
- Abbildung der aktuellen Position eines Items. Erfasst wird nur der feste Lagerort.
- Grundrisse oder grafische Darstellung von Räumen

## Further Notes

- Ein Möbelstück darf nur nicht in einem *eigenen* Lagerort liegen. Ketten über mehrere Möbelstücke (A liegt in einem Lagerort von B, B in einem von A) werden nicht verhindert, weil Lagerorte an Räumen hängen und nicht am Platz ihres Möbelstücks.
- Die Platzierungsregel „höchstens Lagerort oder Raum“ erzwingt der Store; SQLite kann den CHECK nicht nachträglich zur Tabelle hinzufügen.
- Die Regel „Raum nur leer löschbar“ hat Claude ohne Rückfrage festgelegt, damit nichts unbemerkt seinen Platz verliert. Sie lässt sich leicht ändern.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [x] R001: Migration für Wohnung, Raum, Lagerort (Elternelement Raum oder Lagerort, genau eins; optionales Möbelstück-Item) und Platzierung am Item (Lagerort oder Raum, höchstens eins).
- [x] R002: API für Wohnungen und Räume mit Löschregeln: Raum nur leer, Wohnung nur ohne Räume.
- [x] R003: Lagerort-API: Baum je Raum, anlegen, umbenennen, Möbelstück zuordnen und lösen, verschieben mit Zyklusschutz.
- [x] R004: Pfadberechnung (Wohnung › Raum › Lagerort › …) per rekursiver Abfrage. Pfade erscheinen in Lagerort- und Item-Antworten.
- [x] R005: Wegfall: Vorschau der betroffenen Items, dann Löschen eines Lagerorts samt Unterlagerorten in einer Transaktion. Items stehen danach ohne Lagerort im Raum des Lagerorts. Das Löschen eines Möbelstück-Items aus Spec 02 löst denselben Wegfall aus.
- [x] R006: Platzierung von Items setzen und entfernen, auch beim Anlegen. Ein Möbelstück darf nicht in einem eigenen Lagerort platziert werden.
- [x] R007: Item-Liste um Filter nach Raum und Lagerort (inkl. verschachtelter Lagerorte) und „ohne Lagerort“ erweitern.
- [x] R008: Client: Verwaltung von Wohnungen und Räumen, Raum- und Lagerortseiten mit Baum, Items und Möbelstück-Zuordnung, Verschieben von Lagerorten.
- [x] R009: Client: Platzierungsauswahl als durchsuchbare Pfadliste in Erfassen und Bearbeiten, beim Erfassen mit dem zuletzt gewählten Platz vorausgewählt (lokal im Browser gemerkt). Pfad auf der Item-Detailseite.
- [x] R010: Client: Liste „ohne Lagerort“ mit schnellem Einsortieren, Warn-Dialoge mit betroffenen Items vor dem Löschen von Lagerorten und Möbelstück-Items.
- [x] R011: API-Tests für Zyklusschutz, mehrstufigen Wegfall (Lagerort und Möbelstück-Item), Löschregeln, verschachtelte Filter, „ohne Lagerort“, Pfade und Verschieben samt Inhalt.
