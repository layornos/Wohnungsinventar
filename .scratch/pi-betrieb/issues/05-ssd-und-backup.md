# SSD für Daten und Backup außerhalb des Pi

Status: erledigt

Blockiert die Entwicklung nicht.

## Was

1. Eine SSD an den Pi 5 anschließen (USB oder NVMe-HAT), auf der später Datenbank und Fotos liegen.
2. Ein Backup-Ziel außerhalb des Pi festlegen, z. B. ein NAS, den USB-Speicher an der FritzBox oder einen Cloud-Speicher.

## Warum

SQLite auf einer SD-Karte ist anfällig für Verschleiß und Datenverlust (ADR-0002).

## Fertig, wenn

- [x] Die SSD ist eingebunden, und der Mount-Pfad ist hier notiert: `/` (NVMe-SSD, System läuft schon davon); Daten unter `/home/layornos/inventar/data`
- [x] Das Backup-Ziel ist festgelegt und hier notiert: `/mnt/NAS/Backup/inventar` (NAS-Platte am Pi, nächtlich per `inventar-backup.timer`)

Das eigentliche Backup-Skript wird ein eigenes Entwicklungsticket.

## Comments

2026-10-24: Backup-Befehl und Timer sind umgesetzt (deploy/). Die NAS-Platte hängt am selben Pi; gegen Ausfall des ganzen Geräts schützt das nicht.
