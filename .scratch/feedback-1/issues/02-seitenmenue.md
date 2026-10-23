# Seitenmenü statt Knopfleiste auf der Startseite

Status: ready-for-agent
Typ: Feature

## Was

Die Punkte „Umzug“, „Wohnungen“, „Ohne Lagerort“ (und sinnvollerweise auch „Personen“, „Nutzer“, „Abmelden“) stehen nicht mehr als Knöpfe im Kopf der Startseite, sondern in einem Seitenmenü.

- Öffnen per Wisch vom linken Bildschirmrand nach rechts.
- Zusätzlich über ein Menü-Icon (☰) im Kopf.
- Schließen per Wisch nach links, Tipp daneben oder Escape.

## Hinweise

- Auf iOS ist der Wisch vom linken Rand in Safari die Zurück-Geste; in der installierten PWA (standalone) ist er frei. Im Browser bleibt das Icon der verlässliche Weg.
- Barrierefrei: Menü-Knopf mit `aria-expanded`, Fokus ins Menü, Escape schließt.

## Fertig, wenn

- [ ] Menü per Icon und per Rand-Wisch zu öffnen, die genannten Punkte darin, Kopf der Startseite aufgeräumt.
