# Spec 07: Erinnerungen, Behalte-Prüfung und Später verkaufen

Status: needs-info
Blocked by: 06-verkauf

## Problem Statement

Wenn ich ein Item behalte oder erst später verkaufen will, vergesse ich es. Behaltene Dinge sammeln sich an, obwohl ich sie längst nicht mehr brauche, und was ich „in einem Jahr“ verkaufen wollte, liegt nach drei Jahren noch im Keller. Ich will, dass die App mich zur richtigen Zeit daran erinnert, per Mail und in der App, und zwar die Person, der das Item gehört, nicht den ganzen Haushalt.

## Solution

Der Server enthält einen Erinnerungs-Dienst, der regelmäßig prüft, welche Termine fällig sind. Für ein Item mit Verbleib Behalten ist alle drei Jahre eine Behalte-Prüfung fällig, für ein Item mit Verbleib Später verkaufen nach einem Jahr eine Erinnerung. Jede fällige Erinnerung erscheint in der App in einer Erinnerungsliste und geht per Mail an jeden Besitzer, der Nutzer ist. Ist keiner der Besitzer Nutzer, geht sie an alle Nutzer. In der App reagiert man mit einem Tippen, z. B. „Weiter behalten“ oder „Jetzt verkaufen“. Spec 08 nutzt denselben Dienst für Preiserinnerung und Preisplan.

## User Stories

1. Als Nutzer will ich, dass ich für ein behaltenes Item alle drei Jahre eine Behalte-Prüfung bekomme, damit ich regelmäßig entscheide, ob ich es noch brauche.
2. Als Nutzer will ich, dass die nächste Behalte-Prüfung drei Jahre nach der letzten fällig ist, egal ob und wie ich reagiert habe.
3. Als Nutzer will ich für ein Item mit Verbleib Später verkaufen nach einem Jahr eine Erinnerung bekommen, dass es jetzt verkauft werden soll.
4. Als Nutzer will ich Erinnerungen per Mail bekommen, damit ich sie auch sehe, wenn ich die App nicht öffne.
5. Als Nutzer will ich Erinnerungen auch in der App sehen, mit der Zahl der offenen Erinnerungen auf der Startseite.
6. Als Besitzer, der Nutzer ist, will ich die Erinnerungen zu meinen Items bekommen, und andere Nutzer nicht, damit niemand mit fremden Items behelligt wird.
7. Als Nutzer will ich Erinnerungen zu Items bekommen, deren Besitzer alle keinen Nutzerzugang haben, z. B. Items eines Kindes, damit sich jemand darum kümmert.
8. Als Nutzer will ich in der App alle Erinnerungen des Haushalts sehen, nicht nur meine, weil alle alle Items sehen.
9. Als Nutzer will ich auf eine Behalte-Prüfung mit „Weiter behalten“ oder „Verkaufen“ reagieren, wobei „Verkaufen“ den Verbleib auf Verkaufen setzt und mich zum Angebotspreis führt.
10. Als Nutzer will ich auf eine Erinnerung zu Später verkaufen mit „Jetzt verkaufen“, „Noch ein Jahr“ oder „Doch behalten“ reagieren.
11. Als Nutzer will ich eine Erinnerung als erledigt markieren, ohne etwas zu ändern.
12. Als Nutzer will ich erledigte Erinnerungen weiterhin in einem Verlauf sehen.
13. Als Nutzer will ich auf der Detailseite eines Items sehen, wann die nächste Erinnerung fällig ist, und die Erinnerungen zu diesem Item.
14. Als Nutzer will ich in der Mail einen Link, der direkt zur Erinnerung bzw. zum Item in der App führt.
15. Als Nutzer will ich bei vielen gleichzeitig fälligen Erinnerungen eine einzige Mail statt Dutzender.
16. Als Nutzer will ich, dass Erinnerungen, die fällig wurden, während der Pi aus war, nach dem Start nachgeholt werden, aber nur einmal.
17. Als Nutzer will ich, dass eine Erinnerung entfällt, wenn ich den Verbleib vorher ändere oder das Item verkaufe oder lösche.
18. Als Betreiber will ich per Kommandozeile eine Test-Mail verschicken, damit ich die Gmail-Einrichtung prüfen kann.
19. Als Betreiber will ich, dass ein fehlgeschlagener Mailversand später erneut versucht und im Log vermerkt wird, ohne dass die Erinnerung in der App fehlt.

## Implementation Decisions

- **Begriffe wie in `CONTEXT.md`.** Neu ist nur der technische Begriff **Termin**: die nächste geplante Fälligkeit einer Erinnerung für ein Item. Eine Erinnerung entsteht erst, wenn ihr Termin fällig wird.
- **Datenmodell:**
  - **Termin:** Item, Art (Behalte-Prüfung, Später verkaufen; Spec 08 ergänzt Preiserinnerung und Preisplan-Stufe), fällig am (Datum in der Zeitzone des Haushalts). Höchstens ein Termin je Item und Art.
  - **Erinnerung:** Item, Art, fällig am, ausgelöst am, erledigt am, erledigt von (Nutzer), Reaktion (z. B. „weiter behalten“). Sie gilt für den ganzen Haushalt: Wer sie erledigt, erledigt sie für alle. *(Annahme)*
  - **Mailversand:** Erinnerung, Empfänger (Nutzer), Zustand (ausstehend, gesendet, fehlgeschlagen), Versuche, letzter Fehler, gesendet am.
  - Item um **Verbleib seit** erweitern: Zeitpunkt der letzten Änderung des Verbleibs. Bestehende Items bekommen bei der Migration ihr „geändert am“. *(Annahme)*
  - Beim Löschen eines Items verschwinden Termine, Erinnerungen und Mailversand-Einträge.
- **Wann Termine entstehen** (beim Setzen des Verbleibs, in derselben Transaktion):
  - Verbleib Behalten: Behalte-Prüfung am Tag „Verbleib seit“ + 3 Jahre.
  - Verbleib Später verkaufen: Erinnerung am Tag „Verbleib seit“ + 1 Jahr.
  - Bei jedem Wechsel des Verbleibs werden die Termine des alten Verbleibs gelöscht und die des neuen angelegt. Ein Speichern mit unverändertem Verbleib ändert nichts. *(Annahme)*
  - Offene Erinnerungen des alten Verbleibs werden beim Wechsel automatisch erledigt (Reaktion „Verbleib geändert“). *(Annahme)*
  - Beim Markieren als verkauft oder verschenkt (Spec 06) werden alle Termine gelöscht und offene Erinnerungen erledigt. „Verkauf rückgängig“ legt die Termine für den Verbleib neu an, ab dem Tag des Rückgängigmachens. *(Annahme)*
  - Fällt ein Termin auf den 29. Februar, gilt in Nicht-Schaltjahren der 28. Februar.
- **Auslösen eines Termins** (in einer Transaktion): Erinnerung anlegen, Empfänger bestimmen und je Empfänger einen ausstehenden Mailversand anlegen, dann bei der Behalte-Prüfung den nächsten Termin auf *heute* + 3 Jahre setzen (Frist beginnt neu, unabhängig von einer Reaktion) und den Termin Später verkaufen löschen (einmalig). *(Annahme: Später verkaufen erinnert nur einmal)*
  - Nachgeholte Termine nach einem Ausfall werden einmal ausgelöst. Die nächste Behalte-Prüfung rechnet dann vom Auslösetag, nicht vom ursprünglichen Fälligkeitstag. *(Annahme)*
- **Empfängerregel** (zum Zeitpunkt des Auslösens):
  - Empfänger sind alle Besitzer des Items, die Nutzer sind.
  - Ist keiner der Besitzer Nutzer, sind es alle Nutzer.
  - Besitzer ohne Nutzerzugang bekommen nie eine Mail.
- **Reaktionen** (erledigen jeweils die Erinnerung):
  - Behalte-Prüfung: „Weiter behalten“ (keine Änderung, der Termin läuft schon) oder „Verkaufen“ (Verbleib Verkaufen, wie Item bearbeiten).
  - Später verkaufen: „Jetzt verkaufen“ (Verbleib Verkaufen), „Noch ein Jahr“ (neuer Termin heute + 1 Jahr, Verbleib bleibt), „Doch behalten“ (Verbleib Behalten).
  - „Erledigt“ ohne Änderung bei jeder Art.
  - Reaktionen auf eine bereits erledigte Erinnerung werden mit „Erinnerung erledigt“ abgelehnt.
- **Erinnerungs-Dienst im Server-Prozess** (ADR-0001, ein Prozess):
  - Er läuft als Goroutine des Server-Unterbefehls, beim Start sofort und danach alle 15 Minuten. *(Annahme)*
  - Ein Lauf löst alle Termine aus, deren Tag erreicht ist und deren Uhrzeit ab 8:00 Uhr in der Zeitzone des Haushalts liegt, damit keine Mails um Mitternacht kommen. *(Annahme)*
  - Danach verschickt er alle ausstehenden Mails, gebündelt als **eine Sammelmail je Empfänger und Lauf**. *(Annahme)*
  - Fehlgeschlagene Mails werden in späteren Läufen erneut versucht, höchstens 5-mal, und dann als fehlgeschlagen im Log vermerkt. *(Annahme)*
  - Die Uhr ist die bestehende Abhängigkeit `Jetzt` des Servers. Ein Lauf ist eine eigene Funktion, die Tests direkt anstoßen können, statt auf den Takt zu warten.
  - Zwei gleichzeitige Läufe dürfen nichts doppelt auslösen oder senden: Läufe sind serialisiert, und jeder Termin und jeder Mailversand wird in einer Transaktion umgestellt.
- **Mail:**
  - Betreff z. B. „Inventar: 3 Erinnerungen“ bzw. bei einer einzigen „Inventar: Behalte-Prüfung für Schreibtisch“.
  - Klartext mit einer Zeile je Erinnerung (Art, Name des Items, Besitzer) und einem Link zur Erinnerung in der App. Fotos werden nicht angehängt. *(Annahme)*
  - Die Adresse für Links kommt aus der DuckDNS-Domain der Konfiguration (`https://<domain>`). Ohne Domain enthält die Mail keine Links. *(Annahme)*
  - Ohne SMTP-Konfiguration nutzt der Dienst die Log-Implementierung des Mailers aus Spec 01. Der Mailversand gilt dann als gesendet.
- **Zeitzone:** die aus Spec 06.
- **Unterbefehl „Test-Mail“:** schickt eine Mail an eine angegebene Adresse über die konfigurierte Implementierung und meldet einen Fehler verständlich.
- **API:**
  - Erinnerungen auflisten: offen oder erledigt, optional nur die eigenen (an den angemeldeten Nutzer gerichteten), neueste zuerst, seitenweise. Jedes Element enthält Item (Name, Vorschau, Besitzer), Art, fällig am und Empfänger.
  - Anzahl offener Erinnerungen, für die Startseite.
  - Erinnerung abrufen, auf sie reagieren.
  - Item-Abruf enthält die anstehenden Termine und die Erinnerungen des Items.
  - Fehler mit fachlichen Codes: „Erinnerung erledigt“, „Reaktion passt nicht“ (z. B. „Noch ein Jahr“ bei einer Behalte-Prüfung).
- **Client:**
  - Erinnerungsliste mit Umschalter „Meine“ / „Alle“ und Verlauf.
  - Zähler offener Erinnerungen auf der Startseite und in der Navigation.
  - Reaktionsknöpfe je Art. „Verkaufen“ führt danach zur Eingabe des Angebotspreises (Spec 06).
  - Item-Detailseite mit „Nächste Behalte-Prüfung am …“ bzw. „Verkaufen ab …“ und den bisherigen Erinnerungen.
  - Die Mail-Links öffnen die Erinnerung, nach der Anmeldung falls nötig.
  - Keine Web-Push-Benachrichtigungen. *(Annahme)*

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01, dem aufzeichnenden Mailer und einer steuerbaren Uhr. Der Test-Helfer bietet zusätzlich „Erinnerungslauf jetzt ausführen“, weil der Dienst zeitgesteuert ist und sich über HTTP nicht anstoßen lässt. Alles andere wird über HTTP und die aufgezeichneten Mails geprüft.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item wird am 1. März 2026 auf Behalten gesetzt. Nach Vorstellen der Uhr auf den 1. März 2029, 8:00 Uhr, und einem Lauf gibt es eine offene Behalte-Prüfung und eine Mail an den Besitzer. Nach einem weiteren Lauf kommt keine zweite Mail.
  - Nach „Weiter behalten“ und nach gar keiner Reaktion ist die nächste Behalte-Prüfung gleichermaßen am 1. März 2032.
  - Ein Item, das nur einem Kind ohne Nutzerzugang gehört, löst eine Mail an alle Nutzer aus. Ein Item von Anna (Nutzerin) und dem Kind löst nur eine Mail an Anna aus.
  - Wird der Verbleib vor Ablauf des Jahres von Später verkaufen auf Verkaufen geändert, entsteht keine Erinnerung.
  - Zehn gleichzeitig fällige Erinnerungen für dieselbe Nutzerin ergeben genau eine Mail.
  - Schlägt der Mailversand fehl, ist die Erinnerung trotzdem in der App, und der nächste Lauf versucht es erneut.
- **Schwerpunkte:**
  - Empfängerregel in allen Kombinationen
  - Neubeginn der Frist bei der Behalte-Prüfung unabhängig von der Reaktion
  - Einmaligkeit beim Auslösen und Senden, auch bei Nachholen nach Ausfall
  - Wechsel des Verbleibs, Verkauf, Rückgängig und Löschen entfernen bzw. erneuern Termine
  - Uhrzeit- und Zeitzonengrenze (8:00 Uhr Ortszeit, Sommer- und Winterzeit), 29. Februar
  - Reaktionen je Art und Ablehnung unpassender Reaktionen
  - Sammelmail und Wiederholung nach Fehlern
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 05 mit steuerbarer Uhr.

## Out of Scope

- Preiserinnerung und Preisplan (Spec 08), die aber denselben Dienst nutzen
- Web-Push und andere Kanäle außer Mail und App
- Einstellbare Fristen (drei Jahre und ein Jahr sind fest, siehe `CONTEXT.md`)
- Abbestellen von Mails je Nutzer
- HTML-Mails
- Erinnerungen zu Umzügen oder Boxen

## Further Notes

- ADR-0001 hält fest, dass der Pi dauerhaft laufen muss. Das Nachholen nach einem Ausfall mildert das, ersetzt es aber nicht.
- Spec 05 führt die steuerbare Uhr für Tests ein. Im Server gibt es sie bereits als `Jetzt`. Falls Spec 05 noch nicht umgesetzt ist, macht diese Spec sie in den Test-Helfern steuerbar.
- Der Mailversand braucht das Gmail-App-Passwort aus `.scratch/pi-betrieb/issues/06-gmail-app-passwort.md`. Die Entwicklung hängt nicht davon ab.
- Alle mit *(Annahme)* markierten Punkte hat Claude ohne Rückfrage festgelegt. Sie stehen unten unter „Offene Fragen“.

## Offene Fragen

1. **Bezugsdatum für bestehende Items:** Ab wann laufen die drei Jahre für Items, die schon vor dieser Spec Behalten waren? Ab ihrem letzten Änderungsdatum, ab dem Tag der Einführung oder ab dem Anlegen?
2. **Bezugsdatum allgemein:** Zählen die drei Jahre bzw. das eine Jahr ab dem Setzen des Verbleibs (Annahme) oder ab dem Anlegen des Items?
3. **Wechsel des Verbleibs:** Soll ein Wechsel, z. B. von Behalten auf Verkaufen und zurück, die Frist neu starten (Annahme) oder die alte Frist fortsetzen?
4. **Später verkaufen ignoriert:** Erinnert die App nur einmal (Annahme), oder wiederholt sie die Erinnerung, z. B. monatlich, bis man reagiert?
5. **„Noch ein Jahr“ und „Doch behalten“:** Sind diese Reaktionen gewünscht?
6. **Erledigt für alle:** Reicht es, wenn ein Nutzer die Erinnerung erledigt, oder hat jeder Empfänger seinen eigenen Zustand?
7. **Uhrzeit:** Ist 8:00 Uhr Ortszeit für den Versand gut? Sollen Mails gebündelt werden, und wenn ja, je Lauf (Annahme) oder als tägliche bzw. wöchentliche Zusammenfassung?
8. **Nachholen nach Ausfall:** Rechnet die nächste Behalte-Prüfung vom Auslösetag (Annahme) oder vom ursprünglichen Fälligkeitstag?
9. **Links in Mails:** Ist die DuckDNS-Domain die richtige Adresse, oder braucht es eine eigene Einstellung, z. B. mit Port?
10. **Mailinhalt:** Soll das Hauptfoto in der Mail sein? Klartext reicht (Annahme)?
11. **Neue Nutzer:** Bekommt ein Nutzer, der nach dem Auslösen eingeladen wurde, ältere Erinnerungen nachträglich? (Annahme: nein, Empfänger werden beim Auslösen festgelegt.)
12. **Abbestellen:** Soll ein Nutzer Mails abschalten können, z. B. nur noch in der App?
13. **Web-Push:** Reichen Mail und App, oder sind Benachrichtigungen aufs Handy gewünscht?

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration für Termin (Item, Art, fällig am; eindeutig je Item und Art), Erinnerung (Item, Art, fällig, ausgelöst, erledigt, erledigt von, Reaktion) und Mailversand (Erinnerung, Empfänger, Zustand, Versuche, Fehler). Item um „Verbleib seit“ erweitern, bestehende Items mit „geändert am“ vorbelegen.
- [ ] R002: Termine beim Setzen des Verbleibs anlegen, ersetzen und löschen (Behalten +3 Jahre, Später verkaufen +1 Jahr, 29. Februar), offene Erinnerungen beim Wechsel erledigen. Verkauf, Verschenken, Rückgängig (Spec 06) und Löschen berücksichtigen.
- [ ] R003: Erinnerungslauf: fällige Termine ab 8:00 Uhr Ortszeit in einer Transaktion auslösen, Empfänger nach der Empfängerregel bestimmen, Behalte-Prüfung auf heute +3 Jahre neu planen, Später verkaufen einmalig. Läufe serialisiert, nichts doppelt.
- [ ] R004: Mailversand im Lauf: Sammelmail je Empfänger mit Links zur App (aus der DuckDNS-Domain), Wiederholung nach Fehlern bis 5 Versuche, Log bei endgültigem Fehlschlag.
- [ ] R005: Erinnerungs-Dienst als Goroutine des Servers: Lauf beim Start und alle 15 Minuten, sauberes Beenden mit dem Server. Lauf in den Test-Helfern direkt auslösbar.
- [ ] R006: Erinnerungs-API: auflisten (offen oder erledigt, „meine“ oder alle, seitenweise), Anzahl offener, abrufen, reagieren mit Prüfung der Reaktion je Art. Item-Abruf mit Terminen und Erinnerungen.
- [ ] R007: Unterbefehl „Test-Mail“ mit verständlicher Fehlermeldung.
- [ ] R008: Client: Erinnerungsliste mit „Meine“ / „Alle“ und Verlauf, Zähler auf Startseite und Navigation, Reaktionsknöpfe, Weiterleitung zum Angebotspreis nach „Verkaufen“.
- [ ] R009: Client: Item-Detailseite mit nächstem Termin und bisherigen Erinnerungen, Mail-Links öffnen die Erinnerung.
- [ ] R010: API-Tests mit steuerbarer Uhr und aufzeichnendem Mailer: Empfängerregel, Neubeginn der Frist ohne und mit Reaktion, Einmaligkeit, Nachholen, Wechsel des Verbleibs, Verkauf und Löschen, Uhrzeit- und Zeitzonengrenzen, Sammelmail, Wiederholung nach Fehlern, Reaktionen.
