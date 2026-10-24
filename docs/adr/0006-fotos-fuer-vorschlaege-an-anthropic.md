# Fotos für Namens- und Beschreibungsvorschläge an Anthropic

Beim Erfassen kostet das Eintippen von Name und Beschreibung die meiste Zeit. Der Server kann deshalb aus dem ersten Foto eines neuen Items einen deutschen Namen (wie ein Eintrag in einer Inventarliste, 2 bis 5 Wörter) und einen kurzen Beschreibungssatz vorschlagen. Dafür schickt er das Foto an die Claude API von Anthropic (offizielles Go-SDK `anthropic-sdk-go`, Standardmodell `claude-haiku-5-5`, mit `INVENTAR_ANTHROPIC_MODEL` änderbar). Das ist die einzige Stelle, an der Inventardaten das Heimnetz verlassen (ADR-0004), und die Nutzer haben dem ausdrücklich zugestimmt.

Die Funktion ist nur aktiv, wenn `INVENTAR_ANTHROPIC_API_KEY` gesetzt ist. Ohne Schlüssel meldet `GET /api/features` `suggestions: false`, `POST /api/suggestions` antwortet mit `feature_disabled`, und der Client fragt gar nicht erst.

Gesendet wird pro Vorschlag genau ein Foto, nur auf Anfrage des Clients beim Erfassen: das erste Foto, vom Server auf höchstens 1280 Pixel an der langen Kante verkleinert, aufrecht gedreht und als JPEG neu kodiert (dabei fallen EXIF-Daten wie GPS-Position und Kameramodell weg), dazu eine feste deutsche Anweisung. Nicht gesendet werden Namen, Besitzer, Orte, andere Fotos oder sonstige Daten aus der Datenbank. Der Server speichert das Foto für den Vorschlag nicht; gespeichert wird es erst wie bisher beim Anlegen des Items. Der Vorschlag füllt nur leere Felder und ist bis zur Bearbeitung als „Vorschlag“ markiert.

## Consequences

- Der API-Schlüssel steht in `inventar.env` auf dem Pi (`chmod 600`), nicht im Repository, und wird nie geloggt.
- Der Pi braucht ausgehenden HTTPS-Zugang zu `api.anthropic.com`. Eingehend bleibt alles wie in ADR-0004 und ADR-0005.
- Anthropic verarbeitet die Fotos nach seinen API-Bedingungen. Wer das nicht will, lässt den Schlüssel weg; alles andere funktioniert unverändert.
- Jeder Vorschlag kostet etwas (mit `claude-haiku-5-5` deutlich unter einem Zehntel Cent pro Foto) und dauert einige Sekunden. Der Server bricht nach 30 Sekunden ab. Fehler, Ablehnungen und Timeouts zeigt der Client nur als kleinen Hinweis, Speichern geht immer.
- Bei Modellen mit serverseitigem Fallback (`claude-opus-5-5`, `claude-opus-5`, `claude-fable-5-1`, `claude-sonnet-5-5`) wird eine Ablehnung automatisch auf dem von Anthropic empfohlenen Modell wiederholt (`fallbacks: "default"`). `claude-haiku-5-5` hat keinen Fallback, eine Ablehnung bleibt dort ohne Vorschlag.
