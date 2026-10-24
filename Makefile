.PHONY: web build build-pi run check clean

DIST := internal/webui/dist

# Builds the client and puts it where the Go binary embeds it (.gitkeep is kept).
web:
	cd web && npm ci && npm run build
	rm -rf $(DIST)/*
	cp -R web/build/. $(DIST)/

# Binary for this machine.
build: web
	CGO_ENABLED=0 go build -o bin/inventar ./cmd/inventar

# Binary for the Raspberry Pi 5 (ADR-0001); thanks to pure-Go SQLite without a C toolchain.
build-pi: web
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/inventar-linux-arm64 ./cmd/inventar

# Starts the server locally via HTTP on localhost:8080 with data in ./data.
run: web
	go run ./cmd/inventar serve

# All checks: go vet, Go tests, type check and build of the client.
check:
	go vet ./...
	go test ./...
	cd web && npm ci && npm run check && npm run build

clean:
	rm -rf bin web/build $(DIST)/*
