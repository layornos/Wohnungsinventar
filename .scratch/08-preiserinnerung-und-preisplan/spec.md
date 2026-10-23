# Spec 08: Preiserinnerung und Preisplan

Status: needs-info
Blocked by: 06-verkauf, 07-erinnerungen

## Problem Statement

Wenn ein Item zu teuer inseriert ist, bleibt es liegen, und ich vergesse, den Angebotspreis zu senken. Manchmal will ich nur nach einer Weile daran erinnert werden, den Preis zu überprüfen. Manchmal weiß ich schon beim Inserieren, wie ich den Preis senken will, z. B. nach 14 Tagen um 10 % und nach 30 Tagen noch einmal. Den Preis soll die App aber nie selbst ändern, weil ich ihn auch auf der Plattform von Hand ändern muss.

## Solution

Für ein Item mit Verbleib Verkaufen wählt man eine von zwei Arten: eine Preiserinnerung nach N Tagen oder einen Preisplan mit mehreren Stufen. Beide erzeugen über den Erinnerungs-Dienst aus Spec 07 Erinnerungen, per Mail und in der App, mit derselben Empfängerregel. Eine Erinnerung aus einem Preisplan nennt den vorgeschlagenen neuen Angebotspreis. Man übernimmt ihn mit einem Tippen, gibt einen anderen ein oder lässt den Preis, wie er ist. Der Angebotspreis wird nur durch den Nutzer geändert. Standardwerte für N und einen Standard-Preisplan legt man in den Haushaltseinstellungen fest.

## User Stories

1. Als Nutzer will ich für ein Item mit Verbleib Verkaufen eine Preiserinnerung einstellen, die mich nach N Tagen erinnert, den Angebotspreis zu überprüfen.
2. Als Nutzer will ich, dass die Preiserinnerung nach jeder Änderung des Angebotspreises neu zu laufen beginnt.
3. Als Nutzer will ich für ein Item einen Preisplan festlegen, z. B. „nach 14 Tagen um 10 % senken, nach 30 Tagen um weitere 10 %“.
4. Als Nutzer will ich eine Stufe auch als festen neuen Preis angeben können, z. B. „nach 21 Tagen auf 30 €“.
5. Als Nutzer will ich in der Erinnerung zu einer Preisplan-Stufe den aktuellen und den vorgeschlagenen Angebotspreis sehen.
6. Als Nutzer will ich den vorgeschlagenen Preis mit einem Tippen übernehmen, einen anderen eingeben oder den Preis lassen.
7. Als Nutzer will ich, dass der Angebotspreis nie automatisch geändert wird, damit er mit dem Inserat auf der Plattform übereinstimmt.
8. Als Nutzer will ich nach einer Preisänderung den aktualisierten Inserattext kopieren können, um das Inserat anzupassen.
9. Als Nutzer will ich auf der Detailseite sehen, welche Art eingestellt ist und wann die nächste Erinnerung oder Stufe fällig ist.
10. Als Nutzer will ich Preiserinnerung oder Preisplan eines Items ändern oder abschalten.
11. Als Nutzer will ich in den Haushaltseinstellungen einen Standardwert für N und einen Standard-Preisplan festlegen, damit ich sie nicht für jedes Item neu eingeben muss.
12. Als Nutzer will ich, dass Preiserinnerung und Preisplan enden, wenn das Item verkauft wird oder der Verbleib sich ändert.

## Implementation Decisions

- **Begriffe wie in `CONTEXT.md`.** Pro Item gibt es höchstens eine **Preis-Einstellung**: keine, Preiserinnerung oder Preisplan. Beide gleichzeitig gibt es nicht. *(Annahme)*
- **Nur bei Verbleib Verkaufen.** Bei Verschenken (Angebotspreis 0 €) und allen anderen Verbleiben wird mit „kein Verkauf“ abgelehnt. Wechselt der Verbleib weg von Verkaufen, wird die Preis-Einstellung samt Terminen gelöscht. Offene Erinnerungen werden wie in Spec 07 erledigt.
- **Ohne Angebotspreis** ist eine Preiserinnerung erlaubt, ein Preisplan mit Prozent-Stufen nicht („Angebotspreis fehlt“). *(Annahme)*
- **Datenmodell:**
  - **Preis-Einstellung** am Item: Art (Preiserinnerung oder Preisplan), Intervall in Tagen (Preiserinnerung), Startpreis und Startdatum (Preisplan).
  - **Preisplan-Stufe:** gehört zum Preisplan eines Items, mit Tagen nach dem Startdatum und entweder einer Senkung in Prozent oder einem festen Zielpreis in Cent. Die Stufen sind nach Tagen aufsteigend, die Tage sind eindeutig und > 0.
  - **Haushaltseinstellungen** (Tabelle aus Spec 06): Standard-Intervall (Annahme: 14 Tage) und Standard-Preisplan (Annahme: nach 14 Tagen −10 %, nach 30 Tagen −10 %, nach 60 Tagen −20 %).
  - Termin und Erinnerung aus Spec 07 bekommen die neuen Arten **Preiserinnerung** und **Preisplan-Stufe**. Die Erinnerung einer Stufe speichert Stufe, Preis beim Auslösen und vorgeschlagenen Preis.
- **Preiserinnerung:**
  - Termin = letzte Änderung des Angebotspreises, sonst „angeboten seit“ aus Spec 06, jeweils + N Tage. *(Annahme)*
  - Nach dem Auslösen wiederholt sie sich alle N Tage ab dem Auslösetag, bis das Item verkauft ist, der Verbleib wechselt oder sie abgeschaltet wird. *(Annahme)*
  - Jede Änderung des Angebotspreises setzt den Termin auf Änderungstag + N.
- **Preisplan:**
  - Beim Festlegen merkt sich der Plan den aktuellen Angebotspreis als Startpreis und heute als Startdatum. *(Annahme: nicht „angeboten seit“, damit ein nachträglich angelegter Plan nicht sofort mehrere Stufen auslöst)*
  - Jede Stufe ist fällig am Startdatum + ihre Tage, gerechnet ab Startdatum, nicht ab der vorigen Stufe. *(Annahme)*
  - Vorgeschlagener Preis: Prozent-Stufen beziehen sich auf den vorgeschlagenen Preis der vorigen Stufe, die erste auf den Startpreis. „−10 %, −10 %“ von 100 € ergibt also 90 € und 81 €. Gerundet wird auf ganze Euro abwärts, nie unter 0. *(Annahme)*
  - Manuelle Preisänderungen verschieben den Plan nicht. Die Erinnerung zeigt immer den aktuellen Preis neben dem Vorschlag. Liegt der aktuelle Preis schon auf oder unter dem Vorschlag, sagt die Erinnerung das. *(Annahme)*
  - Nach der letzten Stufe endet der Plan. Es kommt keine weitere Erinnerung. *(Annahme)*
  - Den Plan zu ändern legt Startpreis und Startdatum neu fest. Schon ausgelöste Stufen bleiben als Erinnerungen im Verlauf.
  - Nachgeholte Stufen nach einem Ausfall: Sind mehrere Stufen gleichzeitig fällig, wird nur die letzte ausgelöst, mit ihrem Vorschlag. *(Annahme)*
- **Kein automatisches Ändern:** Weder der Lauf noch irgendeine Reaktion ändert den Angebotspreis ohne ausdrückliche Aktion des Nutzers. „Vorschlag übernehmen“ ist eine solche Aktion und läuft über dieselbe Regel wie das Bearbeiten des Angebotspreises in Spec 06.
- **Reaktionen** (zusätzlich zu „Erledigt“ aus Spec 07):
  - Preiserinnerung: „Preis ändern“ (mit neuem Preis), „Preis lassen“.
  - Preisplan-Stufe: „Vorschlag übernehmen“, „Preis ändern“ (mit anderem Preis), „Preis lassen“.
  - Eine Preisänderung über eine Reaktion setzt die Preiserinnerung wie oben neu.
- **Empfänger und Versand** wie in Spec 07, inklusive Sammelmail. Die Mail nennt aktuellen Preis und Vorschlag.
- **API:**
  - Preis-Einstellung eines Items abrufen, setzen (Preiserinnerung mit N oder Preisplan mit Stufen; leer gelassen werden die Standardwerte vorgeschlagen, nicht gespeichert) und entfernen.
  - Haushaltseinstellungen um Standard-Intervall und Standard-Preisplan erweitern.
  - Item-Abruf enthält die Preis-Einstellung, die nächste Fälligkeit und bei einem Preisplan den Vorschlag jeder Stufe.
  - Reaktionen aus Spec 07 um die neuen Reaktionen erweitern.
  - Fehler mit fachlichen Codes: „kein Verkauf“, „Angebotspreis fehlt“, „Stufen ungültig“ (leere Liste, doppelte oder nicht positive Tage, Prozent nicht zwischen 1 und 99, negativer Zielpreis).
- **Client:**
  - Auf der Detailseite eines Items mit Verbleib Verkaufen ein Bereich „Preis“ mit Auswahl keine / Preiserinnerung / Preisplan, vorbelegt mit den Standardwerten.
  - Editor für Preisplan-Stufen mit Vorschau der vorgeschlagenen Preise und Fälligkeitstage.
  - Erinnerungen zu Preisen zeigen aktuellen Preis, Vorschlag und Reaktionsknöpfe. Nach einer Preisänderung wird der Bereich „Inserat“ aus Spec 06 angeboten.
  - Einstellungsseite um Standard-Intervall und Standard-Preisplan erweitern.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01, steuerbarer Uhr, aufzeichnendem Mailer und dem direkt auslösbaren Erinnerungslauf aus Spec 07.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item für 100 € mit Plan „nach 14 Tagen −10 %, nach 30 Tagen −10 %“: Am Tag 14 gibt es eine Erinnerung mit Vorschlag 90 €, und der Angebotspreis ist weiter 100 €. Am Tag 30 ist der Vorschlag 81 €.
  - Eine Preiserinnerung mit N = 7 meldet sich nach 7 Tagen. Wird am Tag 5 der Preis geändert, meldet sie sich erst am Tag 12.
  - Nach dem Markieren als verkauft kommt keine weitere Preis-Erinnerung.
  - Nach einem Ausfall von Tag 10 bis Tag 40 wird nur die Stufe vom Tag 30 ausgelöst.
- **Schwerpunkte:**
  - Angebotspreis ändert sich nie ohne Aktion des Nutzers
  - Berechnung der Vorschläge und Rundung
  - Fälligkeiten relativ zum Startdatum bzw. zur letzten Preisänderung
  - Wiederholung der Preiserinnerung
  - Ende bei Verkauf, Verbleibwechsel, Abschalten und nach der letzten Stufe
  - Prüfung der Stufen
  - Nachholen mehrerer Stufen
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 07.

## Out of Scope

- Automatisches Senken des Angebotspreises (ausdrücklich nicht gewünscht, siehe `CONTEXT.md`)
- Preisvorschläge aus Marktdaten oder Vergleichspreisen
- Mindestpreis als Untergrenze eines Preisplans
- Mehrere benannte Preisplan-Vorlagen
- Preiserinnerung oder Preisplan für Später verkaufen oder Verschenken

## Further Notes

- Die Spec ist bewusst von Spec 07 getrennt, damit die Erinnerungen für Behalten und Später verkaufen nicht auf die Preislogik warten müssen. Sie fügt dem Erinnerungs-Dienst nur neue Arten hinzu.
- Alle mit *(Annahme)* markierten Punkte hat Claude ohne Rückfrage festgelegt. Sie stehen unten unter „Offene Fragen“.

## Offene Fragen

1. **Beide Arten gleichzeitig?** Reicht pro Item entweder Preiserinnerung oder Preisplan, oder sollen beide nebeneinander laufen?
2. **Automatisch für neue Verkäufe?** Soll beim Setzen von Verkaufen automatisch der Standard (Preiserinnerung oder Preisplan) eingeschaltet werden, oder nur auf Wunsch (Annahme)?
3. **Standardwerte:** Passen 14 Tage und der Standard-Preisplan „14 Tage −10 %, 30 Tage −10 %, 60 Tage −20 %“?
4. **Bezugsdatum der Preiserinnerung:** Ab dem Angebotsbeginn bzw. der letzten Preisänderung (Annahme) oder ab dem Einschalten der Erinnerung?
5. **Wiederholung:** Soll sich die Preiserinnerung alle N Tage wiederholen (Annahme) oder nur einmal melden?
6. **Prozentbasis im Preisplan:** Bezieht sich „−10 %“ auf den Vorschlag der vorigen Stufe (Annahme), auf den Startpreis oder auf den tatsächlichen aktuellen Angebotspreis?
7. **Tage im Preisplan:** Gezählt ab Startdatum (Annahme) oder ab der vorigen Stufe?
8. **Rundung:** Auf ganze Euro abwärts (Annahme), auf „glatte“ Preise wie 5er-Schritte oder gar nicht?
9. **Manuelle Preisänderung während eines Plans:** Läuft der Plan unverändert weiter (Annahme), startet er neu, oder endet er?
10. **Nach der letzten Stufe:** Soll danach eine Preiserinnerung übernehmen oder ein Hinweis kommen, z. B. „Verschenken?“?
11. **Mindestpreis:** Soll ein Plan eine Untergrenze haben?
12. **Verschenken:** Soll es bei Verschenken eine Erinnerung geben, wenn nach N Tagen niemand das Item genommen hat?

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration für Preis-Einstellung am Item (Art, Intervall, Startpreis, Startdatum) und Preisplan-Stufen (Tage, Prozent oder Zielpreis). Termin- und Erinnerungsarten um Preiserinnerung und Preisplan-Stufe erweitern, Erinnerung um Stufe, Preis beim Auslösen und Vorschlag. Standardwerte in den Haushaltseinstellungen.
- [ ] R002: Preis-Einstellung abrufen, setzen und entfernen mit Prüfungen („kein Verkauf“, „Angebotspreis fehlt“, „Stufen ungültig“). Beim Wechsel weg von Verkaufen und beim Verkauf löschen.
- [ ] R003: Termine der Preiserinnerung: letzte Preisänderung bzw. „angeboten seit“ + N, Neusetzen bei jeder Preisänderung, Wiederholung alle N Tage nach dem Auslösen.
- [ ] R004: Termine des Preisplans: Startdatum + Tage je Stufe, Vorschläge aus Prozent oder Zielpreis mit Rundung, beim Nachholen nur die letzte fällige Stufe, Ende nach der letzten Stufe.
- [ ] R005: Erinnerungslauf aus Spec 07 um die neuen Arten erweitern, Mailtext mit aktuellem Preis und Vorschlag. Der Lauf ändert nie den Angebotspreis.
- [ ] R006: Reaktionen „Vorschlag übernehmen“, „Preis ändern“, „Preis lassen“. Preisänderungen laufen über die Regeln aus Spec 06 und setzen die Preiserinnerung neu.
- [ ] R007: Client: Bereich „Preis“ auf der Detailseite mit Auswahl der Art, Stufen-Editor mit Vorschau, nächste Fälligkeit. Einstellungsseite um die Standardwerte erweitern.
- [ ] R008: Client: Preis-Erinnerungen mit aktuellem Preis, Vorschlag und Reaktionsknöpfen, danach Angebot des aktualisierten Inserattexts.
- [ ] R009: API-Tests mit steuerbarer Uhr: Vorschläge und Rundung, Fälligkeiten, Neusetzen nach Preisänderung, Wiederholung, Nachholen mehrerer Stufen, Ende bei Verkauf und Verbleibwechsel, Angebotspreis bleibt ohne Aktion unverändert, Prüfung der Stufen.
