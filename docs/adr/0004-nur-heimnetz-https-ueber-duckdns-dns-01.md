# Nur im Heimnetz erreichbar, HTTPS über DuckDNS und DNS-01

_Teilweise abgelöst durch [ADR-0005](0005-hinter-gemeinsamem-caddy-auf-dem-pi.md): Auf dem Pi holt ein gemeinsamer Caddy das Zertifikat statt des Go-Servers. Der eigene TLS-Modus bleibt für Installationen ohne Proxy unterstützt._

Der Server ist nur im Heimnetz erreichbar. Von unterwegs kommt man über das WireGuard-VPN der FritzBox herein, es gibt keine Portfreigabe ins Internet. Die PWA braucht trotzdem HTTPS. Deshalb zeigt eine DuckDNS-Subdomain auf die *private* IP des Pi, und das Let's-Encrypt-Zertifikat wird über die DNS-01-Challenge geholt, bei der ein TXT-Record bei DuckDNS gesetzt wird. Dafür muss der Pi von außen nicht erreichbar sein. Der Go-Server holt und erneuert das Zertifikat selbst (z. B. mit certmagic und dem DuckDNS-Provider von libdns), damit es wie in ADR-0001 bei einem einzigen Prozess bleibt.

## Consequences

- Die FritzBox blockiert per DNS-Rebind-Schutz öffentliche Namen, die auf private IPs zeigen. Die DuckDNS-Domain muss dort als Ausnahme eingetragen werden, sonst lässt sie sich im Heimnetz nicht auflösen.
- Die WireGuard-Clients müssen die FritzBox als DNS-Server verwenden, damit die Domain auch über das VPN aufgelöst wird.
- Der Pi braucht eine feste IP im Heimnetz, sonst stimmt der DuckDNS-Eintrag nicht mehr.
- Der DuckDNS-Token gehört in die Server-Konfiguration und nicht ins Repository.
