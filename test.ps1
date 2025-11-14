# Test runner script for Windows PowerShell
# Usage: .\test.ps1 [command]
# Example: .\test.ps1 repo

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

function Show-Help {
    Write-Host "Available commands:" -ForegroundColor Cyan
    Write-Host "  test                - Run all tests" -ForegroundColor White
    Write-Host "  test-verbose        - Run tests with verbose output" -ForegroundColor White
    Write-Host "  test-coverage       - Run tests with coverage report" -ForegroundColor White
    Write-Host "  test-coverage-html  - Generate HTML coverage report" -ForegroundColor White
    Write-Host "  repo                - Run only repository tests" -ForegroundColor White
    Write-Host "  event               - Run only event repository tests" -ForegroundColor White
    Write-Host "  contact             - Run only contact repository tests" -ForegroundColor White
    Write-Host "  file                - Run only file repository tests" -ForegroundColor White
    Write-Host "  org                 - Run only organization repository tests" -ForegroundColor White
    Write-Host "  user                - Run only user repository tests" -ForegroundColor White
    Write-Host "  race                - Run tests with race detector" -ForegroundColor White
    Write-Host "  build               - Build the application" -ForegroundColor White
    Write-Host "  build-all           - Build all packages" -ForegroundColor White
    Write-Host "  clean               - Clean build artifacts and test cache" -ForegroundColor White
    Write-Host ""
    Write-Host "Example: .\test.ps1 repo" -ForegroundColor Yellow
}

switch ($Command) {
    "test" {
        Write-Host "Running all tests..." -ForegroundColor Green
        go test ./...
    }
    "test-verbose" {
        Write-Host "Running all tests (verbose)..." -ForegroundColor Green
        go test -v ./...
    }
    "test-coverage" {
        Write-Host "Running tests with coverage..." -ForegroundColor Green
        go test -cover ./...
    }
    "test-coverage-html" {
        Write-Host "Generating coverage report..." -ForegroundColor Green
        go test -coverprofile=coverage.out ./...
        if ($LASTEXITCODE -eq 0) {
            go tool cover -html=coverage.out -o coverage.html
            Write-Host "Coverage report generated: coverage.html" -ForegroundColor Cyan
            # Open in browser
            Start-Process coverage.html
        }
    }
    "repo" {
        Write-Host "Running repository tests..." -ForegroundColor Green
        go test -v ./internal/storage/repo/...
    }
    "event" {
        Write-Host "Running event repository tests..." -ForegroundColor Green
        go test -v -run TestEvent ./internal/storage/repo
    }
    "contact" {
        Write-Host "Running contact repository tests..." -ForegroundColor Green
        go test -v -run TestContact ./internal/storage/repo
    }
    "file" {
        Write-Host "Running file repository tests..." -ForegroundColor Green
        go test -v -run TestFile ./internal/storage/repo
    }
    "org" {
        Write-Host "Running organization repository tests..." -ForegroundColor Green
        go test -v -run TestOrganization ./internal/storage/repo
    }
    "user" {
        Write-Host "Running user repository tests..." -ForegroundColor Green
        go test -v -run TestUser ./internal/storage/repo
    }
    "race" {
        Write-Host "Running tests with race detector..." -ForegroundColor Green
        go test -race ./...
    }
    "build" {
        Write-Host "Building application..." -ForegroundColor Green
        go build -o bin/srmt-admin.exe ./cmd/srmt-admin
    }
    "build-all" {
        Write-Host "Building all packages..." -ForegroundColor Green
        go build ./...
    }
    "clean" {
        Write-Host "Cleaning..." -ForegroundColor Green
        go clean
        go clean -testcache
        if (Test-Path coverage.out) { Remove-Item coverage.out }
        if (Test-Path coverage.html) { Remove-Item coverage.html }
        if (Test-Path bin) { Remove-Item -Recurse -Force bin }
        Write-Host "Clean complete!" -ForegroundColor Cyan
    }
    "help" {
        Show-Help
    }
    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host ""
        Show-Help
    }
}
