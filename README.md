# Wohnungsinventar

Selbstgehostete Inventarverwaltung für einen Haushalt: Items, Lagerorte, Umzug, Verkauf. Ein Go-Server mit SQLite liefert eine SvelteKit-PWA aus und läuft auf einem Raspberry Pi 5. Begriffe stehen in [`CONTEXT.md`](CONTEXT.md), Architekturentscheidungen in [`docs/adr/`](docs/adr/).

## Entwicklung

Voraussetzungen: Go 1.26+, Node 22 oder 24 (LTS).

```sh
make check   # Go-Vet, Go-Tests, Typprüfung und Build des Clients
make run     # Client bauen, Server auf http://localhost:8080 starten, Daten in ./data
```

Ersten Nutzer anlegen (fragt das Passwort ab):

```sh
go run ./cmd/inventar nutzer-anlegen --name "Alex" --email alex@example.org
```

Für die Arbeit am Client mit Hot Reload laufen Go-Server (`go run ./cmd/inventar serve`) und `npm run dev` in `web/` nebeneinander. Vite leitet `/api` an `localhost:8080` weiter.

Ohne DuckDNS-Angaben läuft der Server per HTTP. Browser behandeln `localhost` als sichere Umgebung, PWA und Kamera funktionieren dort also auch ohne Zertifikat. Ohne SMTP-Angaben schreibt der Server Mails nur ins Log.

## Betrieb auf dem Pi

```sh
make build-pi   # erzeugt bin/inventar-linux-arm64
```

Das Binary auf den Pi kopieren. Es enthält den Client und alle Migrationen.

| Variable | Bedeutung | Standard |
|---|---|---|
| `INVENTAR_DATA_DIR` | Verzeichnis für Datenbank, Fotos und Zertifikate. Sollte auf der SSD liegen. | `./data` |
| `INVENTAR_ADDR` | Adresse und Port | `:443` mit DuckDNS, sonst `localhost:8080` |
| `INVENTAR_DUCKDNS_DOMAIN` | z. B. `inventar-xyz.duckdns.org`. Aktiviert HTTPS (ADR-0004). | – |
| `INVENTAR_DUCKDNS_TOKEN` | DuckDNS-Token, Pflicht zusammen mit der Domain | – |
| `INVENTAR_SMTP_HOST` | z. B. `smtp.gmail.com`. Aktiviert den Mailversand. | – |
| `INVENTAR_SMTP_PORT` | | `587` |
| `INVENTAR_SMTP_USER` | Gmail-Adresse | – |
| `INVENTAR_SMTP_PASSWORD` | Gmail-App-Passwort | – |
| `INVENTAR_SMTP_FROM` | Absender | `INVENTAR_SMTP_USER` |

Token und Passwörter gehören in eine Umgebungsdatei auf dem Pi, nicht ins Repository. Unvollständige Angaben, z. B. eine Domain ohne Token, verhindern den Start mit einer klaren Fehlermeldung.

Mit DuckDNS holt der Server das Let's-Encrypt-Zertifikat per DNS-01 selbst und erneuert es. Der Pi muss dafür von außen nicht erreichbar sein. Damit Port 443 ohne root nutzbar ist: `sudo setcap cap_net_bind_service=+ep inventar-linux-arm64`.

Befehle:

```sh
inventar serve
inventar nutzer-anlegen --name NAME --email EMAIL
inventar passwort-zuruecksetzen --email EMAIL   # beendet alle Sitzungen des Nutzers
```
