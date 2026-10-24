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

- [x] Ein frisch aufgenommenes Foto lässt sich in 90°-Schritten drehen und frei zuschneiden, bevor das Item gespeichert wird.
- [x] Ein gespeichertes Foto lässt sich auf der Detailseite ebenso drehen und zuschneiden.

## Comments

- Umgesetzt ohne Bibliothek, nur mit Canvas: Drehen in 90°-Schritten und ein freier Zuschnitt mit vier Eckgriffen und verschiebbarem Rahmen sind wenig Code (`web/src/lib/zuschnitt.ts`, `web/src/lib/FotoBearbeiten.svelte`). Eine Bibliothek hätte vor allem eine Abhängigkeit gebracht. Bedienung per Pointer Events (Finger und Maus) und Pfeiltasten; alle Knöpfe und Griffe sind beschriftet. Die EXIF-Ausrichtung von Handy-Fotos wenden die Browser beim Zeichnen selbst an, das Ergebnis ist ein aufrechtes JPEG ohne EXIF. Ergebnisse über 16 Mio. Pixel werden verkleinert (Canvas-Grenze von iOS Safari).
- Erfassen: ✎-Knopf an jedem Foto in der Leiste, das bearbeitete Foto ersetzt das gewählte vor dem Speichern. Ohne Änderung bleibt die Originaldatei.
- Detailseite: Knopf „Drehen / Zuschneiden“ am gezeigten Foto. Neuer Endpunkt `PUT /api/fotos/{id}` (multipart, genau ein Feld `foto`, Antwort: das Item). Der Server speichert das Foto wie einen Upload unter einem neuen zufälligen Namen samt neuer Vorschau, ID, Reihenfolge und Hauptfoto bleiben, die alte Datei samt Vorschau wird gelöscht. Damit ändern sich `url` und `vorschauUrl`, die alten Adressen liefern 404.
- Tests in `internal/server/fotos_test.go`: `TestBearbeitetesFotoErsetztDasOriginal`, `TestFotoErsetzenLehntUngueltigeAnfragenAb`.
