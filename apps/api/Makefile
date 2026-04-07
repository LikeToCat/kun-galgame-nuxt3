.PHONY: dev build run migrate clean

# Development with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Build production binary
build:
	go build -o build/server ./cmd/server

# Run production binary
run: build
	./build/server

# Run database migration
migrate:
	go run ./cmd/migrate

# Clean build artifacts
clean:
	rm -rf build/ tmp/

# Install development tools
tools:
	go install github.com/air-verse/air@latest
