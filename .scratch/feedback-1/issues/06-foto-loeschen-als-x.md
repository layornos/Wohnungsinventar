# Foto löschen über ein ×-Symbol am Bild

Status: ready-for-agent
Typ: Feature

## Was

Auf der Detailseite entfällt der Knopf „Foto löschen“. Stattdessen erscheint auf dem ausgewählten Foto rechts oben ein runder Knopf mit einem ×. Ein Tipp darauf löscht das Foto (mit Rückfrage wie bisher).

## Hinweise

- Ausreichend große Tippfläche (mindestens 44 px), gut sichtbar auf hellen und dunklen Fotos, `aria-label="Foto löschen"`.
- Beim einzigen Foto bleibt das × weg oder ist deaktiviert, weil der Server das letzte Foto ohnehin ablehnt („letztes_foto“).

## Fertig, wenn

- [ ] ×-Knopf am ausgewählten Foto löscht es nach Rückfrage, der bisherige Knopf ist entfernt.

## Comments

- Umgesetzt: Der Knopf „Foto löschen“ in der Knopfzeile ist entfernt. Am großen Foto sitzt rechts oben ein runder ×-Knopf (44 × 44 px, halbtransparent dunkel mit hellem Rand und Schatten, `aria-label="Foto löschen"`), der über `fotoLoeschen` mit der bisherigen Rückfrage löscht. Bei nur einem Foto wird das × nicht angezeigt. Änderungen in `web/src/routes/items/[id]/+page.svelte` und `web/src/app.css` (`.grossfoto`, `.grossfoto button.foto-loeschen`).
