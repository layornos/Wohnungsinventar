# Foto nach dem Aufnehmen drehen und zuschneiden

Status: ready-for-agent
Typ: Feature

## Was

Nach dem Aufnehmen oder Auswählen eines Fotos (Erfassen und Detailseite) will der Nutzer es drehen und zuschneiden können, bevor es hochgeladen wird.

## Entschieden

- Drehen und Zuschneiden gibt es beim Erfassen **und** nachträglich an bereits gespeicherten Fotos.
- Freies Format, kein festes Seitenverhältnis.
- Nachträglich bearbeitet ersetzt das Foto das Original, samt neuer Vorschau. Die Foto-Adresse muss sich dabei ändern (siehe Issue 04).
- Bearbeitet wird im Browser (Canvas, ggf. Bibliothek).

## Fertig, wenn

- [ ] Ein frisch aufgenommenes Foto lässt sich in 90°-Schritten drehen und frei zuschneiden, bevor das Item gespeichert wird.
- [ ] Ein gespeichertes Foto lässt sich auf der Detailseite ebenso drehen und zuschneiden.
