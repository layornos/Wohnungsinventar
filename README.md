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
go run ./cmd/inventar create-user --name "Alex" --email alex@example.org
```

Für die Arbeit am Client mit Hot Reload laufen Go-Server (`go run ./cmd/inventar serve`) und `npm run dev` in `web/` nebeneinander. Vite leitet `/api` an `localhost:8080` weiter.

Ohne DuckDNS-Angaben läuft der Server per HTTP. Browser behandeln `localhost` als sichere Umgebung, PWA und Kamera funktionieren dort also auch ohne Zertifikat. Ohne SMTP-Angaben schreibt der Server Mails nur ins Log.

### Browser-Tests

```sh
make e2e     # baut bin/inventar, installiert Chromium für Playwright bei Bedarf und führt die Tests aus
```

Die Playwright-Tests liegen in [`e2e/`](e2e/), mit eigener `package.json`, damit der Client in `web/` ohne Testabhängigkeiten gebaut wird. Jeder Test-Worker startet `bin/inventar serve` auf einem freien Port mit leerem Datenverzeichnis und einem Testnutzer. Jeder Test läuft in Handy- (390×844) und Desktop-Größe (1280×800); Konsolenfehler und unerwartete Fehlerantworten des Servers lassen ihn scheitern. Mit `E2E_SKIP_BUILD=1` nutzen die Tests ein vorhandenes Binary; einzelne Tests: `cd e2e && npx playwright test tests/move.spec.ts --project desktop`, Bericht: `npx playwright show-report`.

Bei jedem Push und Pull Request laufen `make check` und `make e2e` als GitHub Action ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)); schlägt etwas fehl, hängt der Playwright-Bericht am Lauf.

## Betrieb auf dem Pi

Auf dem Pi (`kodi`, Debian 12 arm64) läuft der Server als systemd-Dienst per HTTP auf `127.0.0.1:8080`. Davor terminiert ein gemeinsamer Caddy für alle Dienste TLS mit DuckDNS-Zertifikaten (ADR-0005). Alle Dateien dafür liegen in [`deploy/`](deploy/).

### Einmalige Einrichtung

Auf dem Pi:

```sh
mkdir -p ~/inventar/bin ~/inventar/data
```

Vom Entwicklungsrechner aus `deploy/inventar.env.example` als `~/inventar/inventar.env` auf den Pi kopieren, anpassen und mit `chmod 600` schützen. Dann die Units installieren:

```sh
scp deploy/inventar.env.example layornos@kodi:inventar/inventar.env
scp deploy/inventar.service deploy/inventar-backup.service deploy/inventar-backup.timer layornos@kodi:/tmp/
ssh layornos@kodi 'chmod 600 ~/inventar/inventar.env && sudo mv /tmp/inventar*.service /tmp/inventar-backup.timer /etc/systemd/system/ && sudo systemctl daemon-reload'
deploy/deploy.sh   # baut, kopiert das Binary und startet den Dienst
ssh layornos@kodi 'sudo systemctl enable inventar.service && mkdir -p /mnt/NAS/Backup/inventar && sudo systemctl enable --now inventar-backup.timer'
```

Den Site-Block aus [`deploy/Caddyfile.snippet`](deploy/Caddyfile.snippet) in die Caddyfile übernehmen. Der Caddy-Container braucht das Plugin `caddy-dns/duckdns` und die Umgebungsvariablen `INVENTAR_DOMAIN` (z. B. `inventar-xyz.duckdns.org`) und `DUCKDNS_TOKEN`. Die Hinweise aus ADR-0004 zu DNS-Rebind-Schutz, VPN-DNS und fester IP gelten weiter.

Danach den ersten Nutzer anlegen:

```sh
ssh -t layornos@kodi 'set -a; . ~/inventar/inventar.env; ~/inventar/bin/inventar create-user --name "Alex" --email alex@example.org'
```

### Updates

```sh
deploy/deploy.sh                                    # Ziel per INVENTAR_DEPLOY_HOST änderbar
```

Das Skript baut mit `make build-pi`, lädt das Binary unter einem temporären Namen hoch, ersetzt das alte und startet den Dienst neu. Es lässt sich beliebig oft ausführen. Das Binary enthält den Client und alle Migrationen; die Datenbank wird beim Start automatisch migriert.

### Konfiguration

| Variable | Bedeutung | Standard |
|---|---|---|
| `INVENTAR_DATA_DIR` | Verzeichnis für Datenbank (`inventar.db`), Fotos (Ordner `fotos`) und ggf. Zertifikate. Sollte auf der SSD liegen. | `./data` |
| `INVENTAR_ADDR` | Adresse und Port. Hinter dem Proxy `127.0.0.1:8080`. | `:443` mit DuckDNS, sonst `localhost:8080` |
| `INVENTAR_BEHIND_PROXY` | `true`, wenn ein Reverse-Proxy davor TLS terminiert (ADR-0005). Der Server behandelt Anfragen dann als HTTPS (`Secure`-Cookie, `https`-Einladungslinks) und vertraut `X-Forwarded-Proto`. Nur setzen, wenn der Server ausschließlich über den Proxy erreichbar ist. Nicht mit DuckDNS kombinierbar. | `false` |
| `INVENTAR_MAX_PHOTO_MB` | Größte erlaubte Fotodatei in MB | `20` |
| `INVENTAR_DUCKDNS_DOMAIN` | z. B. `inventar-xyz.duckdns.org`. Der Server holt das Zertifikat selbst (ADR-0004), nur ohne Proxy. | – |
| `INVENTAR_DUCKDNS_TOKEN` | DuckDNS-Token, Pflicht zusammen mit der Domain | – |
| `INVENTAR_SMTP_HOST` | z. B. `smtp.gmail.com`. Aktiviert den Mailversand. | – |
| `INVENTAR_SMTP_PORT` | | `587` |
| `INVENTAR_SMTP_USER` | Gmail-Adresse | – |
| `INVENTAR_SMTP_PASSWORD` | Gmail-App-Passwort | – |
| `INVENTAR_SMTP_FROM` | Absender | `INVENTAR_SMTP_USER` |
| `INVENTAR_ANTHROPIC_API_KEY` | API-Schlüssel für die Claude API. Aktiviert Vorschläge für Name und Beschreibung aus dem ersten Foto beim Erfassen; das Foto wird dafür an Anthropic geschickt (ADR-0006). Ohne Schlüssel ist die Funktion aus. | – |
| `INVENTAR_ANTHROPIC_MODEL` | Claude-Modell für die Vorschläge, nur zusammen mit dem Schlüssel | `claude-haiku-5-5` |

Token, Passwörter und der API-Schlüssel gehören in die Umgebungsdatei auf dem Pi, nicht ins Repository. Unvollständige oder widersprüchliche Angaben, z. B. eine Domain ohne Token oder `INVENTAR_BEHIND_PROXY` zusammen mit DuckDNS, verhindern den Start mit einer klaren Fehlermeldung.

Ohne Proxy kann der Server TLS auch selbst übernehmen: Mit DuckDNS holt er das Let's-Encrypt-Zertifikat per DNS-01 und erneuert es. Der Pi muss dafür von außen nicht erreichbar sein. Damit Port 443 ohne root nutzbar ist: `sudo setcap cap_net_bind_service=+ep inventar-linux-arm64`.

### Befehle

```sh
inventar serve
inventar create-user --name NAME --email EMAIL
inventar reset-password --email EMAIL           # beendet alle Sitzungen des Nutzers
inventar backup --to DIR                        # Sicherung von Datenbank und Fotos nach DIR
```

### Backup

`inventar backup --to DIR` liest wie `serve` das Datenverzeichnis aus `INVENTAR_DATA_DIR` und schreibt nach `DIR` eine konsistente Sicherung im selben Aufbau: `DIR/inventar.db` als Schnappschuss per `VACUUM INTO` (geht auch, während der Server läuft) und `DIR/fotos/`. Fotos werden nur kopiert, wenn sie neu sind oder sich Größe oder Änderungszeit geändert haben, nächtliche Läufe sind also billig. Gelöschte Fotos bleiben in der Sicherung liegen. Zertifikate werden nicht gesichert, sie lassen sich neu holen.

Auf dem Pi startet `inventar-backup.timer` die Sicherung jede Nacht gegen 03:30 nach `/mnt/NAS/Backup/inventar`. War der Pi um diese Zeit aus, wird sie beim nächsten Start nachgeholt. Den letzten Lauf zeigt `systemctl status inventar-backup.service`, von Hand startet ihn `sudo systemctl start inventar-backup.service`. Die Sicherung enthält immer nur den letzten Stand; ältere Stände braucht es auf dem NAS (z. B. Snapshots).

### Wiederherstellen

```sh
sudo systemctl stop inventar.service
mv ~/inventar/data ~/inventar/data.alt                    # alten Stand aufheben
mkdir -p ~/inventar/data
cp /mnt/NAS/Backup/inventar/inventar.db ~/inventar/data/
cp -a /mnt/NAS/Backup/inventar/fotos ~/inventar/data/
sudo systemctl start inventar.service
```

Danach in der App prüfen, ob Items und Fotos da sind, und erst dann `~/inventar/data.alt` löschen. Fotos, die nach der Sicherung gelöscht wurden, liegen danach wieder im Ordner, werden aber von keinem Item mehr verwendet; das stört nicht.

### Update von einer Version mit deutschen Befehlen

Beim Update von einer Version mit deutschen Befehlen (`starten`, `nutzer-anlegen`, `passwort-zuruecksetzen`) Aufrufe und Dienstdatei anpassen und `INVENTAR_MAX_FOTO_MB` in `INVENTAR_MAX_PHOTO_MB` umbenennen. Die Datenbank wird beim Start automatisch migriert (englische Tabellennamen, Daten bleiben erhalten); alle Nutzer müssen sich danach einmal neu anmelden.
