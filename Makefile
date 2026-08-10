.PHONY: build css tailwind-download start stop dev test vet fmt check install clean tidy

# Build the binary (rebuilds CSS first so the embedded stylesheet is fresh).
build: css
	mkdir -p bin
	go build -o bin/harbor ./cmd/harbor

# Compile Tailwind utilities from web/input.css → internal/web/app.css (embedded).
# Scans Go files so only used classes ship in the embed.
css:
	@if [ ! -f .bin/tailwindcss ]; then echo "Missing .bin/tailwindcss — run: make tailwind-download"; exit 1; fi
	./.bin/tailwindcss --input web/input.css --output web/app.css --content "**/*.go" --minify
	mkdir -p internal/web
	cp web/app.css internal/web/app.css

# Fetch the Tailwind CLI binary (run once per checkout).
tailwind-download:
	go run ./cmd/harbor tailwind download

# Start the web UI in the background (daemon).
start: css
	go run ./cmd/harbor start

# Stop the daemon.
stop:
	go run ./cmd/harbor stop

# Run the server in the foreground (development).
dev: css
	go run ./cmd/harbor start --foreground

# Run all tests.
test:
	go test ./...

# go vet across the module.
vet:
	go vet ./...

# Format Go source.
fmt:
	gofmt -s -w .

# Lint + test in one call.
check: vet test

# Build + install binary to PATH (~/go/bin).
install: build
	cp bin/harbor ~/go/bin/harbor

# Remove build artifacts.
clean:
	rm -rf bin/

# Tidy module deps.
tidy:
	go mod tidy