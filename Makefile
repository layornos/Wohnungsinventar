.PHONY: web build build-pi run check clean

DIST := internal/webui/dist

# Baut den Client und legt ihn zum Einbetten ins Go-Binary ab (.gitkeep bleibt erhalten).
web:
	cd web && npm ci && npm run build
	rm -rf $(DIST)/*
	cp -R web/build/. $(DIST)/

# Binary für diesen Rechner.
build: web
	CGO_ENABLED=0 go build -o bin/inventar ./cmd/inventar

# Binary für den Raspberry Pi 5 (ADR-0001); dank reinem Go-SQLite ohne C-Toolchain.
build-pi: web
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/inventar-linux-arm64 ./cmd/inventar

# Startet den Server lokal per HTTP auf localhost:8080 mit Daten in ./data.
run: web
	go run ./cmd/inventar starten

# Alle Prüfungen: Go-Vet, Go-Tests, Typprüfung und Build des Clients.
check:
	go vet ./...
	go test ./...
	cd web && npm ci && npm run check && npm run build

clean:
	rm -rf bin web/build $(DIST)/*
