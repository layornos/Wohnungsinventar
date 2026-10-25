# Gmail-App-Passwort für den Mailversand erstellen

Status: done

Blockiert die Entwicklung nicht. Bis es da ist, schreibt der Server Mails in der Entwicklung nur ins Log.

## Was

1. Für das Gmail-Konto, das die Erinnerungen verschicken soll, die Bestätigung in zwei Schritten aktivieren, falls sie noch aus ist.
2. Unter *Google-Konto › Sicherheit › App-Passwörter* ein App-Passwort erstellen, z. B. mit dem Namen „Inventar Pi“.
3. Das Passwort sicher ablegen. Es kommt in die Server-Konfiguration und **nicht ins Repository**.

## Fertig, wenn

- [x] Die Absenderadresse ist hier notiert: `layornos@gmail.com`
- [x] Das App-Passwort ist sicher abgelegt

## Umsetzung

Am 2026-10-25 erledigt, ohne neues App-Passwort: Der Server nutzt dasselbe Gmail-Konto und App-Passwort wie die NAS-Backup-Info (`msmtp`, Konto `gmail` in `/etc/msmtprc`). Die Werte stehen als `INVENTAR_SMTP_*` in `~/inventar/inventar.env` (Rechte 600); `inventar test-mail` kam an.

Beide Dienste hängen damit am selben App-Passwort: Wird es bei Google widerrufen, fallen Backup-Info und Erinnerungsmails gleichzeitig aus. Dann ein neues erstellen und in beiden Dateien eintragen, oder dem Server ein eigenes „Inventar Pi“ geben.
