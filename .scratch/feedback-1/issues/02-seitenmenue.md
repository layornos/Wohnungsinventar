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

## Comments

- Umgesetzt in `web/src/lib/Seitenmenue.svelte` als modaler `<dialog>` am linken Rand: ☰-Knopf im Kopf der Startseite (`aria-expanded`, `aria-controls`), Wisch ab den linken 24 px öffnet, Wisch nach links, Tipp auf den Hintergrund oder Escape schließt. Der Dialog hält den Fokus im Menü und gibt ihn beim Schließen an den Knopf zurück. Keine neue Bibliothek. Im Browser auf einem Gerät noch nicht ausprobiert, vor allem der Rand-Wisch in der installierten PWA auf iOS.

2026-10-24, nach Test auf dem iPhone: In der installierten App ist der Wisch vom linken Rand die Zurück-Geste von iOS; beim Zurückgehen ging jedes Mal das Menü auf, auch mit Verzögerung. Auf Wunsch des Nutzers ist das Öffnen per Wisch entfernt. Das Menü öffnet nur noch über einen schwebenden ☰-Knopf unten links (Daumenbereich), gegenüber von „Erfassen“. Schließen per Wisch nach links, Tipp daneben oder Escape bleibt.
