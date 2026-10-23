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
