# Spec 07: Erinnerungen, Behalte-Prüfung und Später verkaufen

Status: ready-for-agent
Blocked by: 06-verkauf

## Problem Statement

Wenn ich ein Item behalte oder erst später verkaufen will, vergesse ich es. Behaltene Dinge sammeln sich an, obwohl ich sie längst nicht mehr brauche, und was ich „in einem Jahr“ verkaufen wollte, liegt nach drei Jahren noch im Keller. Ich will, dass die App mich zur richtigen Zeit daran erinnert, per Mail und in der App, und zwar die Person, der das Item gehört, nicht den ganzen Haushalt.

## Solution

Der Server enthält einen Erinnerungs-Dienst, der regelmäßig prüft, welche Termine fällig sind. Für ein Item mit Verbleib Behalten ist alle drei Jahre eine Behalte-Prüfung fällig, für ein Item mit Verbleib Später verkaufen nach einem Jahr eine Erinnerung. Jede fällige Erinnerung erscheint sofort in der App in einer Erinnerungsliste. Einmal im Monat, am 1., bekommt jeder Besitzer, der Nutzer ist, eine Sammelmail mit seinen offenen Erinnerungen; ist keiner der Besitzer Nutzer, geht sie an alle Nutzer. In der App reagiert man mit einem Tippen, z. B. „Weiter behalten“ oder „Jetzt verkaufen“. Spec 08 nutzt denselben Dienst für Preiserinnerung und Preisplan.

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
11. Als Nutzer will ich, dass eine Erinnerung erledigt ist, wenn ich den Verbleib des Items direkt ändere.
12. Als Nutzer will ich erledigte Erinnerungen weiterhin in einem Verlauf sehen.
13. Als Nutzer will ich auf der Detailseite eines Items sehen, wann die nächste Erinnerung fällig ist, und die Erinnerungen zu diesem Item.
14. Als Nutzer will ich in der Mail einen Link, der zur Erinnerungsliste in der App führt.
15. Als Nutzer will ich höchstens eine Mail im Monat, mit allen offenen Erinnerungen darin, statt einzelner Mails.
16. Als Nutzer will ich, dass Erinnerungen und die Monatsmail, die fällig wurden, während der Pi aus war, nach dem Start nachgeholt werden, aber nur einmal.
17. Als Nutzer will ich, dass eine Erinnerung entfällt, wenn ich den Verbleib vorher ändere oder das Item verkaufe oder lösche.
18. Als Betreiber will ich per Kommandozeile eine Test-Mail verschicken, damit ich die Gmail-Einrichtung prüfen kann.
19. Als Betreiber will ich, dass ein fehlgeschlagener Mailversand später erneut versucht und im Log vermerkt wird, ohne dass die Erinnerung in der App fehlt.
20. Als Nutzer will ich Erinnerungsmails für mich abschalten können und Erinnerungen dann nur in der App sehen.

## Implementation Decisions

Geklärt mit dem Nutzer am 2026-10-25 (Grilling); die früheren Annahmen sind damit entschieden. Leitlinie wie bei Spec 06: schlicht, „nicht zu fancy“.

- **Begriffe wie in `CONTEXT.md`.** Neu ist nur der technische Begriff **Termin**: die nächste geplante Fälligkeit einer Erinnerung für ein Item. Eine Erinnerung entsteht erst, wenn ihr Termin fällig wird.
- **Datenmodell:**
  - **Termin:** Item, Art (Behalte-Prüfung, Später verkaufen; Spec 08 ergänzt Preiserinnerung und Preisplan-Stufe), fällig am (Datum in der Zeitzone des Haushalts). Höchstens ein Termin je Item und Art.
  - **Erinnerung:** Item, Art, fällig am, ausgelöst am, erledigt am, erledigt von (Nutzer, leer bei automatischem Erledigen), Reaktion. Sie gilt für den ganzen Haushalt: Wer sie erledigt, erledigt sie für alle.
  - **Monatsmail:** je Nutzer und Monat höchstens eine, mit Zustand (ausstehend, gesendet, fehlgeschlagen), Versuchen, letztem Fehler und gesendet am, und den Erinnerungen, die sie enthielt.
  - Nutzer um **Erinnerungen per Mail** (ja/nein, Standard ja) erweitern.
  - Item um **Verbleib seit** erweitern: Zeitpunkt der letzten Änderung des Verbleibs. Bestehende Items bekommen bei der Migration ihr `updated_at`.
  - Beim Löschen eines Items verschwinden seine Termine und Erinnerungen.
- **Wann Termine entstehen** (beim Setzen des Verbleibs, in derselben Transaktion):
  - Verbleib Behalten: Behalte-Prüfung am Tag „Verbleib seit“ + 3 Jahre.
  - Verbleib Später verkaufen: Erinnerung am Tag „Verbleib seit“ + 1 Jahr.
  - Bei jedem Wechsel des Verbleibs beginnt die Frist neu: Die Termine des alten Verbleibs werden gelöscht und die des neuen angelegt. Ein Speichern mit unverändertem Verbleib ändert nichts.
  - Offene Erinnerungen des alten Verbleibs werden beim Wechsel automatisch erledigt (Reaktion „Verbleib geändert“).
  - Beim Markieren als verkauft oder verschenkt (Spec 06) werden alle Termine gelöscht und offene Erinnerungen erledigt. „Zurück in den Bestand“ legt die Termine für den Verbleib neu an, ab diesem Tag.
  - Fällt ein Termin auf den 29. Februar, gilt in Nicht-Schaltjahren der 28. Februar.
- **Auslösen eines Termins** (in einer Transaktion, sobald sein Tag erreicht ist): Erinnerung anlegen; bei der Behalte-Prüfung den nächsten Termin auf den *ursprünglichen Fälligkeitstag* + 3 Jahre setzen, unabhängig von einer Reaktion und auch nach einem Ausfall; den Termin Später verkaufen löschen (einmalig). Nachgeholte Termine nach einem Ausfall werden genau einmal ausgelöst.
- **Wenn niemand reagiert:** Die Erinnerung bleibt offen in der App, bis jemand reagiert. Es gibt keine wiederholten Mails; sie steht nur in der nächsten Monatsmail, solange sie offen ist (siehe unten).
- **Reaktionen** (erledigen jeweils die Erinnerung; keinen eigenen Knopf „Erledigt“):
  - Behalte-Prüfung: „Weiter behalten“ (keine Änderung, der nächste Termin läuft schon) oder „Verkaufen“ (Verbleib Verkaufen, wie Item bearbeiten; der Client führt danach zum Angebotspreis).
  - Später verkaufen: „Jetzt verkaufen“ (Verbleib Verkaufen), „Noch ein Jahr“ (neuer Termin heute + 1 Jahr, Verbleib bleibt), „Doch behalten“ (Verbleib Behalten, die drei Jahre beginnen).
  - Wer verschenken will, wählt „Verkaufen“ und stellt den Verbleib auf der Detailseite um.
  - Reaktionen auf eine bereits erledigte Erinnerung werden mit „Erinnerung erledigt“ abgelehnt, unpassende mit „Reaktion passt nicht“.
- **Empfängerregel** (bestimmt beim Versand der Monatsmail, nicht beim Auslösen; so bekommen auch später eingeladene Nutzer noch offene Erinnerungen):
  - Empfänger sind alle Besitzer des Items, die Nutzer sind und Erinnerungen per Mail eingeschaltet haben.
  - Ist keiner der Besitzer Nutzer, sind es alle Nutzer mit eingeschalteter Mail.
  - Haben alle Besitzer, die Nutzer sind, die Mail abgeschaltet, bekommt niemand eine Mail; es wird nicht auf andere Nutzer ausgewichen.
  - Besitzer ohne Nutzerzugang bekommen nie eine Mail.
- **Monatsmail:**
  - Je Nutzer höchstens eine Mail pro Monat, am 1. ab 8:00 Uhr in der Zeitzone des Haushalts.
  - Sie enthält alle Erinnerungen, die seit der letzten Monatsmail fällig wurden, für die der Nutzer Empfänger ist und die beim Versand noch offen sind. Ist nichts offen, gibt es keine Mail.
  - War der Pi am 1. aus, wird die Mail nach dem Start nachgeholt, höchstens eine pro Monat.
  - Klartext ohne Fotos, Betreff z. B. „Inventar: 3 offene Erinnerungen“, eine Zeile je Erinnerung (Art, Name des Items, Besitzer), am Ende ein Link auf die Erinnerungsliste der App.
  - Die Adresse für den Link kommt aus der neuen Einstellung **`INVENTAR_PUBLIC_URL`** (auf dem Pi `https://mosiminventar.duckdns.org`; seit ADR-0005 kennt der Server seine öffentliche Adresse sonst nicht). Ohne sie enthält die Mail statt des Links den Hinweis „In der App unter Erinnerungen“.
  - Fehlgeschlagene Mails werden in späteren Läufen erneut versucht, höchstens 5-mal, und dann als fehlgeschlagen im Log vermerkt. Die Erinnerungen stehen unabhängig davon in der App.
  - Ohne SMTP-Konfiguration nutzt der Dienst die Log-Implementierung des Mailers. Die Mail gilt dann als gesendet.
- **Erinnerungs-Dienst im Server-Prozess** (ADR-0001, ein Prozess):
  - Er läuft als Goroutine des Server-Unterbefehls, beim Start sofort und danach alle 15 Minuten.
  - Ein Lauf löst zuerst alle fälligen Termine aus (ab 8:00 Uhr Ortszeit ihres Tages) und verschickt dann, falls fällig, die Monatsmails.
  - Die Uhr ist die bestehende Abhängigkeit `Jetzt` des Servers. Ein Lauf ist eine eigene Funktion, die Tests direkt anstoßen können.
  - Läufe sind serialisiert, und jeder Termin und jede Monatsmail wird in einer Transaktion umgestellt, damit nichts doppelt ausgelöst oder gesendet wird.
- **Zeitzone:** fest `Europe/Berlin` wie in Spec 06.
- **Unterbefehl „Test-Mail“:** schickt eine Mail an eine angegebene Adresse über die konfigurierte Implementierung und meldet einen Fehler verständlich.
- **API:**
  - Erinnerungen auflisten: offen oder erledigt, optional nur die eigenen (für die der angemeldete Nutzer Empfänger wäre), neueste zuerst, seitenweise. Jedes Element enthält Item (Name, Vorschau, Besitzer), Art, fällig am und, wenn erledigt, Reaktion, wer und wann.
  - Anzahl offener Erinnerungen, für die Startseite und das Seitenmenü.
  - Erinnerung abrufen, auf sie reagieren.
  - Item-Abruf enthält die anstehenden Termine und die Erinnerungen des Items.
  - Eigene Einstellung „Erinnerungen per Mail“ abrufen und ändern.
  - Fehler mit fachlichen Codes: „Erinnerung erledigt“, „Reaktion passt nicht“.
- **Client:**
  - Erinnerungsliste (Seite `/erinnerungen`) mit Umschalter „Meine“ / „Alle“ und Verlauf mit Reaktion, Person und Datum.
  - Zähler offener Erinnerungen auf der Startseite und im Seitenmenü.
  - Reaktionsknöpfe je Art. „Verkaufen“ und „Jetzt verkaufen“ führen danach zur Eingabe des Angebotspreises (Spec 06).
  - Item-Detailseite mit „Nächste Behalte-Prüfung am …“ bzw. „Verkaufen ab …“ und den bisherigen Erinnerungen.
  - Schalter „Erinnerungen per Mail“ unter „Einstellungen“.
  - Der Mail-Link öffnet die Erinnerungsliste, nach der Anmeldung falls nötig.

## Testing Decisions

- **Test-Seam ist die HTTP-API** mit den Test-Helfern aus Spec 01, dem aufzeichnenden Mailer und einer steuerbaren Uhr. Der Test-Helfer bietet zusätzlich „Erinnerungslauf jetzt ausführen“, weil der Dienst zeitgesteuert ist und sich über HTTP nicht anstoßen lässt. Alles andere wird über HTTP und die aufgezeichneten Mails geprüft.
- **Gute Tests prüfen beobachtbares Verhalten**, z. B.:
  - Ein Item wird am 1. März 2026 auf Behalten gesetzt. Nach Vorstellen der Uhr auf den 1. März 2029, 8:00 Uhr, und einem Lauf gibt es eine offene Behalte-Prüfung, aber noch keine Mail. Am 1. April 2029, 8:00 Uhr, kommt genau eine Mail an den Besitzer; ein weiterer Lauf schickt keine zweite.
  - Nach „Weiter behalten“ und nach gar keiner Reaktion ist die nächste Behalte-Prüfung gleichermaßen am 1. März 2032, auch wenn der Server vom 1. bis 5. März 2029 aus war.
  - Ein Item, das nur einem Kind ohne Nutzerzugang gehört, landet in der Monatsmail aller Nutzer. Ein Item von Anna (Nutzerin) und dem Kind nur in Annas Mail. Hat Anna die Mail abgeschaltet, bekommt niemand eine Mail dazu.
  - Wird der Verbleib vor Ablauf des Jahres von Später verkaufen auf Verkaufen geändert, entsteht keine Erinnerung.
  - Zehn im Lauf des Monats fällige Erinnerungen für dieselbe Nutzerin ergeben am 1. genau eine Mail; eine davon, die vorher in der App erledigt wurde, fehlt darin.
  - Ein nach dem Fälligwerden eingeladener Besitzer bekommt die noch offene Erinnerung in seiner Monatsmail.
  - Schlägt der Mailversand fehl, ist die Erinnerung trotzdem in der App, und der nächste Lauf versucht es erneut.
- **Schwerpunkte:**
  - Empfängerregel in allen Kombinationen, mit abgeschalteter Mail
  - fester Takt der Behalte-Prüfung vom ursprünglichen Fälligkeitstag, unabhängig von Reaktion und Ausfall
  - Einmaligkeit beim Auslösen und beim Versand der Monatsmail, auch beim Nachholen
  - Wechsel des Verbleibs, Verkauf, „Zurück in den Bestand“ und Löschen entfernen bzw. erneuern Termine
  - Uhrzeit- und Zeitzonengrenze (8:00 Uhr Ortszeit, Sommer- und Winterzeit, Monatswechsel), 29. Februar
  - Reaktionen je Art und Ablehnung unpassender Reaktionen
  - Wiederholung nach Fehlern
- **Client:** nur Typprüfung und Build.
- **Vorbild im Code:** die API-Tests aus Spec 05 mit steuerbarer Uhr.

## Out of Scope

- Preiserinnerung und Preisplan (Spec 08), die aber denselben Dienst nutzen
- Web-Push und andere Kanäle außer Mail und App
- Einstellbare Fristen (drei Jahre und ein Jahr sind fest, siehe `CONTEXT.md`)
- Wiederholte Mails zu einer offenen Erinnerung, tägliche oder wöchentliche Mails
- HTML-Mails
- Erinnerungen zu Umzügen oder Boxen

## Further Notes

- ADR-0001 hält fest, dass der Pi dauerhaft laufen muss. Das Nachholen nach einem Ausfall mildert das, ersetzt es aber nicht.
- Spec 05 führt die steuerbare Uhr für Tests ein. Im Server gibt es sie bereits als `Jetzt`. Falls Spec 05 noch nicht umgesetzt ist, macht diese Spec sie in den Test-Helfern steuerbar.
- Der Mailversand braucht das Gmail-App-Passwort aus `.scratch/pi-betrieb/issues/06-gmail-app-passwort.md`. Die Entwicklung hängt nicht davon ab.

## Tasks

Acceptance criteria — completing all of these delivers the spec end to end:

- [ ] R001: Migration für Termin (Item, Art, fällig am; eindeutig je Item und Art), Erinnerung (Item, Art, fällig, ausgelöst, erledigt, erledigt von, Reaktion) und Monatsmail (Nutzer, Monat, Zustand, Versuche, Fehler, enthaltene Erinnerungen). Item um „Verbleib seit“ (vorbelegt mit `updated_at`), Nutzer um „Erinnerungen per Mail“ erweitern.
- [ ] R002: Termine beim Setzen des Verbleibs anlegen, ersetzen und löschen (Behalten +3 Jahre, Später verkaufen +1 Jahr, 29. Februar), offene Erinnerungen beim Wechsel erledigen. Verkauf, Verschenken, „Zurück in den Bestand“ (Spec 06) und Löschen berücksichtigen.
- [ ] R003: Erinnerungslauf: fällige Termine ab 8:00 Uhr Ortszeit in einer Transaktion auslösen, Behalte-Prüfung auf ursprünglichen Fälligkeitstag +3 Jahre neu planen, Später verkaufen einmalig. Läufe serialisiert, nichts doppelt.
- [ ] R004: Monatsmail am 1. ab 8:00 Uhr: je Nutzer höchstens eine, mit den seit der letzten Mail fällig gewordenen, noch offenen Erinnerungen nach der Empfängerregel (beim Versand bestimmt, abgeschaltete Mail beachtet), Link aus `INVENTAR_PUBLIC_URL`, Nachholen nach Ausfall, Wiederholung nach Fehlern bis 5 Versuche, Log bei endgültigem Fehlschlag.
- [ ] R005: Erinnerungs-Dienst als Goroutine des Servers: Lauf beim Start und alle 15 Minuten, sauberes Beenden mit dem Server. Lauf in den Test-Helfern direkt auslösbar. Einstellung `INVENTAR_PUBLIC_URL` in der Konfiguration und in `deploy/`.
- [ ] R006: Erinnerungs-API: auflisten (offen oder erledigt, „meine“ oder alle, seitenweise), Anzahl offener, abrufen, reagieren mit Prüfung der Reaktion je Art. Item-Abruf mit Terminen und Erinnerungen. Eigene Mail-Einstellung abrufen und ändern.
- [ ] R007: Unterbefehl „Test-Mail“ mit verständlicher Fehlermeldung.
- [ ] R008: Client: Erinnerungsliste mit „Meine“ / „Alle“ und Verlauf, Zähler auf Startseite und im Seitenmenü, Reaktionsknöpfe, Weiterleitung zum Angebotspreis nach „Verkaufen“, Schalter „Erinnerungen per Mail“ in den Einstellungen.
- [ ] R009: Client: Item-Detailseite mit nächstem Termin und bisherigen Erinnerungen, Mail-Link öffnet die Erinnerungsliste.
- [ ] R010: API-Tests mit steuerbarer Uhr und aufzeichnendem Mailer: Empfängerregel mit abgeschalteter Mail, fester Takt ohne und mit Reaktion und nach Ausfall, Einmaligkeit, Wechsel des Verbleibs, Verkauf und Löschen, Uhrzeit-, Zeitzonen- und Monatsgrenzen, Monatsmail mit nur offenen Erinnerungen, Wiederholung nach Fehlern, Reaktionen.
