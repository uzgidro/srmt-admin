# Configuration Files

This directory contains configuration files for SRMT Prime.

## Quick Start

### For Local Development

```bash
# 1. Copy example
cp local.example.yaml local.yaml

# 2. Edit with your settings
nano local.yaml

# 3. Run app
export CONFIG_PATH=config/local.yaml
go run ./cmd
```

### For Docker

```bash
# 1. Copy example
cp docker.example.yaml docker.yaml

# 2. Edit with your secrets
nano docker.yaml

# 3. Start Docker
make docker-up
```

## Files

### Templates (Committed to Git)
- `*.example.yaml` - Configuration templates with no secrets
  - `local.example.yaml` - Template for local development
  - `docker.example.yaml` - Template for Docker deployment

### Actual Configs (Git-Ignored, Contain Secrets)
- `local.yaml` - Your local development configuration
- `dev.yaml` - Development server configuration
- `prod.yaml` - Production server configuration
- `docker.yaml` - Docker deployment configuration
- `ascue.yaml` - ASCUE integration (optional)
- `reservoir.yaml` - Reservoir integration (optional)

## Security

⚠️ **NEVER commit actual config files to git!**

Actual config files contain:
- Database passwords
- API keys
- JWT secrets
- Service credentials

They are automatically **git-ignored** to prevent accidental commits.

## Configuration Structure

```yaml
env: "local|dev|prod"

# Database
storage_path: "postgresql://user:pass@host:port/db"
migrations_path: "file://./migrations/postgres"

# Services
mongo:
  host: 'localhost'
  port: '27017'
  username: 'admin'
  password: 'password'

minio:
  endpoint: "localhost:9000"
  access_key: "key"
  secret_key: "secret"

# HTTP Server
http_server:
  address: "0.0.0.0:9010"
  timeout: 30s
  allowed_origins:
    - "http://localhost:3000"

# Security
jwt:
  secret: "your-secret-key"
  access_timeout: 15m
  refresh_timeout: 168h

callback_api_key: 'your-api-key'
```

## Docker Notes

When using Docker, use service names from `docker-compose.yml`:

```yaml
# ✅ Use service names
storage_path: "postgresql://user:pass@postgres:5432/db"
mongo:
  host: 'mongodb'
minio:
  endpoint: "minio:9000"

# ❌ Don't use localhost
storage_path: "postgresql://user:pass@localhost:5432/db"
```

## Environment Variable

Set `CONFIG_PATH` to specify which config to use:

```bash
# Local development
export CONFIG_PATH=config/local.yaml

# Production
export CONFIG_PATH=config/prod.yaml

# Docker (set in docker-compose.yml)
CONFIG_PATH=/app/config/docker.yaml
```

## Documentation

For complete configuration guide, see: [../docs/CONFIG.md](../docs/CONFIG.md)
