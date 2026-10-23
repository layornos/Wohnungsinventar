# DuckDNS-Domain vom DNS-Rebind-Schutz der FritzBox ausnehmen

Status: ready-for-human
Blocked by: 02

Blockiert die Entwicklung nicht.

## Was

In der FritzBox unter *Heimnetz › Netzwerk › Netzwerkeinstellungen › DNS-Rebind-Schutz* die DuckDNS-Domain aus Ticket 02 als Hostnamen-Ausnahme eintragen.

## Warum

Die FritzBox blockiert öffentliche Namen, die auf private IPs zeigen. Ohne die Ausnahme lässt sich die Domain im Heimnetz nicht auflösen (ADR-0004).

## Fertig, wenn

- [ ] `nslookup <subdomain>` von einem Gerät im Heimnetz liefert die private IP des Pi
