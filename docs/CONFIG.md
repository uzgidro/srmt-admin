# Configuration Guide

This guide explains how to manage configuration files for SRMT Prime, especially in Docker environments.

## Table of Contents
- [Configuration Strategy](#configuration-strategy)
- [File Structure](#file-structure)
- [Docker Configuration](#docker-configuration)
- [Local Development](#local-development)
- [Environment Variables](#environment-variables)
- [Security Best Practices](#security-best-practices)

---

## Configuration Strategy

### The Problem

Configuration files contain:
- ✅ **Structure**: What settings are available
- ❌ **Secrets**: Database passwords, API keys, JWT secrets

**We want to:**
- ✅ Commit example configs (show structure)
- ❌ Never commit actual configs (contain secrets)
- ✅ Mount configs in Docker (keep secrets out of images)

### The Solution

```
config/
├── *.example.yaml      # Committed to git (templates)
├── local.yaml          # Your local config (git-ignored)
├── dev.yaml            # Dev environment (git-ignored)
├── prod.yaml           # Production (git-ignored)
└── docker.yaml         # Docker deployment (git-ignored)
```

**Example configs** show structure, **actual configs** contain your secrets.

---

## File Structure

### Example Files (Committed)

**`config/docker.example.yaml`** - Template for Docker deployment
```yaml
env: "prod"
storage_path: "postgresql://srmt_user:srmt_password@postgres:5432/srmt"
jwt:
  secret: "CHANGE-THIS-SECRET-IN-PRODUCTION"
```

**`config/local.example.yaml`** - Template for local development
```yaml
env: "local"
storage_path: "postgresql://postgres:postgres@localhost:5432/srmt_dev"
```

### Actual Files (Git-Ignored)

These files contain your **actual secrets** and are **never committed**:

- `config/local.yaml` - Your local development config
- `config/dev.yaml` - Development server config
- `config/prod.yaml` - Production server config
- `config/docker.yaml` - Docker deployment config
- `config/ascue.yaml` - ASCUE integration config (optional)
- `config/reservoir.yaml` - Reservoir integration config (optional)

### .gitignore

```gitignore
# Ignore configs with secrets
/config/*.yaml
/config/*.yml

# Keep example configs (no secrets)
!/config/*.example.yaml
```

---

## Docker Configuration

### Option 1: Mount Config Files (Recommended)

**Best for:** Production deployments, keeping secrets out of images

#### Step 1: Create Your Config

```bash
# Copy example
cp config/docker.example.yaml config/docker.yaml

# Edit with your secrets
nano config/docker.yaml
```

#### Step 2: Update docker-compose.yml

```yaml
services:
  app:
    volumes:
      - ./config:/app/config:ro  # Read-only mount
    environment:
      - CONFIG_PATH=/app/config/docker.yaml
```

#### Step 3: Start Services

```bash
make docker-up
```

**What happens:**
1. Docker builds image **without** config files
2. At runtime, mounts your local `config/` directory
3. App reads config from mounted volume
4. Secrets never baked into image ✅

### Option 2: Environment Variables

**Best for:** Cloud platforms (Kubernetes, ECS, etc.)

```yaml
services:
  app:
    environment:
      - CONFIG_PATH=/app/config/prod.yaml
      - JWT_SECRET=${JWT_SECRET}          # From .env
      - DB_PASSWORD=${DB_PASSWORD}        # From .env
```

Create `.env` file (git-ignored):
```bash
JWT_SECRET=your-secret-here
DB_PASSWORD=your-password-here
```

### Option 3: Secrets Management

**Best for:** Production with Docker Swarm/Kubernetes

```yaml
services:
  app:
    secrets:
      - db_password
      - jwt_secret
    environment:
      - CONFIG_PATH=/app/config/prod.yaml

secrets:
  db_password:
    external: true
  jwt_secret:
    external: true
```

---

## Local Development

### Setup

1. **Copy example config:**
```bash
cp config/local.example.yaml config/local.yaml
```

2. **Edit with your settings:**
```bash
# Use your local database credentials
nano config/local.yaml
```

3. **Set environment variable:**
```bash
# Linux/Mac
export CONFIG_PATH=config/local.yaml

# Windows (PowerShell)
$env:CONFIG_PATH="config\local.yaml"

# Windows (CMD)
set CONFIG_PATH=config\local.yaml
```

4. **Run application:**
```bash
make dev
```

### Multiple Environments

Keep separate configs for different environments:

```bash
# Local development
export CONFIG_PATH=config/local.yaml
go run ./cmd

# Connect to dev server
export CONFIG_PATH=config/dev.yaml
go run ./cmd

# Testing
export CONFIG_PATH=config/test.yaml
go test ./...
```

---

## Environment Variables

### Required Variable

**`CONFIG_PATH`** - Path to configuration file

```bash
# Absolute path
export CONFIG_PATH=/app/config/prod.yaml

# Relative path (from project root)
export CONFIG_PATH=config/local.yaml
```

### Docker Environment Variables

In `docker-compose.yml`:

```yaml
services:
  app:
    environment:
      - CONFIG_PATH=/app/config/prod.yaml
      - TZ=Asia/Almaty  # Optional: timezone
```

### Overriding Config Values

You can override specific config values with environment variables (if implemented):

```bash
# Override HTTP port
export HTTP_PORT=8080

# Override database connection
export DB_URL="postgresql://user:pass@localhost/db"
```

---

## Security Best Practices

### ✅ Do's

1. **Use example configs for structure:**
```bash
cp config/docker.example.yaml config/docker.yaml
```

2. **Keep secrets out of git:**
```gitignore
/config/*.yaml
!/config/*.example.yaml
```

3. **Mount configs in Docker (don't copy):**
```yaml
volumes:
  - ./config:/app/config:ro  # Read-only
```

4. **Use strong secrets in production:**
```yaml
jwt:
  secret: "$(openssl rand -base64 32)"  # 256-bit random
```

5. **Restrict file permissions:**
```bash
chmod 600 config/prod.yaml  # Owner read/write only
```

### ❌ Don'ts

1. **Don't commit configs with secrets:**
```bash
# BAD - contains secrets
git add config/prod.yaml
```

2. **Don't hardcode secrets in Dockerfile:**
```dockerfile
# BAD - secrets in image
ENV JWT_SECRET="my-secret"
```

3. **Don't copy configs into Docker image:**
```dockerfile
# BAD - secrets baked into image
COPY config ./config
```

4. **Don't share configs in public repos:**
```bash
# BAD - exposes secrets
git push origin main  # with prod.yaml committed
```

---

## Configuration Reference

### Complete Config Structure

```yaml
# Environment: local, dev, prod
env: "prod"

# PostgreSQL connection string
storage_path: "postgresql://user:pass@host:port/db?sslmode=disable"

# Path to database migrations
migrations_path: "file://./migrations/postgres"

# Timezone for timestamps
timezone: 'UTC'

# MongoDB configuration
mongo:
  host: 'localhost'
  port: '27017'
  username: 'admin'
  password: 'password'
  auth_source: 'admin'

# MinIO object storage
minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false

# External upload services (optional)
upload:
  archive: 'http://parser:19789/parse-archive'
  stock: 'http://parser:19789/parse-stock'
  modsnow: 'http://parser:19789/parse-modsnow'

# HTTP server settings
http_server:
  address: "0.0.0.0:9010"
  timeout: 30s
  idle_timeout: 60s
  allowed_origins:
    - "https://your-frontend.com"

# JWT authentication
jwt:
  secret: "your-secret-key-here"
  access_timeout: 15m
  refresh_timeout: 168h

# API key for callbacks
callback_api_key: 'your-api-key-here'

# Weather API integration (optional)
weather:
  base_url: "https://api.openweathermap.org/data/2.5"
  api_key: "your-api-key"

# MinIO bucket name
bucket: 'srmt-files'

# Telegram bot (optional)
telegram:
  api_key: 'bot-token'
```

### Docker-Specific Settings

When running in Docker, use service names from docker-compose.yml:

```yaml
# Use docker-compose service names
storage_path: "postgresql://user:pass@postgres:5432/db"  # ← postgres (service name)
mongo:
  host: 'mongodb'  # ← mongodb (service name)
minio:
  endpoint: "minio:9000"  # ← minio (service name)
```

---

## Troubleshooting

### Error: "config file not found"

**Problem:** `CONFIG_PATH` not set or wrong path

**Solution:**
```bash
# Check if file exists
ls -la config/local.yaml

# Set correct path
export CONFIG_PATH=config/local.yaml
```

### Error: "permission denied reading config"

**Problem:** Config file has wrong permissions

**Solution:**
```bash
# Make readable
chmod 644 config/local.yaml

# Or read-only for owner
chmod 400 config/prod.yaml
```

### Docker: Config not found

**Problem:** Config not mounted in container

**Solution:** Check docker-compose.yml:
```yaml
services:
  app:
    volumes:
      - ./config:/app/config:ro  # Ensure this exists
```

### Secrets Leaked in Git

**Problem:** Accidentally committed config with secrets

**Solution:**
```bash
# Remove from git history (destructive!)
git rm --cached config/prod.yaml
git commit -m "Remove config with secrets"

# For sensitive data, consider:
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch config/prod.yaml' \
  --prune-empty --tag-name-filter cat -- --all

# Rotate all secrets immediately!
```

---

## Quick Reference

```bash
# Setup new environment
cp config/docker.example.yaml config/docker.yaml
nano config/docker.yaml  # Add your secrets

# Local development
export CONFIG_PATH=config/local.yaml
make dev

# Docker production
make docker-up  # Uses config/prod.yaml (mounted)

# Docker development
make docker-dev-up  # Uses config/local.yaml (mounted)

# Check config is loaded
docker-compose logs app | grep "timezone configured"
```

---

## Related Documentation

- [Docker Guide](DOCKER.md) - Docker deployment
- [Wire FAQ](WIRE_FAQ.md) - Dependency injection
- [README](../README.md) - Main documentation
