# Styles und Designliste aufräumen

Status: ready-for-agent
Typ: Aufräumen

## Was

Aus dem Code-Review nach Feedback-Runde 1:

- `app.css` (rund 1500 Zeilen) ändert sich aus zwei Gründen: Komponenten und Designs. Die Abweichungen je Design in eigene Dateien auslagern (z. B. `theme-neon.css`, `theme-soft.css`), wie schon bei `desktop.css`.
- Die Negativliste `:root:not([data-theme='neon'], …)` für das dunkle Minimal positiv formulieren, damit ein neues Design sie nicht bricht.
- Die Pillenform in Weich über eine Klasse einschalten statt über eine lange `:not(…)`-Ausnahmeliste.
- Die Designliste steht an mehreren Stellen (Inline-Script in `app.html`, Theme-Modul im Client, Go-Konstanten, CSS, Themenfarben doppelt). Eine gemeinsame Quelle schaffen oder die Stellen zumindest im Client zusammenführen.
- Den Breakpoint 1024px nur an einer Stelle festlegen.
- Menü-Symbole aus der Menü-Komponente auslagern, typsicher statt String-Kaskade.
- Die doppelte Bearbeitungslogik um den Foto-Editor in Detail- und Erfassen-Seite zusammenführen.
- Foto-ID und Dateiname als einen Typ führen statt als zwei Werte.

## Fertig, wenn

- [ ] Ein viertes Design hinzuzufügen verlangt Änderungen an höchstens zwei Stellen im Client plus der erlaubten Werte im Server.
- [ ] `make check` grün, Aussehen in allen drei Designs unverändert.
