# Automatische Browser-Tests und CI

Status: ready-for-agent
Typ: Qualität

## Was

- **Browser-Tests im Repo** (Playwright): Sie starten den gebauten Server mit leerem Datenverzeichnis und einem Testnutzer. Abgedeckte Abläufe:
  - Anmelden
  - Item mit Foto erfassen
  - Platz anlegen in der Platzauswahl
  - Design wechseln
  - Menü auf einer Unterseite öffnen
  - Umzug anlegen, Box mit Zielraum anlegen, packen, Etikett, Auspacken mit „Alle hier einräumen“
  - Abmelden

  Handy- und Desktop-Größe. Fehlerhafte API-Antworten und Konsolenfehler lassen den Test scheitern.
- **GitHub Action:** Bei jedem Push und Pull Request laufen `make check` und die Browser-Tests.

## Fertig, wenn

- [ ] `make e2e` läuft lokal grün, und die Action ist grün auf GitHub.
