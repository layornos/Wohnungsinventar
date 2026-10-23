# SSD für Daten und Backup außerhalb des Pi

Status: ready-for-human

Blockiert die Entwicklung nicht.

## Was

1. Eine SSD an den Pi 5 anschließen (USB oder NVMe-HAT), auf der später Datenbank und Fotos liegen.
2. Ein Backup-Ziel außerhalb des Pi festlegen, z. B. ein NAS, den USB-Speicher an der FritzBox oder einen Cloud-Speicher.

## Warum

SQLite auf einer SD-Karte ist anfällig für Verschleiß und Datenverlust (ADR-0002).

## Fertig, wenn

- [ ] Die SSD ist eingebunden, und der Mount-Pfad ist hier notiert: `__________`
- [ ] Das Backup-Ziel ist festgelegt und hier notiert: `__________`

Das eigentliche Backup-Skript wird ein eigenes Entwicklungsticket.
