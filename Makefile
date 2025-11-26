# Makefile for SRMT Prime

.PHONY: help wire build run clean test dev docker-build docker-up docker-down docker-dev-up docker-dev-down docker-logs docker-clean

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Local Development:"
	@echo "  make wire           - Regenerate Wire dependency injection code"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application"
	@echo "  make dev            - Generate Wire code and run application"
	@echo "  make clean          - Remove built binaries"
	@echo "  make test           - Run tests"
	@echo "  make all            - Generate Wire, build, and run"
	@echo ""
	@echo "Docker (Production):"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-up      - Start all services with docker-compose"
	@echo "  make docker-down    - Stop all services"
	@echo "  make docker-logs    - Show logs from all services"
	@echo "  make docker-clean   - Remove all containers and volumes"
	@echo ""
	@echo "Docker (Development):"
	@echo "  make docker-dev-up  - Start dev environment with live reload"
	@echo "  make docker-dev-down - Stop dev environment"
	@echo "  make docker-dev-logs - Show dev logs"

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

# ===== Docker Commands =====

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t srmt-admin:latest .
	@echo "Docker image built successfully"

# Start all services (production)
docker-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d
	@echo "Services started. Access app at http://localhost:9010"

# Stop all services
docker-down:
	@echo "Stopping services..."
	docker-compose down
	@echo "Services stopped"

# Show logs from all services
docker-logs:
	docker-compose logs -f

# Show logs from app only
docker-logs-app:
	docker-compose logs -f app

# Restart app service
docker-restart:
	@echo "Restarting app service..."
	docker-compose restart app

# Clean up Docker (remove containers, networks, volumes)
docker-clean:
	@echo "Cleaning up Docker resources..."
	docker-compose down -v
	docker system prune -f
	@echo "Docker cleanup complete"

# ===== Docker Development Commands =====

# Start development environment
docker-dev-up:
	@echo "Starting development environment..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Dev environment started. Access app at http://localhost:9010"

# Stop development environment
docker-dev-down:
	@echo "Stopping development environment..."
	docker-compose -f docker-compose.dev.yml down
	@echo "Dev environment stopped"

# Show development logs
docker-dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

# Rebuild and restart dev environment
docker-dev-rebuild:
	@echo "Rebuilding dev environment..."
	docker-compose -f docker-compose.dev.yml up -d --build
	@echo "Dev environment rebuilt"

# Execute command in dev container
docker-dev-exec:
	docker-compose -f docker-compose.dev.yml exec app sh

# ===== Database Commands =====

# Access PostgreSQL CLI
docker-db-psql:
	docker-compose exec postgres psql -U srmt_user -d srmt

# Access MongoDB CLI
docker-db-mongo:
	docker-compose exec mongodb mongosh -u admin -p admin_password

# Backup PostgreSQL database
docker-db-backup:
	@echo "Backing up PostgreSQL database..."
	docker-compose exec -T postgres pg_dump -U srmt_user srmt > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup complete"
