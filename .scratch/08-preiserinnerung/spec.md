# Spec 08: Preiserinnerung und Verschenk-Erinnerung

Status: done
Blocked by: 06-verkauf, 07-erinnerungen

## Problem Statement

Wenn ein Item zu teuer inseriert ist, bleibt es liegen, und ich vergesse, den Angebotspreis zu senken. Genauso bleibt Verschenk-Kram ewig in der Ecke stehen, wenn ihn niemand nimmt. Ich will nach einer Weile daran erinnert werden, ohne für jedes Item etwas einstellen zu müssen. Den Preis soll die App nie selbst ändern, weil ich ihn auch auf der Plattform von Hand ändern muss.

## Solution

Für jedes Item mit Verbleib Verkaufen und Angebotspreis gibt es automatisch eine Preiserinnerung: Wurde der Preis seit 30 Tagen nicht geändert, erinnert die App daran, ihn zu prüfen. Für jedes Item mit Verbleib Verschenken gibt es nach 30 Tagen eine Verschenk-Erinnerung. Beide laufen über den Erinnerungs-Dienst aus Spec 07: in der App mit Zähler und in der Monatsmail, mit derselben Empfängerregel. Man reagiert mit einem Tippen; den Preis ändert nur der Nutzer.

## User Stories

1. Als Nutzer will ich erinnert werden, den Angebotspreis eines Items zu prüfen, das seit 30 Tagen zum selben Preis angeboten wird.
2. Als Nutzer will ich dafür nichts pro Item einstellen müssen.
3. Als Nutzer will ich, dass die Preiserinnerung nach jeder Änderung des Angebotspreises neu zu laufen beginnt.
4. Als Nutzer will ich in der Preiserinnerung den aktuellen Preis sehen und direkt einen neuen eingeben oder den Preis lassen.
5. Als Nutzer will ich nach einer Preisänderung den aktualisierten Inserattext kopieren können, um das Inserat anzupassen.
6. Als Nutzer will ich, dass der Angebotspreis nie automatisch geändert wird, damit er mit dem Inserat auf der Plattform übereinstimmt.
7. Als Nutzer will ich erinnert werden, wenn ein Item seit 30 Tagen zu verschenken ist, und es dann weiter anbieten oder als verschenkt markieren.
8. Als Nutzer will ich auf der Detailseite sehen, wann die nächste Preis- bzw. Verschenk-Erinnerung fällig ist.
9. Als Nutzer will ich, dass diese Erinnerungen enden, wenn das Item verkauft oder verschenkt wird oder der Verbleib sich ändert.

## Implementation Decisions

Geklärt mit dem Nutzer am 2026-10-25 (Grilling). Ursprünglich enthielt die Spec auch einen Preisplan mit Stufen, Prozent-Senkungen und Rundung sowie einstellbare Fristen; das ist gestrichen. Grund: Die Mail kommt nach Spec 07 nur einmal im Monat, Stufen im 14-Tage-Takt würden per Mail zu spät gemeldet, und der Preisplan wäre „zu fancy“.

- **Begriffe wie in `CONTEXT.md`:** Preiserinnerung und Verschenk-Erinnerung sind zwei neue Arten von Termin und Erinnerung aus Spec 07. Es gibt keine Einstellung pro Item und keine Standardwerte in den Haushaltseinstellungen.
- **Frist:** fest 30 Tage, wie die festen Fristen in Spec 07.
- **Preiserinnerung** (Verbleib Verkaufen mit Angebotspreis):
  - Item um **Preis geändert am** erweitern: Zeitpunkt der letzten Änderung des Angebotspreises.
  - Termin = späterer Zeitpunkt von „Preis geändert am“ und „angeboten seit“ (Spec 06) + 30 Tage.
  - Ohne Angebotspreis gibt es keinen Termin. Wird ein Preis eingetragen, entsteht der Termin ab dann.
  - Jede Änderung des Angebotspreises (auch über Item bearbeiten) setzt den Termin auf Änderungstag + 30 Tage.
  - Reaktionen: **„Preis ändern“** mit neuem Preis (läuft über die Regeln aus Spec 06, setzt den Termin wie oben neu; der Client bietet danach den Bereich „Inserat“ zum Kopieren an) oder **„Preis lassen“** (nächster Termin = ursprünglicher Fälligkeitstag + 30 Tage, fester Takt wie bei der Behalte-Prüfung).
  - Bleibt sie unbeantwortet, kommt der nächste Termin trotzdem im festen Takt (Fälligkeitstag + 30 Tage); es entsteht aber höchstens eine offene Preiserinnerung je Item, eine noch offene wird nicht verdoppelt.
- **Verschenk-Erinnerung** (Verbleib Verschenken):
  - Termin = „angeboten seit“ + 30 Tage.
  - Reaktionen: **„Weiter anbieten“** (nächster Termin = Fälligkeitstag + 30 Tage) oder **„Verschenkt“** (der Client öffnet den Dialog aus Spec 06; mit dem Markieren ist die Erinnerung erledigt).
  - Unbeantwortet: wie bei der Preiserinnerung, höchstens eine offene je Item.
- **Ende:** Ein Wechsel des Verbleibs, das Markieren als verkauft oder verschenkt und das Löschen entfernen die Termine und erledigen offene Erinnerungen wie in Spec 07. „Zurück in den Bestand“ legt die Termine ab diesem Tag neu an.
- **Kein automatisches Ändern:** Weder der Lauf noch eine Reaktion ändert den Angebotspreis ohne ausdrückliche Eingabe des Nutzers.
- **Empfänger, Anzeige und Mail** wie in Spec 07: Erinnerung in der App mit Zähler, gilt für den ganzen Haushalt, Monatsmail am 1. mit allen offenen Erinnerungen. Die Mailzeile nennt bei der Preiserinnerung den aktuellen Preis, z. B. „Preis prüfen: Stehlampe (40 €, Sandro)“, bei der Verschenk-Erinnerung „Noch zu verschenken: Wasserkocher (Sandro)“.
- **API:**
  - Reaktionen aus Spec 07 um „Preis ändern“ (mit Preis), „Preis lassen“, „Weiter anbieten“ erweitern. „Verschenkt“ läuft über das Markieren aus Spec 06.
  - Item-Abruf enthält den nächsten Termin der Preis- bzw. Verschenk-Erinnerung.
  - Fehler mit fachlichen Codes aus Spec 06 und 07, z. B. „Reaktion passt nicht“.
- **Client:**
  - Preiserinnerung in der Erinnerungsliste mit aktuellem Preis, Eingabefeld für den neuen Preis und den Knöpfen „Preis ändern“ und „Preis lassen“; nach einer Änderung Angebot des aktualisierten Inserattexts.
  - Verschenk-Erinnerung mit „Weiter anbieten“ und „Verschenkt“.
  - Item-Detailseite mit „Preis prüfen ab …“ bzw. „Erinnerung zum Verschenken ab …“.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern, steuerbarer Uhr, aufzeichnendem Mailer und dem direkt auslösbaren Erinnerungslauf aus Spec 07.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item wird am 1. März für 40 € auf Verkaufen gesetzt. Am 31. März gibt es eine offene Preiserinnerung, und der Angebotspreis ist weiter 40 €.
  - Wird der Preis am 20. März auf 35 € geändert, gibt es am 31. März keine Preiserinnerung, erst am 19. April.
  - Nach „Preis lassen“ am 5. April ist die nächste Preiserinnerung am 30. April (Fälligkeitstag + 30).
  - Ein Item ohne Angebotspreis bekommt keine Preiserinnerung.
  - Ein Item, das am 1. März auf Verschenken gesetzt wird, hat am 31. März eine Verschenk-Erinnerung; nach dem Markieren als verschenkt kommt keine weitere.
  - Bleibt eine Preiserinnerung zwei Fristen lang unbeantwortet, gibt es trotzdem nur eine offene.
- **Schwerpunkte:**
  - Angebotspreis ändert sich nie ohne Eingabe des Nutzers
  - Fälligkeit relativ zur letzten Preisänderung bzw. „angeboten seit“, fester Takt nach „Preis lassen“ und „Weiter anbieten“
  - höchstens eine offene Erinnerung je Item und Art
  - Ende bei Verkauf, Verschenken, Verbleibwechsel und Löschen
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 07.

## Out of Scope

- Preisplan mit Stufen, Prozent-Senkungen oder Zielpreisen
- Einstellbare Fristen, Standardwerte in den Haushaltseinstellungen, Einstellung pro Item
- Automatisches Senken des Angebotspreises (ausdrücklich nicht gewünscht, siehe `CONTEXT.md`)
- Preisvorschläge aus Marktdaten oder Vergleichspreisen
- Preiserinnerung für Später verkaufen (dafür gibt es die Erinnerung aus Spec 07)

## Further Notes

- Die Spec ist bewusst von Spec 07 getrennt, damit die Erinnerungen für Behalten und Später verkaufen nicht warten müssen. Sie fügt dem Erinnerungs-Dienst nur zwei neue Arten hinzu.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [x] R001: Migration: Item um „Preis geändert am“ erweitern. Termin- und Erinnerungsarten um Preiserinnerung und Verschenk-Erinnerung erweitern.
- [x] R002: Termine der Preiserinnerung: späterer Zeitpunkt von Preisänderung und „angeboten seit“ + 30 Tage, nur mit Angebotspreis, Neusetzen bei jeder Preisänderung, fester Takt nach „Preis lassen“, höchstens eine offene je Item.
- [x] R003: Termine der Verschenk-Erinnerung: „angeboten seit“ + 30 Tage, fester Takt nach „Weiter anbieten“, höchstens eine offene je Item.
- [x] R004: Ende und Neuanlage bei Verbleibwechsel, Verkauf, Verschenken, „Zurück in den Bestand“ und Löschen wie in Spec 07.
- [x] R005: Reaktionen „Preis ändern“, „Preis lassen“, „Weiter anbieten“; Preisänderungen über die Regeln aus Spec 06. Mailzeilen mit aktuellem Preis. Der Lauf ändert nie den Angebotspreis.
- [x] R006: Client: Preis- und Verschenk-Erinnerungen mit Reaktionen und Preiseingabe, danach Angebot des aktualisierten Inserattexts; nächste Fälligkeit auf der Detailseite.
- [x] R007: API-Tests mit steuerbarer Uhr: Fälligkeiten, Neusetzen nach Preisänderung, fester Takt, keine Erinnerung ohne Preis, höchstens eine offene, Ende bei Verkauf und Verbleibwechsel, Angebotspreis bleibt ohne Eingabe unverändert.
