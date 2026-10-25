# Styles und Designliste aufräumen

Status: done
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

Aus dem Handover vom 24.10. dazugekommen:

- Klasse `box-label-sheet` in `label-sheet` umbenennen
- Etikettseite der Box auf `LabelActions.svelte` umstellen
- doppelter Knopf „Foto aufnehmen“ beim Erfassen auf dem Handy
- Leerzeichen vor „›“ in `PathView`
- doppelte Rahmen und Trennlinien in den Listen beim Packen und Auspacken
- `tmp/` in `.gitignore`

## Fertig, wenn

- [x] Ein viertes Design hinzuzufügen verlangt Änderungen an höchstens zwei Stellen im Client plus der erlaubten Werte im Server.
- [x] `make check` grün, Aussehen in allen drei Designs unverändert.

## Umsetzung

- Je Design eine Datei unter `web/src/themes/` (Tokens, Schrift, Vorschau im Designwähler, strukturelle Abweichungen); `app.css` enthält nur noch Komponenten.
- `web/src/lib/themes.ts` ist die einzige Designliste im Client, samt Browserleisten-Farben und Kachelansicht. Das Inline-Script in `app.html` bekommt sie beim Build über `hooks.server.ts`; die CSS-Dateien lädt `themes/index.ts` per Glob. Ein viertes Design: Eintrag in `themes.ts`, neue Datei in `themes/`, erlaubter Wert in `store.Theme`.
- Kacheln statt Liste über `<html data-item-view="tiles">` statt über Selektoren mit Designnamen; dunkles Minimal über `:root[data-theme='minimal']`.
- Pillen in Weich über die Tokens `--button-radius` und `--button-weight`; Feld-Knöpfe und Knöpfe auf Fotos behalten `--radius`.
- Breakpoint nur noch in einem Block in `desktop.css`; das Seitenmenü fragt, ob die Seitenleiste sichtbar ist.
- `MenuIcon.svelte` mit Typ `MenuIconName`, `photoEdit.svelte.ts` für beide Foto-Seiten, `store.PhotoRef` im Server.
- Geprüft mit Bildschirmfotos aller Seiten in allen Designs (Minimal hell und dunkel, Neon, Weich) auf Handy und Desktop: Der Umbau war pixelgleich. Sichtbar geändert haben sich nur Packen und Auspacken (ohne Doppellinien) und Erfassen auf dem Handy (Kamera-Knopf nur noch unten).
