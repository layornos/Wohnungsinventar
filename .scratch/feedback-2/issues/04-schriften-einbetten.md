# Schriften einbetten

Status: ready-for-agent
Typ: Verbesserung

## Was

Die Designs nutzen Inter (Minimal), Space Grotesk (Neon) und Plus Jakarta Sans (Weich). Sie sind nicht eingebettet und fehlen deshalb z. B. auf dem iPhone. Die Schriften sollen als woff2 mit der App ausgeliefert werden. Es gibt keine Anfragen an externe Server (ADR-0004), nur die nötigen Schnitte und lateinische Zeichen samt Umlauten. Eine Schrift wird nur geladen, wenn ihr Design aktiv ist. Die Lizenzen (alle SIL OFL) liegen bei.

## Fertig, wenn

- [ ] Auf einem Gerät ohne installierte Schriften sieht jedes Design wie im Entwurf aus.
- [ ] Der Service Worker cacht nur, was gebraucht wird, und die Größe bleibt im Rahmen.
