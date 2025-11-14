.PHONY: test test-verbose test-coverage test-repo test-event test-contact test-coverage-html build clean

# Run all tests
test:
	@echo "Running all tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running all tests (verbose)..."
	go test -v ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

# Run tests with detailed coverage and generate HTML report
test-coverage-html:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run only repository tests
test-repo:
	@echo "Running repository tests..."
	go test -v ./internal/storage/repo/...

# Run only event repository tests
test-event:
	@echo "Running event repository tests..."
	go test -v -run TestEvent ./internal/storage/repo

# Run only contact repository tests
test-contact:
	@echo "Running contact repository tests..."
	go test -v -run TestContact ./internal/storage/repo

# Run only file repository tests
test-file:
	@echo "Running file repository tests..."
	go test -v -run TestFile ./internal/storage/repo

# Run only organization repository tests
test-org:
	@echo "Running organization repository tests..."
	go test -v -run TestOrganization ./internal/storage/repo

# Run only user repository tests
test-user:
	@echo "Running user repository tests..."
	go test -v -run TestUser ./internal/storage/repo

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Build the application
build:
	@echo "Building application..."
	go build -o bin/srmt-admin ./cmd/srmt-admin

# Run build for all packages (check for compilation errors)
build-all:
	@echo "Building all packages..."
	go build ./...

# Clean build artifacts and test cache
clean:
	@echo "Cleaning..."
	go clean
	go clean -testcache
	rm -f coverage.out coverage.html
	rm -rf bin/

# Run migrations (requires DATABASE_URL environment variable)
migrate-up:
	@echo "Running migrations..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "Error: DATABASE_URL not set"; \
		exit 1; \
	fi
	migrate -path migrations/postgres -database "$(DATABASE_URL)" up

# Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "Error: DATABASE_URL not set"; \
		exit 1; \
	fi
	migrate -path migrations/postgres -database "$(DATABASE_URL)" down 1

# Show help
help:
	@echo "Available targets:"
	@echo "  test                - Run all tests"
	@echo "  test-verbose        - Run tests with verbose output"
	@echo "  test-coverage       - Run tests with coverage report"
	@echo "  test-coverage-html  - Generate HTML coverage report"
	@echo "  test-repo           - Run only repository tests"
	@echo "  test-event          - Run only event repository tests"
	@echo "  test-contact        - Run only contact repository tests"
	@echo "  test-file           - Run only file repository tests"
	@echo "  test-org            - Run only organization repository tests"
	@echo "  test-user           - Run only user repository tests"
	@echo "  test-race           - Run tests with race detector"
	@echo "  build               - Build the application"
	@echo "  build-all           - Build all packages"
	@echo "  clean               - Clean build artifacts and test cache"
	@echo "  migrate-up          - Run database migrations (requires DATABASE_URL)"
	@echo "  migrate-down        - Rollback last migration (requires DATABASE_URL)"
	@echo "  help                - Show this help message"
