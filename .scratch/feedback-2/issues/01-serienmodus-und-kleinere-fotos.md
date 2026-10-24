# Serienmodus beim Erfassen und kleinere Fotos

Status: ready-for-agent
Typ: Feature

## Was

- **Serienmodus:** Nach „Speichern“ auf der Erfassen-Seite ist das Formular sofort bereit für das nächste Item. Besitzer, Platz und Verbleib bleiben stehen, und die Kamera öffnet sich mit einem Tipp. Wo der Browser es erlaubt, öffnet sie sich direkt. Ein Regal geht damit als Foto, Name, Foto, Name.
- **Kleinere Fotos:** Der Client verkleinert Fotos vor dem Hochladen, die längste Kante auf etwa 2048 px, als JPEG mit Qualität etwa 0,85. Die EXIF-Ausrichtung bleibt richtig. Das gilt beim Erfassen, beim Hinzufügen auf der Detailseite und für das Ergebnis des Foto-Editors.

## Fertig, wenn

- [ ] Zwischen zwei Items ist nur ein Tipp für die Kamera nötig, Besitzer, Platz und Verbleib bleiben erhalten.
- [ ] Ein 8-MB-Handyfoto kommt mit weniger als 1 MB auf dem Server an und steht aufrecht.
