# Server: Zielraum der Box, Lagerort einpacken, Einräumen, Möbelinhalt, QR-Code

Status: erledigt
Spec: `../spec.md`

## Was

Alles serverseitige aus Spec 09 nach dem API-Vertrag dort: Zielraum der Box (Migration), wirksamer Zielraum, `POST /api/boxes/{id}/pack`, `POST /api/boxes/{id}/unpack`, Möbel nehmen ihre Lagerorte mit (beim Einräumen und beim Abschluss), geänderter Abschluss, `previousPlace` in der Packliste, `GET /api/boxes/{id}/qr` (SVG).

## Fertig, wenn

- [ ] Alle Beispiele aus „Testing Decisions“ der Spec sind als HTTP-API-Tests umgesetzt und grün.
- [ ] `make check` grün.
