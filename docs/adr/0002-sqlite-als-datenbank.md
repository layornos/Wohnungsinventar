# SQLite als Datenbank

Die Daten liegen in einer einzigen SQLite-Datei auf dem Pi und nicht in PostgreSQL. Bei einem einzigen Haushalt mit wenigen gleichzeitigen Nutzern reicht SQLite gut aus. Es braucht keinen separaten Datenbankdienst auf dem Pi, und ein Backup ist eine Dateikopie. Fotos werden als Dateien neben der Datenbank abgelegt, nicht als BLOBs darin, damit die Datenbank klein und schnell zu sichern bleibt.

## Consequences

- Wer später auf PostgreSQL umsteigen will, muss migrieren. Der Datenbankzugriff sollte deshalb hinter einer schmalen Schnittstelle liegen und Standard-SQL bevorzugen.
- SQLite auf einer SD-Karte ist anfällig für Verschleiß und Datenverlust. Daten und Fotos sollten auf einer SSD liegen, und es braucht regelmäßige Backups außerhalb des Pi.
