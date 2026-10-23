# Verbleib beim ersten Item nicht übernommen

Status: needs-info
Typ: Fehler

## Beobachtet

Beim ersten angelegten Item wurde der gewählte Verbleib nicht übernommen.

## Stand

Im Code (`web/src/lib/VerbleibAuswahl.svelte`, `web/src/routes/items/neu/+page.svelte`) ist keine offensichtliche Ursache zu sehen; die API-Tests für das Anlegen mit Verbleib sind grün.

## Rückfragen an den Nutzer

- Wo wurde der Verbleib gewählt: beim Erfassen oder später auf der Detailseite unter „Bearbeiten“?
- Was wurde danach angezeigt: „Offen“ in der Liste, auf der Detailseite oder beides?
- Tritt es reproduzierbar auf, z. B. immer beim ersten Item nach dem Öffnen der Seite?
- Welches Gerät und welcher Browser?

## Fertig, wenn

- [ ] Reproduziert, Ursache gefunden, behoben, Regressionstest.
