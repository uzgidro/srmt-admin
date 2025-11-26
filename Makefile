# Makefile for SRMT Prime

.PHONY: help wire build run clean test dev

# Default target
help:
	@echo "Available targets:"
	@echo "  make wire      - Regenerate Wire dependency injection code"
	@echo "  make build     - Build the application"
	@echo "  make run       - Run the application"
	@echo "  make dev       - Generate Wire code and run application"
	@echo "  make clean     - Remove built binaries"
	@echo "  make test      - Run tests"
	@echo "  make all       - Generate Wire, build, and run"

# Generate Wire dependency injection code
wire:
	@echo "Generating Wire code..."
	cd cmd && go run github.com/google/wire/cmd/wire
	@echo "Wire code generated successfully"

# Build the application
build: wire
	@echo "Building application..."
	go build -o srmt-admin.exe ./cmd
	@echo "Build complete: srmt-admin.exe"

# Run the application (requires CONFIG_PATH environment variable)
run:
	@echo "Running application..."
	go run ./cmd

# Development: generate Wire code and run
dev: wire
	@echo "Starting application in development mode..."
	go run ./cmd

# Build optimized production binary
build-prod: wire
	@echo "Building production binary..."
	go build -ldflags="-s -w" -o srmt-admin ./cmd
	@echo "Production build complete: srmt-admin"

# Clean built binaries
clean:
	@echo "Cleaning built binaries..."
	rm -f srmt-admin.exe srmt-admin
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	golangci-lint run

# Install development dependencies
install-deps:
	@echo "Installing Wire..."
	go install github.com/google/wire/cmd/wire@latest
	@echo "Dependencies installed"

# Verify Wire generation is up to date
verify-wire: wire
	@echo "Verifying Wire generation is up to date..."
	@git diff --exit-code cmd/wire_gen.go || (echo "Error: wire_gen.go is not up to date. Run 'make wire'" && exit 1)
	@echo "Wire generation verified"

# All: generate, build, and run
all: build
	@echo "Running application..."
	./srmt-admin.exe
