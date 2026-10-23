# Selbstgehostet auf einem Raspberry Pi 5: Go-Server und PWA-Client

Die Anwendung läuft als Client/Server-Anwendung auf einem Raspberry Pi 5 im Haushalt. Der Server ist in Go geschrieben, weil er als einzelnes, statisch gelinktes Binary für ARM64 gebaut werden kann und auf dem Pi wenig Ressourcen braucht. Der Client ist eine Progressive Web App statt einer nativen App: Items werden meist mit dem Handy fotografiert und Boxen am Handy gepackt, aber eine native App bräuchte App-Store-Verteilung und zwei Codebasen. Der Go-Server liefert die PWA selbst aus, es gibt also nur einen Prozess zu betreiben.

## Consequences

- Eine PWA braucht HTTPS für Service Worker und Kamerazugriff, auch im Heimnetz. Der Pi braucht daher ein gültiges Zertifikat und einen Hostnamen, nicht nur eine IP-Adresse.
- Erinnerungen werden vom Server zeitgesteuert verschickt. Der Pi muss also dauerhaft laufen, sonst fallen Erinnerungen aus oder kommen verspätet.
