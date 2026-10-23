# DuckDNS-Subdomain anlegen

Status: ready-for-human
Blocked by: 01

Blockiert die Entwicklung nicht.

## Was

1. Bei duckdns.org anmelden und eine Subdomain anlegen, z. B. `inventar-xyz.duckdns.org`.
2. Als IP **die private IP des Pi** aus Ticket 01 eintragen, nicht die öffentliche.
3. Den Token sicher ablegen, z. B. im Passwortmanager. Er kommt später in die Server-Konfiguration und **nicht ins Repository**.

## Warum

Für HTTPS über die DNS-01-Challenge mit Let's Encrypt, ohne Portfreigabe (ADR-0004).

## Fertig, wenn

- [ ] Die Subdomain existiert und ist hier notiert: `__________.duckdns.org`
- [ ] Der Token ist sicher abgelegt
- [ ] `dig +short <subdomain>` von außerhalb des Heimnetzes liefert die private IP des Pi
