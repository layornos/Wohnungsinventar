# Feste IP für den Pi in der FritzBox

Status: ready-for-human

Blockiert die Entwicklung nicht. Entwickelt wird lokal auf `localhost`, das Browser als sichere Umgebung behandeln.

## Was

Den Raspberry Pi 5 in der FritzBox so einstellen, dass er immer dieselbe IPv4-Adresse bekommt: *Heimnetz › Netzwerk › Gerät bearbeiten › „Diesem Netzwerkgerät immer die gleiche IPv4-Adresse zuweisen“*.

## Warum

Der DuckDNS-Eintrag zeigt auf diese private IP (ADR-0004). Ändert sie sich, ist die App nicht mehr erreichbar.

## Fertig, wenn

- [ ] Der Pi hat eine feste IPv4-Adresse, und sie ist hier notiert: `192.168.178.___`
