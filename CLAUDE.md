# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Regenerate Wire DI (required after changing providers or adding dependencies)
make wire
# or: cd cmd && go run github.com/google/wire/cmd/wire && cd ..

# Build
make build              # generates Wire + builds srmt-admin.exe

# Run (requires CONFIG_PATH env var)
export CONFIG_PATH=config/local.yaml
make dev                # generates Wire + runs app
go run ./cmd            # run directly (Wire must already be generated)

# Tests
make test               # go test -v ./...
go test -v ./internal/lib/service/alarm/...   # single package
go test -v -run TestDetector ./internal/lib/service/alarm/...  # single test

# Formatting & linting
make fmt                # go fmt ./...
make lint               # golangci-lint run

# Production build
make build-prod         # optimized binary with stripped symbols
```

## Architecture

**Module:** `srmt-admin` | **Go version:** 1.25.7 | **Router:** chi/v5

### Layered Pattern

```
HTTP Handlers  →  Services (business logic)  →  Repositories (data access)
```

- **Newer modules (HRM)** use the full Handler → Service → Repo pattern with a service layer.
- **Older modules** use Handler → Repo directly (no service layer).

### Dependency Injection (Wire)

Compile-time DI via Google Wire. Four provider sets in `internal/providers/`:

| File | ProviderSet | Provides |
|------|-------------|----------|
| `config.go` | `ConfigProviderSet` | Config, Logger, JWT, Location, Redis config, etc. |
| `storage.go` | `StorageProviderSet` | PostgreSQL, MongoDB, MinIO, Redis drivers/repos |
| `services.go` | `ServiceProviderSet` | Token, ASCUE, Metrics, Reservoir, Alarm, all HRM services |
| `http.go` | `HTTPProviderSet` | Router, HTTP Server, AppContainer |

**Wire workflow:** `cmd/wire.go` (injector definition, `//go:build wireinject`) → `go generate` or `make wire` → `cmd/wire_gen.go` (generated, git-ignored).

All deps flow through `providers.AppContainer` → `router.AppDependencies` → individual handlers via `router.SetupRoutes()`.

### Handler Pattern

Handlers define **local interfaces** for their dependencies and accept them via a `New()` constructor that returns `http.HandlerFunc`:

```go
type Getter interface {
    GetByID(ctx context.Context, id int64) (*model.Thing, error)
}

func New(log *slog.Logger, svc Getter) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) { ... }
}
```

### Key Conventions

- **Response helpers** (`internal/lib/api/response/`): `resp.OK()`, `resp.Created()`, `resp.BadRequest(msg)`, `resp.NotFound(msg)`, `resp.Conflict(msg)`, `resp.Forbidden(msg)`, `resp.Unauthorized(msg)`, `resp.InternalServerError(msg)`, `resp.Delete()`, `resp.ValidationErrors(errs)`
- **JSON rendering:** `github.com/go-chi/render` — `render.JSON`, `render.DecodeJSON`, `render.Status`
- **Validation:** `github.com/go-playground/validator/v10`
- **Auth claims:** `mwauth.ClaimsFromContext(ctx)` returns `(*token.Claims, bool)` with `.UserID`, `.ContactID`, `.Roles`
- **Role middleware:** `mwauth.RequireAnyRole("role1", "role2")`
- **Sentinel errors:** `internal/storage/storage.go` — `ErrUserNotFound`, `ErrDuplicate`, `ErrDataNotFound`, `ErrUniqueViolation`, etc.
- **Migrations:** `migrations/postgres/` — auto-run on startup via `golang-migrate`. Format: `000NNN_name.{up,down}.sql`

### Key Paths

| What | Where |
|------|-------|
| Entry point | `cmd/main.go` |
| Wire injector | `cmd/wire.go` |
| Provider sets | `internal/providers/*.go` |
| Router & routes | `internal/http-server/router/router.go` |
| Handlers | `internal/http-server/handlers/{module}/{action}.go` |
| Services | `internal/lib/service/{module}/service.go` |
| Models | `internal/lib/model/{module}/model.go` |
| DTOs | `internal/lib/dto/{name}.go` |
| PG repos | `internal/storage/repo/{module}.go` (methods on `*Repo`) |
| Mongo repos | `internal/storage/mongo/` |
| Middleware | `internal/http-server/middleware/{name}/` |
| Config | `internal/config/config.go` (cleanenv, YAML + env overrides) |
| Migrations | `migrations/postgres/` |

### Adding a New Feature (checklist)

1. Add migration in `migrations/postgres/` (next sequence number)
2. Add model in `internal/lib/model/{module}/`
3. Add repo methods on `*Repo` in `internal/storage/repo/`
4. (If using service layer) Add service in `internal/lib/service/{module}/`
5. Add handler(s) in `internal/http-server/handlers/{module}/{action}.go`
6. Register routes in `internal/http-server/router/router.go`
7. If new service/repo needed in DI: add provider in `internal/providers/`, update `AppContainer`/`AppDependencies`, run `make wire`

### Config

YAML config loaded via `cleanenv`. Set `CONFIG_PATH` env var. Example configs in `config/*.example.yaml`. Actual configs (`local.yaml`, `dev.yaml`, `prod.yaml`) are git-ignored.

### Databases

- **PostgreSQL** — primary relational store (pgx/v5 driver)
- **MongoDB** — supplementary document store
- **MinIO** — S3-compatible object storage
- **Redis** — caching

### Documentation

- `docs/WIRE_FAQ.md` — Wire DI troubleshooting
- `docs/CONFIG.md` — configuration guide
- `docs/DOCKER.md` — Docker deployment
- `docs/HRM_BACKEND_API.md` — HRM API spec
- `docs/PRODUCTION_DEPLOYMENT.md` — production deployment
