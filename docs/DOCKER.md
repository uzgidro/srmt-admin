# Docker Guide for SRMT Prime (with Wire DI)

This guide explains how to run SRMT Prime in Docker with Google Wire dependency injection.

## Table of Contents
- [Quick Start](#quick-start)
- [Wire + Docker: How It Works](#wire--docker-how-it-works)
- [Production Deployment](#production-deployment)
- [Development with Docker](#development-with-docker)
- [Docker Commands Reference](#docker-commands-reference)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### Production Mode

```bash
# Start all services (app, PostgreSQL, MongoDB, MinIO)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Development Mode

```bash
# Start dev environment with live reload
make docker-dev-up

# View dev logs
make docker-dev-logs

# Stop dev environment
make docker-dev-down
```

---

## Wire + Docker: How It Works

### The Challenge

Wire generates code at compile time (`wire_gen.go`), which is **git-ignored**. Docker builds need to generate this file during the build process.

### The Solution

The Dockerfile has **two critical steps** for Wire:

```dockerfile
# Step 1: Copy source code
COPY . .

# Step 2: Generate Wire code BEFORE building
RUN cd cmd && go run github.com/google/wire/cmd/wire

# Step 3: Build the entire cmd package (includes wire_gen.go)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o srmt-admin ./cmd
```

### Key Points

1. **Wire generation happens IN Docker** - `wire_gen.go` is created during the build
2. **Build entire package** - `./cmd` not `./cmd/main.go` (includes `wire_gen.go`)
3. **No wire_gen.go in git** - `.dockerignore` excludes it, Docker regenerates it fresh
4. **Multi-stage build** - Builder stage has Go, final stage is minimal Alpine

### What NOT to Do

❌ **Don't** try to build only `main.go`:
```dockerfile
# BAD - won't include wire_gen.go
RUN go build -o app ./cmd/main.go
```

❌ **Don't** commit `wire_gen.go` to solve Docker issues:
```bash
# BAD - creates merge conflicts, outdated code
git add cmd/wire_gen.go
```

✅ **Do** build the entire package:
```dockerfile
# GOOD - includes all files in cmd/ package
RUN go build -o app ./cmd
```

---

## Production Deployment

### Files Overview

```
.
├── Dockerfile              # Production multi-stage build
├── docker-compose.yml      # Production stack
├── .dockerignore          # Excludes unnecessary files
└── config/                # Mounted at runtime
    └── prod.yaml          # Production config
```

### Step-by-Step Deployment

#### 1. Build Docker Image

```bash
# Option 1: Using Makefile
make docker-build

# Option 2: Manual
docker build -t srmt-admin:latest .
```

**What happens during build:**
1. Base image: `golang:1.24.4-alpine`
2. Downloads Go dependencies
3. **Generates Wire code** (`cmd/wire_gen.go`)
4. Compiles application
5. Creates minimal runtime image with only the binary

#### 2. Configure Environment

Create or update `config/prod.yaml`:

```yaml
env: "prod"
storage_path: "host=postgres user=srmt_user password=srmt_password dbname=srmt sslmode=disable"
migrations_path: "file://migrations"
timezone: "Asia/Almaty"

http_server:
  address: ":8080"
  timeout: 30s
  idle_timeout: 60s
  allowed_origins:
    - "https://your-frontend.com"

jwt:
  secret: "your-secret-key-here"
  access_timeout: 15m
  refresh_timeout: 168h

mongo:
  uri: "mongodb://admin:admin_password@mongodb:27017"
  database: "srmt"

minio:
  endpoint: "minio:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  bucket: "srmt-files"
```

#### 3. Start Services

```bash
# Start all services in background
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f app
```

#### 4. Verify Deployment

```bash
# Check app is running
curl http://localhost:8080/api/v3/analytics

# Check health endpoint
docker-compose ps  # Should show "healthy" status for app
```

#### 5. Monitor & Manage

```bash
# View logs
make docker-logs

# Restart app
make docker-restart

# Stop all services
make docker-down
```

### Production Architecture

```
┌─────────────────────────────────────────┐
│  Docker Compose Stack                   │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────┐                       │
│  │   App       │  Port 8080            │
│  │  (Wire DI)  │                       │
│  └──────┬──────┘                       │
│         │                               │
│    ┌────┼──────┬─────────┐            │
│    │    │      │         │            │
│  ┌─▼─┐ ┌▼──┐ ┌▼───┐  ┌──▼──┐        │
│  │PG │ │Mng│ │MinIO│  │Wire │        │
│  │SQL│ │ DB│ │ S3  │  │ Gen │        │
│  └───┘ └───┘ └─────┘  └─────┘        │
│                                         │
│  Persistent Volumes:                    │
│  • postgres-data                        │
│  • mongodb-data                         │
│  • minio-data                           │
└─────────────────────────────────────────┘
```

---

## Development with Docker

### Development Setup

Development mode provides:
- ✅ **Live reload** - Code changes reflected immediately
- ✅ **Source code mounted** - Edit locally, run in container
- ✅ **Separate database** - Won't affect production data
- ✅ **Wire auto-regeneration** - Regenerates on container restart

### File: `docker-compose.dev.yml`

```yaml
services:
  app:
    volumes:
      - ./:/app           # Mount source code
    command: sh -c "cd cmd && go run github.com/google/wire/cmd/wire && cd .. && go run ./cmd"
```

### Development Workflow

#### 1. Start Dev Environment

```bash
make docker-dev-up
```

#### 2. Make Code Changes

Edit any file locally. When you want to apply changes:

```bash
# Option 1: Restart container (regenerates Wire + restarts app)
docker-compose -f docker-compose.dev.yml restart app

# Option 2: Rebuild if dependencies changed
make docker-dev-rebuild
```

#### 3. View Logs

```bash
# All services
make docker-dev-logs

# App only
docker-compose -f docker-compose.dev.yml logs -f app
```

#### 4. Execute Commands in Container

```bash
# Get shell in app container
make docker-dev-exec

# Run tests in container
docker-compose -f docker-compose.dev.yml exec app go test ./...

# Generate Wire manually
docker-compose -f docker-compose.dev.yml exec app sh -c "cd cmd && go run github.com/google/wire/cmd/wire"
```

### Development vs Production

| Aspect | Development | Production |
|--------|-------------|------------|
| **Source Code** | Mounted volume | Copied into image |
| **Wire Generation** | On container start | During image build |
| **Build** | `go run` (slower) | Compiled binary (fast) |
| **Image Size** | ~1GB (includes Go) | ~30MB (binary only) |
| **Restart Time** | Fast (mounted) | Slow (rebuild image) |
| **Use Case** | Local development | Deployment |

---

## Docker Commands Reference

### Production Commands

```bash
# Build & Deploy
make docker-build        # Build image
make docker-up           # Start all services
make docker-down         # Stop all services
make docker-restart      # Restart app only

# Monitoring
make docker-logs         # View all logs
make docker-logs-app     # View app logs only
docker-compose ps        # Check service status

# Cleanup
make docker-clean        # Remove containers & volumes
make docker-down         # Stop services (keep volumes)
```

### Development Commands

```bash
# Dev Environment
make docker-dev-up       # Start dev mode
make docker-dev-down     # Stop dev mode
make docker-dev-rebuild  # Rebuild & restart
make docker-dev-exec     # Shell into container

# Dev Monitoring
make docker-dev-logs     # View dev logs
docker-compose -f docker-compose.dev.yml ps  # Check dev status
```

### Database Commands

```bash
# PostgreSQL
make docker-db-psql      # Access PostgreSQL CLI
make docker-db-backup    # Backup database

# MongoDB
make docker-db-mongo     # Access MongoDB CLI

# Direct queries
docker-compose exec postgres psql -U srmt_user -d srmt -c "SELECT COUNT(*) FROM users;"
```

### Manual Docker Commands

```bash
# Build image manually
docker build -t srmt-admin:latest .

# Run container manually
docker run -p 8080:8080 \
  -v $(pwd)/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  srmt-admin:latest

# Inspect Wire generation in build
docker build --target builder -t srmt-builder .
docker run --rm srmt-builder ls -la /build/cmd/
```

---

## Troubleshooting

### Error: "undefined: InitializeApp" in Docker

**Problem:** Wire code not generated during build.

**Solution:** Check Dockerfile has Wire generation step:

```dockerfile
RUN cd cmd && go run github.com/google/wire/cmd/wire
```

**Verify:**
```bash
# Build and check if wire_gen.go exists
docker build --target builder -t test .
docker run --rm test ls /build/cmd/wire_gen.go
```

### Error: "wire: generate failed" during Docker build

**Problem:** Provider error in Wire configuration.

**Solution:**
1. Test Wire locally first:
```bash
cd cmd && go run github.com/google/wire/cmd/wire
```

2. Fix errors locally, then rebuild Docker image

3. Check build logs:
```bash
docker build -t srmt-admin:latest . 2>&1 | grep wire
```

### Container Crashes Immediately

**Problem:** Config file missing or invalid.

**Check logs:**
```bash
docker-compose logs app
```

**Common issues:**
- Config file not mounted: Check `volumes:` in docker-compose.yml
- Wrong CONFIG_PATH: Should be `/app/config/prod.yaml`
- Database not ready: Add `depends_on` with health checks

**Fix:**
```yaml
services:
  app:
    volumes:
      - ./config:/app/config:ro  # Ensure this exists
    environment:
      - CONFIG_PATH=/app/config/prod.yaml  # Correct path
    depends_on:
      postgres:
        condition: service_healthy  # Wait for DB
```

### Database Connection Refused

**Problem:** App trying to connect before database is ready.

**Solution:** Use health checks in docker-compose.yml:

```yaml
postgres:
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U srmt_user -d srmt"]
    interval: 10s
    timeout: 5s
    retries: 5

app:
  depends_on:
    postgres:
      condition: service_healthy  # Wait for healthy
```

### Changes Not Reflected (Dev Mode)

**Problem:** Source mounted but app not restarting.

**Solutions:**

1. Restart container:
```bash
docker-compose -f docker-compose.dev.yml restart app
```

2. Regenerate Wire if providers changed:
```bash
docker-compose -f docker-compose.dev.yml exec app sh -c "cd cmd && go run github.com/google/wire/cmd/wire"
docker-compose -f docker-compose.dev.yml restart app
```

3. Check volume mount:
```bash
docker-compose -f docker-compose.dev.yml exec app ls /app
# Should show your source files
```

### Image Too Large

**Problem:** Docker image > 500MB.

**Check:** Are you using multi-stage build?

```dockerfile
# Stage 1: Builder (large, ~1GB)
FROM golang:1.24.4-alpine AS builder
...

# Stage 2: Runtime (small, ~30MB)
FROM alpine:latest
COPY --from=builder /build/srmt-admin .
```

**Verify image size:**
```bash
docker images srmt-admin
# Should be ~30-40MB
```

### Port Already in Use

**Problem:** `0.0.0.0:8080: bind: address already in use`

**Solutions:**

1. Stop other services:
```bash
docker-compose down
```

2. Change port in docker-compose.yml:
```yaml
ports:
  - "8081:8080"  # Use 8081 externally
```

3. Find and stop conflicting process:
```bash
# Windows
netstat -ano | findstr :8080
taskkill /PID <pid> /F

# Linux
lsof -i :8080
kill -9 <pid>
```

---

## Best Practices

### 1. Use Multi-Stage Builds

✅ **Do:**
```dockerfile
FROM golang:1.24.4-alpine AS builder
...
FROM alpine:latest  # Minimal runtime
```

❌ **Don't:**
```dockerfile
FROM golang:1.24.4-alpine  # Large final image
```

### 2. Generate Wire in Docker

✅ **Do:**
```dockerfile
COPY . .
RUN cd cmd && go run github.com/google/wire/cmd/wire
RUN go build ./cmd
```

❌ **Don't:**
```bash
# Don't pre-generate and commit wire_gen.go
git add cmd/wire_gen.go
```

### 3. Use .dockerignore

✅ **Do:**
```.dockerignore
*.exe
*.md
.git
cmd/wire_gen.go  # Let Docker regenerate
```

### 4. Health Checks

✅ **Do:**
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --spider http://localhost:8080/api/v3/analytics
```

### 5. Non-Root User

✅ **Do:**
```dockerfile
RUN adduser -D appuser
USER appuser
```

### 6. Volume Mounts for Config

✅ **Do:**
```yaml
volumes:
  - ./config:/app/config:ro  # Read-only
```

❌ **Don't:**
```dockerfile
COPY config ./config  # Config in image
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main]

jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: docker build -t srmt-admin:${{ github.sha }} .

      - name: Test Wire generation
        run: |
          docker run --rm srmt-admin:${{ github.sha }} \
            sh -c "test -f /app/srmt-admin || exit 1"

      - name: Push to registry
        run: |
          docker tag srmt-admin:${{ github.sha }} registry.example.com/srmt-admin:latest
          docker push registry.example.com/srmt-admin:latest
```

---

## Quick Reference

```bash
# Production
make docker-up          # Start
make docker-logs        # Monitor
make docker-down        # Stop

# Development
make docker-dev-up      # Start dev
make docker-dev-logs    # Monitor dev
make docker-dev-exec    # Shell into container

# Database
make docker-db-psql     # PostgreSQL CLI
make docker-db-mongo    # MongoDB CLI

# Cleanup
make docker-clean       # Full cleanup
```

For more information on Wire itself, see [docs/WIRE_FAQ.md](WIRE_FAQ.md).
