# Testing Guide

This document explains how to run the repository tests on Windows.

## Prerequisites

1. **Docker Desktop must be running** - Testcontainers uses Docker to spin up PostgreSQL
2. Go 1.21+ installed
3. All dependencies installed: `go mod download`

## Quick Start

### PowerShell (Recommended)
```powershell
# Run all repository tests
.\test.ps1 repo

# Run event tests only
.\test.ps1 event

# Run with coverage report
.\test.ps1 test-coverage-html
```

### Command Prompt (CMD)
```cmd
REM Run all repository tests
test.bat repo

REM Run event tests only
test.bat event

REM Run with coverage report
test.bat test-coverage-html
```

### Direct Go Commands
```powershell
# Run all tests
go test ./...

# Run repository tests
go test -v ./internal/storage/repo/...

# Run specific test
go test -v -run TestEventRepository_AddEvent ./internal/storage/repo
```

## Available Commands

| Command | Description |
|---------|-------------|
| `test` | Run all tests |
| `test-verbose` | Run tests with verbose output |
| `test-coverage` | Run tests with coverage report |
| `test-coverage-html` | Generate HTML coverage report (opens in browser) |
| `repo` | Run only repository tests |
| `event` | Run only event repository tests |
| `contact` | Run only contact repository tests |
| `file` | Run only file repository tests |
| `org` | Run only organization repository tests |
| `user` | Run only user repository tests |
| `race` | Run tests with race detector |
| `build` | Build the application |
| `build-all` | Build all packages |
| `clean` | Clean build artifacts and test cache |
| `help` | Show help message |

## Examples

### Run all repository tests
```powershell
# PowerShell
.\test.ps1 repo

# CMD
test.bat repo

# Direct
go test -v ./internal/storage/repo/...
```

### Run specific test suite
```powershell
# Event tests
.\test.ps1 event

# Contact tests
.\test.ps1 contact

# User tests
.\test.ps1 user
```

### Generate coverage report
```powershell
# PowerShell (auto-opens in browser)
.\test.ps1 test-coverage-html

# CMD (auto-opens in browser)
test.bat test-coverage-html

# Direct
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run a single test
```powershell
# Run only AddEvent test
go test -v -run TestEventRepository_AddEvent ./internal/storage/repo

# Run all Event tests
go test -v -run TestEvent ./internal/storage/repo
```

## Test Structure

```
internal/storage/repo/
├── testing/
│   ├── testdb.go       # Testcontainer setup
│   └── fixtures.go     # Test data fixtures
├── contact_test.go     # Contact repository tests
├── event_test.go       # Event repository tests (27 tests)
├── file_test.go        # File repository tests
├── organization_test.go # Organization repository tests
├── user_test.go        # User repository tests
├── department_test.go  # Department repository tests
├── position_test.go    # Position repository tests
└── role_test.go        # Role repository tests
```

## Troubleshooting

### Docker not running
**Error:** `Cannot connect to the Docker daemon`

**Solution:** Start Docker Desktop and wait for it to fully start

### Tests are slow
**Reason:** Testcontainers needs to:
1. Download postgres:16-alpine image (first run only)
2. Start a container for each test file
3. Run migrations

**Normal speed:** 10-30 seconds per test file

### Port conflicts
**Error:** `port already in use`

**Solution:**
```powershell
# Stop all containers
docker stop $(docker ps -aq)

# Clean up
docker system prune -f
```

### Migrations fail
**Error:** `failed to apply migrations`

**Solution:** Check that migration files exist:
```powershell
ls migrations/postgres/
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v ./internal/storage/repo/...
```

## Writing New Tests

### Basic Test Template
```go
func TestRepository_Method(t *testing.T) {
    testDB := repotest.SetupTestDB(t)
    defer testDB.Cleanup(t)

    repo := testDB.NewRepo()
    ctx := context.Background()

    t.Run("successfully does something", func(t *testing.T) {
        // Arrange
        fixtures := repotest.LoadFixtures(t, repo)

        // Act
        result, err := repo.Method(ctx, ...)

        // Assert
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })

    t.Run("returns error when...", func(t *testing.T) {
        // Test error case
    })
}
```

## Test Coverage Summary

| Repository | Test Cases | Status |
|-----------|-----------|--------|
| Event | 27 | ✅ Comprehensive |
| Contact | 15 | ✅ Complete |
| File | 7 | ✅ Complete |
| File Category | 4 | ✅ Complete |
| Organization | 7 | ✅ Complete |
| User | 7 | ✅ Complete |
| Department | 5 | ✅ Complete |
| Position | 5 | ✅ Complete |
| Role | 5 | ✅ Complete |
| **Total** | **82** | **✅ Ready** |

## Resources

- [Testcontainers Go Documentation](https://golang.testcontainers.org/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
