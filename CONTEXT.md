# Wohnungsinventarverwaltung

Verwaltet die Gegenstände eines einzigen Haushalts: wo sie sich befinden, wem sie gehören, was mit ihnen passieren soll (behalten, verkaufen, verschenken) und wohin sie bei einem Umzug gebracht werden.

## Language

### Personen

**Haushalt**:
Die Gemeinschaft, deren Bestand verwaltet wird. Es gibt genau einen Haushalt; alle Nutzer sehen alle Items.
_Avoid_: Familie, WG, Mandant

**Person**:
Ein Mensch, der als Besitzer von Items geführt wird. Eine Person muss kein Nutzer sein.
_Avoid_: Kontakt

**Nutzer**:
Eine Person, die Zugang zur Anwendung hat und Benachrichtigungen erhält.
_Avoid_: User, Account, Konto

**Einladung**:
Ein Link, mit dem ein Nutzer eine Person in den Haushalt holt; nur so entsteht ein neuer Nutzer. Sie gilt sieben Tage, nur einmal und lässt sich zurückziehen.
_Avoid_: Registrierung, Invite

**Besitzer**:
Eine Person, der ein Item gehört. Jedes Item hat mindestens einen Besitzer, kann aber mehrere haben.
_Avoid_: Eigentümer, Owner

### Bestand

**Item**:
Ein einzelner physischer Gegenstand im Bestand, mit mindestens einem Foto.
_Avoid_: Gegenstand, Ding, Artikel, Sache

**Wohnung**:
Die Räumlichkeit, in der der Haushalt lebt oder bei einem Umzug leben wird. Sie besteht aus Räumen.
_Avoid_: Adresse, Zuhause

**Raum**:
Ein Zimmer einer Wohnung, z. B. Küche oder Arbeitszimmer. Ein Item kann direkt in einem Raum stehen, z. B. ein Sofa.
_Avoid_: Zimmer, Ort

**Lagerort**:
Der feste Platz, an den ein Item gehört – nicht, wo es gerade liegt. Etwas, in dem Items liegen, z. B. eine Schreibtisch-Schublade oder ein Regalfach. Lagerorte können verschachtelt sein und zu einem Item gehören (das Möbelstück, z. B. der Schreibtisch).
_Avoid_: Container, Behälter, Fach

**Platzierung**:
Wo ein Item hingehört: ein Lagerort, oder direkt ein Raum ohne Lagerort (z. B. ein Sofa), oder noch nichts.
_Avoid_: Position, Standort

**Wegfall**:
Ein Lagerort verschwindet samt seinen Unterlagerorten, weil er gelöscht wird oder weil sein Möbelstück gelöscht wird. Die Items darin stehen danach ohne Lagerort in dem Raum, in dem der Lagerort lag.

**Ohne Lagerort**:
Zustand eines Items, das noch keinen Lagerort hat oder dessen Lagerort weggefallen ist, z. B. weil das Möbelstück verkauft wurde oder ein Umzug abgeschlossen wurde. Das Item muss neu zugewiesen werden.
_Avoid_: heimatlos, verwaist

### Umzug

**Umzug**:
Ein zeitlich begrenzter Vorgang von einer alten in eine neue Wohnung, der angelegt und abgeschlossen wird. Während er läuft, hat ein Item einen Herkunftsraum, einen Zielraum und ggf. eine Box; beim Abschluss steht es im Zielraum, ohne Lagerort.
_Avoid_: Wohnungswechsel

**Herkunftsraum**:
Der Raum, aus dem ein Item bei einem Umzug kommt.
_Avoid_: Quellraum, alter Raum

**Zielraum**:
Der Raum der neuen Wohnung, in den ein Item bei einem Umzug gebracht werden soll.
_Avoid_: neuer Raum, Bestimmungsort

**Box**:
Eine Umzugskiste, in die Items für einen Umzug gepackt werden. Der Transport wird nicht abgebildet; das Item ist einfach in der Box.
_Avoid_: Kiste, Karton, Umzugskarton

### Verbleib

**Verbleib**:
Was mit einem Item geschehen soll: Behalten, Verkaufen, Später verkaufen oder Verschenken.
_Avoid_: Kategorie, Status, Entscheidung

**Verbleib offen**:
Ein Item, für das noch kein Verbleib festgelegt ist. Das ist erlaubt, damit das Erfassen beim Packen nicht aufhält.
_Avoid_: unentschieden, unsortiert

**Behalten**:
Verbleib eines Items, das bleibt. Alle drei Jahre gibt es eine Behalte-Prüfung.

**Behalte-Prüfung**:
Erinnerung alle drei Jahre, ob ein behaltenes Item weiter behalten oder verkauft werden soll. Die Frist beginnt nach jeder Erinnerung neu, unabhängig von einer Reaktion.
_Avoid_: Review, Überprüfung

**Später verkaufen**:
Verbleib eines Items, das erst in einem Jahr verkauft werden soll. Nach Ablauf gibt es eine Erinnerung.
_Avoid_: in einem Jahr verkaufen, zurückgestellt

**Verkaufen**:
Verbleib eines Items, das zu einem Angebotspreis angeboten wird.

**Verschenken**:
Verbleib eines Items, das kostenlos abgegeben wird. Es durchläuft denselben Ablauf wie ein Verkauf; im Inserattext steht „zu verschenken“ statt eines Preises.
_Avoid_: spenden, abgeben

### Verkauf

**Angebotspreis**:
Der Preis, zu dem ein Item aktuell angeboten wird.
_Avoid_: Verkaufspreis, Wunschpreis

**Verkaufserlös**:
Der Betrag, zu dem ein Item tatsächlich verkauft wurde.
_Avoid_: Verkaufspreis (für den tatsächlich erzielten Betrag)

**Inserattext**:
Ein Textbaustein für Verkaufsplattformen, der immer den Gewährleistungsausschluss enthält.
_Avoid_: Anzeigentext, Beschreibung

**Archiv**:
Die Sammlung verkaufter und verschenkter Items, die mit Verkaufserlös erhalten bleiben.
_Avoid_: Papierkorb, gelöscht

**Erlösübersicht**:
Die Summe der Verkaufserlöse in einem einstellbaren Zeitraum.
_Avoid_: Statistik, Report

**Gewährleistungsausschluss**:
Der Hinweis in einem Inserattext, dass es sich um einen Privatverkauf ohne Garantie, Gewährleistung und Rücknahme handelt.
_Avoid_: Haftungsausschluss

**Preiserinnerung**:
Die Erinnerung, den Angebotspreis eines Items zu überprüfen, das seit 30 Tagen zu diesem Preis angeboten wird. Sie gilt automatisch für jedes Item mit Verbleib Verkaufen und Angebotspreis; der Angebotspreis wird nie automatisch geändert.
_Avoid_: Preisalarm, Preisplan

**Verschenk-Erinnerung**:
Die Erinnerung, dass ein Item seit 30 Tagen zu verschenken ist und noch niemand es genommen hat.

### Benachrichtigung

**Erinnerung**:
Ein Hinweis, der in der App angezeigt und jedem Besitzer, der Nutzer ist, per Mail geschickt wird. Besitzer ohne Nutzerzugang werden nicht benachrichtigt; ist keiner der Besitzer Nutzer, geht die Erinnerung an alle Nutzer.
_Avoid_: Notification, Reminder, Alarm

## Code names

Code, comments, API paths, JSON fields, database tables and CLI commands are English (see `CLAUDE.md`). Every domain term has exactly one English name in code; use it consistently. The German terms above remain the language of the UI and of conversations with the household. _Avoid_ lists apply to the German domain language, not to these code names (e.g. **Erinnerung** is `Reminder` in code).

| Begriff | Code |
|---|---|
| Haushalt | `Household` |
| Person | `Person` |
| Nutzer | `User` |
| Einladung | `Invitation` |
| Besitzer | `Owner` |
| Item | `Item` |
| Foto / Hauptfoto | `Photo` / `MainPhoto` |
| Vorschlag (Name und Beschreibung aus dem Foto, ADR-0006) | `Suggestion` |
| Wohnung | `Apartment` |
| Raum | `Room` |
| Lagerort | `StorageLocation` |
| Möbelstück (Item, zu dem ein Lagerort gehört) | `Furniture` |
| Platz (Raum oder Lagerort zur Auswahl) | `Place` |
| Platzierung | `Placement` |
| Wegfall | `Removal` |
| Ohne Lagerort | `Unstored` |
| Umzug | `Move` |
| Herkunftsraum | `OriginRoom` |
| Zielraum | `TargetRoom` |
| Box | `Box` |
| Verbleib | `Disposition` |
| Verbleib offen | `Undecided` (Disposition `null`) |
| Behalten | `Keep` |
| Behalte-Prüfung | `KeepReview` |
| Später verkaufen | `SellLater` |
| Verkaufen | `Sell` |
| Verschenken | `GiveAway` |
| Angebotspreis | `AskingPrice` |
| Verkaufserlös | `SaleProceeds` |
| Inserattext | `ListingText` |
| Archiv | `Archive` |
| Erlösübersicht | `ProceedsSummary` |
| Gewährleistungsausschluss | `WarrantyDisclaimer` |
| Preiserinnerung | `PriceReminder` |
| Verschenk-Erinnerung | `GiveAwayReminder` |
| Erinnerung | `Reminder` |
| Design (Darstellung) | `Theme` |
