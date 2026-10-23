# Prüfen, ob die Domain über WireGuard aufgelöst wird

Status: ready-for-human
Blocked by: 03

Blockiert die Entwicklung nicht.

## Was

Bei den WireGuard-Konfigurationen der Handys und Laptops prüfen, ob als DNS-Server die FritzBox eingetragen ist (Zeile `DNS = 192.168.178.1` o. ä.). Bei Konfigurationen, die die FritzBox erzeugt hat, ist das normalerweise schon so.

## Warum

Nur dann funktioniert dieselbe Adresse auch von unterwegs (ADR-0004).

## Fertig, wenn

- [ ] Im Mobilfunknetz mit aktivem VPN liefert die Subdomain die private IP des Pi, auf jedem Gerät, das die App nutzen soll
