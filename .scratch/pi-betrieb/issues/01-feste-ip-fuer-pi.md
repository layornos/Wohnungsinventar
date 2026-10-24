# Feste IP für den Pi in der FritzBox

Status: erledigt

Blockiert die Entwicklung nicht. Entwickelt wird lokal auf `localhost`, das Browser als sichere Umgebung behandeln.

## Was

Den Raspberry Pi 5 in der FritzBox so einstellen, dass er immer dieselbe IPv4-Adresse bekommt: *Heimnetz › Netzwerk › Gerät bearbeiten › „Diesem Netzwerkgerät immer die gleiche IPv4-Adresse zuweisen“*.

## Warum

Der DuckDNS-Eintrag zeigt auf diese private IP (ADR-0004). Ändert sie sich, ist die App nicht mehr erreichbar.

## Fertig, wenn

- [x] Der Pi hat eine feste IPv4-Adresse, und sie ist hier notiert: `192.168.178.87`

## Comments

2026-10-24: Der Pi bezieht die Adresse per DHCP; `mosimfilme.duckdns.org` zeigt seit Längerem auf 192.168.178.87, die feste Zuweisung in der FritzBox ist also offenbar gesetzt.
