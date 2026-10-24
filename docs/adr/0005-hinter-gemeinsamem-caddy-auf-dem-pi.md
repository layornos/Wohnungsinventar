# Auf dem Pi hinter einem gemeinsamen Caddy statt eigenem TLS

Auf dem Pi sind die Ports 80 und 443 schon belegt, und dort laufen mehrere Dienste, die jeweils eine eigene DuckDNS-Subdomain mit HTTPS brauchen. Statt dass jeder Dienst wie in ADR-0004 sein Zertifikat selbst holt und einen eigenen Port belegt, terminiert ein einziger Caddy (Docker, Host-Netzwerk, DNS-01 über das Plugin caddy-dns/duckdns) TLS für alle Subdomains und leitet weiter. Der Inventar-Server läuft als systemd-Dienst per HTTP nur auf `127.0.0.1:8080` und ist damit von außen nur über Caddy erreichbar. Mit `INVENTAR_BEHIND_PROXY=true` behandelt er alle Anfragen als HTTPS: Das Sitzungs-Cookie bekommt `Secure`, und der Einladungslink beginnt mit `https`. Nur in diesem Modus wertet er `X-Forwarded-Proto` aus. Ohne die Einstellung ignoriert er den Header, weil ihn sonst jeder Client selbst setzen könnte.

Das teilt ADR-0004 nur zum Teil ab: Heimnetz, VPN, DuckDNS auf die private IP und DNS-01 bleiben wie beschlossen, nur holt jetzt Caddy das Zertifikat statt des Go-Servers. Der eigene TLS-Modus aus ADR-0004 bleibt für Installationen ohne Proxy unterstützt. Beides zugleich ist ein Konfigurationsfehler, und der Server startet dann nicht.

## Consequences

- Es laufen zwei Prozesse statt einem (ADR-0001), dafür teilen sich alle Dienste auf dem Pi einen Proxy und eine Zertifikatsverwaltung.
- Der DuckDNS-Token gehört jetzt in die Caddy-Konfiguration, nicht mehr in die des Inventar-Servers.
- `INVENTAR_BEHIND_PROXY` darf nur gesetzt sein, wenn der Server ausschließlich über den Proxy erreichbar ist, also auf `127.0.0.1` lauscht. Sonst könnte ein Client im Heimnetz den Proxy umgehen.
- Fällt Caddy aus, ist die App nicht erreichbar, auch wenn der Inventar-Server läuft.
