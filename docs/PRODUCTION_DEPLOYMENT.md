# Production Deployment Guide (Docker Hub → Server)

This guide covers deploying SRMT Prime from Docker Hub to production servers.

## Table of Contents
- [Deployment Workflow](#deployment-workflow)
- [Strategy 1: Volume Mounts (Recommended)](#strategy-1-volume-mounts-recommended)
- [Strategy 2: Docker Secrets](#strategy-2-docker-secrets)
- [Strategy 3: Environment Variables](#strategy-3-environment-variables)
- [Strategy 4: Config Server](#strategy-4-config-server)
- [CI/CD Integration](#cicd-integration)
- [Security Best Practices](#security-best-practices)

---

## Deployment Workflow

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Local / CI    │      │   Docker Hub    │      │  Prod Server    │
├─────────────────┤      ├─────────────────┤      ├─────────────────┤
│                 │      │                 │      │                 │
│ 1. Build Image  │─────▶│ 2. Push Image   │─────▶│ 3. Pull Image   │
│                 │      │                 │      │                 │
│ ✅ Code         │      │ ✅ Binary       │      │ ✅ Binary       │
│ ✅ Wire Gen     │      │ ❌ No Configs   │      │ ⚠️  Need Configs│
│ ❌ No Configs   │      │ ❌ No Secrets   │      │                 │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

**Key Principle:**
- ✅ Image contains: Application binary, migrations, dependencies
- ❌ Image does NOT contain: Config files, secrets, credentials

**Why?**
- Same image can be used in dev/staging/prod with different configs
- Secrets never exposed in public Docker Hub
- Configs can be updated without rebuilding image

---

## Strategy 1: Volume Mounts (Recommended)

**Best for:** VPS, dedicated servers, simple deployments

### How It Works

```
Server Filesystem          Docker Container
┌──────────────────┐      ┌──────────────────┐
│ /opt/srmt/       │      │ /app/            │
│ ├── config/      │─────▶│ ├── config/      │ (mounted)
│ │   └── prod.yaml│      │ │   └── prod.yaml│
│ └── data/        │      │ └── srmt-admin   │
└──────────────────┘      └──────────────────┘
```

### Step-by-Step

#### 1. On Production Server: Create Directory Structure

```bash
# Create application directory
sudo mkdir -p /opt/srmt/config
sudo mkdir -p /opt/srmt/data

# Set permissions
sudo chown -R $USER:$USER /opt/srmt
```

#### 2. Copy Config File to Server

**Option A: Manual Copy**
```bash
# From your local machine
scp config/prod.yaml user@server:/opt/srmt/config/prod.yaml
```

**Option B: Create on Server**
```bash
# SSH to server
ssh user@server

# Create config
cat > /opt/srmt/config/prod.yaml << 'EOF'
env: "prod"
storage_path: "postgresql://srmt_user:PASSWORD@localhost:5432/srmt"
migrations_path: "file://./migrations/postgres"
timezone: 'UTC'

mongo:
  host: 'localhost'
  port: '27017'
  username: 'admin'
  password: 'PASSWORD'
  auth_source: 'admin'

minio:
  endpoint: "s3.example.com:9000"
  access_key: "ACCESS_KEY"
  secret_key: "SECRET_KEY"
  use_ssl: true

http_server:
  address: "0.0.0.0:9010"
  timeout: 30s
  idle_timeout: 60s
  allowed_origins:
    - "https://app.example.com"

jwt:
  secret: "PRODUCTION_SECRET_HERE"
  access_timeout: 15m
  refresh_timeout: 168h

callback_api_key: 'API_KEY_HERE'

bucket: 'srmt-prod'
EOF

# Secure the file
chmod 600 /opt/srmt/config/prod.yaml
```

#### 3. Pull Image from Docker Hub

```bash
# Login to Docker Hub (if private repo)
docker login

# Pull latest image
docker pull yourusername/srmt-admin:latest

# Or specific version
docker pull yourusername/srmt-admin:v1.2.3
```

#### 4. Run Container with Volume Mount

**Option A: Using docker run**
```bash
docker run -d \
  --name srmt-admin \
  --restart unless-stopped \
  -p 9010:9010 \
  -v /opt/srmt/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  yourusername/srmt-admin:latest
```

**Option B: Using docker-compose (Recommended)**

Create `/opt/srmt/docker-compose.yml`:
```yaml
version: '3.8'

services:
  app:
    image: yourusername/srmt-admin:latest
    container_name: srmt-admin
    restart: unless-stopped
    ports:
      - "9010:9010"
    volumes:
      - /opt/srmt/config:/app/config:ro
    environment:
      - CONFIG_PATH=/app/config/prod.yaml
    depends_on:
      - postgres
      - mongodb

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: srmt
      POSTGRES_USER: srmt_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"

  mongodb:
    image: mongo:7
    restart: unless-stopped
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
    volumes:
      - mongodb-data:/data/db
    ports:
      - "127.0.0.1:27017:27017"

volumes:
  postgres-data:
  mongodb-data:
```

Create `/opt/srmt/.env`:
```bash
DB_PASSWORD=your_db_password_here
MONGO_PASSWORD=your_mongo_password_here
```

Start services:
```bash
cd /opt/srmt
docker-compose up -d
```

#### 5. Verify Deployment

```bash
# Check container is running
docker ps

# Check logs
docker logs srmt-admin

# Test endpoint
curl http://localhost:9010/api/v3/analytics

# Check health
docker inspect srmt-admin | grep Health -A 5
```

#### 6. Update Deployment

```bash
# Pull new image
docker pull yourusername/srmt-admin:latest

# Recreate container
docker-compose up -d

# Or with docker run
docker stop srmt-admin
docker rm srmt-admin
docker run -d ... # (same command as step 4)
```

### Advantages
- ✅ Config updates without rebuilding image
- ✅ Same image for dev/staging/prod
- ✅ Easy to manage
- ✅ Secrets stay on server

### Disadvantages
- ⚠️ Need SSH/file access to update config
- ⚠️ Manual config management

---

## Strategy 2: Docker Secrets

**Best for:** Docker Swarm, production clusters

### How It Works

```
Docker Secrets Store          Container
┌──────────────────┐         ┌──────────────────┐
│ db_password      │────────▶│ /run/secrets/    │
│ jwt_secret       │         │ db_password      │
│ api_key          │         │ jwt_secret       │
└──────────────────┘         └──────────────────┘
```

### Step-by-Step

#### 1. Create Secrets

```bash
# Create secrets
echo "your_db_password" | docker secret create db_password -
echo "your_jwt_secret" | docker secret create jwt_secret -
echo "your_api_key" | docker secret create api_key -

# Or from files
docker secret create prod_config /opt/srmt/config/prod.yaml
```

#### 2. Deploy with Secrets

**docker-compose.yml for Swarm:**
```yaml
version: '3.8'

services:
  app:
    image: yourusername/srmt-admin:latest
    secrets:
      - prod_config
      - db_password
      - jwt_secret
    environment:
      - CONFIG_PATH=/run/secrets/prod_config
    deploy:
      replicas: 2
      restart_policy:
        condition: on-failure
    ports:
      - "9010:9010"

secrets:
  prod_config:
    external: true
  db_password:
    external: true
  jwt_secret:
    external: true
```

#### 3. Deploy Stack

```bash
docker stack deploy -c docker-compose.yml srmt
```

### Advantages
- ✅ Encrypted at rest
- ✅ Automatic distribution to nodes
- ✅ Integrated with Docker

### Disadvantages
- ⚠️ Requires Docker Swarm
- ⚠️ More complex setup

---

## Strategy 3: Environment Variables

**Best for:** Cloud platforms, Kubernetes, simple configs

### How It Works

Modify your app to read from environment variables OR use a config template with env var substitution.

### Option A: Environment Variables Only

**On Server:**
```bash
docker run -d \
  --name srmt-admin \
  -p 9010:9010 \
  -e DB_HOST=postgres \
  -e DB_USER=srmt_user \
  -e DB_PASSWORD=secret \
  -e MONGO_HOST=mongodb \
  -e JWT_SECRET=your_secret \
  yourusername/srmt-admin:latest
```

**docker-compose.yml:**
```yaml
services:
  app:
    image: yourusername/srmt-admin:latest
    environment:
      - DB_HOST=postgres
      - DB_USER=srmt_user
      - DB_PASSWORD=${DB_PASSWORD}
      - MONGO_HOST=mongodb
      - JWT_SECRET=${JWT_SECRET}
    env_file:
      - .env
```

**.env file:**
```bash
DB_PASSWORD=your_password
JWT_SECRET=your_secret
```

### Option B: Config Template with envsubst

**1. Create config template on server:**

`/opt/srmt/config/prod.yaml.template`:
```yaml
env: "prod"
storage_path: "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/srmt"
jwt:
  secret: "${JWT_SECRET}"
```

**2. Generate config at runtime:**

```bash
# Generate actual config
envsubst < /opt/srmt/config/prod.yaml.template > /opt/srmt/config/prod.yaml

# Or in entrypoint script
docker run -d \
  -v /opt/srmt/config:/app/config \
  -e DB_USER=srmt_user \
  -e DB_PASSWORD=secret \
  --entrypoint sh \
  yourusername/srmt-admin:latest \
  -c "envsubst < /app/config/prod.yaml.template > /app/config/prod.yaml && ./srmt-admin"
```

### Advantages
- ✅ Works everywhere (Kubernetes, ECS, etc.)
- ✅ Easy to manage in cloud platforms
- ✅ No file management

### Disadvantages
- ⚠️ Requires app changes (if not already supported)
- ⚠️ Many env vars can be messy
- ⚠️ Less readable than YAML

---

## Strategy 4: Config Server

**Best for:** Large deployments, multiple services, centralized config

### How It Works

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   App        │─────▶│ Config Server│◀─────│   Consul     │
│  Container   │fetch │  (Vault,     │      │   etcd       │
└──────────────┘      │   Consul)    │      │   Spring     │
                      └──────────────┘      └──────────────┘
```

### Example: Using Consul

**1. Store config in Consul:**
```bash
consul kv put srmt/prod/db_password "secret"
consul kv put srmt/prod/jwt_secret "secret"
```

**2. App fetches config at startup:**
```go
// Pseudocode - would need implementation
config := fetchFromConsul("srmt/prod")
```

**3. Run container:**
```bash
docker run -d \
  -e CONSUL_ADDR=consul:8500 \
  -e CONFIG_PREFIX=srmt/prod \
  yourusername/srmt-admin:latest
```

### Advantages
- ✅ Centralized config management
- ✅ Dynamic updates
- ✅ Audit trail
- ✅ Multiple environments

### Disadvantages
- ⚠️ Requires config server infrastructure
- ⚠️ App needs integration code
- ⚠️ More complexity

---

## CI/CD Integration

### GitHub Actions Example

**.github/workflows/deploy.yml:**
```yaml
name: Build and Deploy

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}

      - name: Extract version
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            yourusername/srmt-admin:latest
            yourusername/srmt-admin:${{ steps.version.outputs.VERSION }}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/')
    steps:
      - name: Deploy to production
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.PROD_HOST }}
          username: ${{ secrets.PROD_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            cd /opt/srmt
            docker pull yourusername/srmt-admin:${{ steps.version.outputs.VERSION }}
            docker-compose up -d
```

### GitLab CI Example

**.gitlab-ci.yml:**
```yaml
stages:
  - build
  - deploy

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD
    - docker build -t yourusername/srmt-admin:$CI_COMMIT_TAG .
    - docker push yourusername/srmt-admin:$CI_COMMIT_TAG
  only:
    - tags

deploy:
  stage: deploy
  script:
    - ssh user@server "cd /opt/srmt && docker pull yourusername/srmt-admin:$CI_COMMIT_TAG && docker-compose up -d"
  only:
    - tags
```

---

## Security Best Practices

### 1. Image Security

```bash
# ✅ Use specific tags, not :latest
docker pull yourusername/srmt-admin:v1.2.3

# ✅ Verify image digest
docker pull yourusername/srmt-admin@sha256:abc123...

# ✅ Scan for vulnerabilities
docker scan yourusername/srmt-admin:v1.2.3
```

### 2. Config File Security

```bash
# ✅ Restrict permissions
chmod 600 /opt/srmt/config/prod.yaml
chown root:root /opt/srmt/config/prod.yaml

# ✅ Mount as read-only
-v /opt/srmt/config:/app/config:ro

# ✅ Use encrypted filesystem
# Store configs on encrypted volume
```

### 3. Secrets Management

```bash
# ❌ Don't store in git
git add config/prod.yaml  # BAD!

# ❌ Don't expose in logs
echo $JWT_SECRET  # BAD!

# ✅ Use secrets management
docker secret create jwt_secret -

# ✅ Rotate secrets regularly
# Update secrets every 90 days
```

### 4. Network Security

```yaml
# ✅ Don't expose databases publicly
postgres:
  ports:
    - "127.0.0.1:5432:5432"  # Only localhost

# ✅ Use internal networks
networks:
  internal:
    internal: true
```

### 5. Audit & Monitoring

```bash
# ✅ Enable audit logs
docker logs srmt-admin | grep "config loaded"

# ✅ Monitor unauthorized access
fail2ban, cloudflare, etc.

# ✅ Alert on config changes
inotifywait -m /opt/srmt/config
```

---

## Complete Production Deployment Script

**deploy.sh:**
```bash
#!/bin/bash
set -e

# Configuration
IMAGE="yourusername/srmt-admin"
VERSION="${1:-latest}"
DEPLOY_DIR="/opt/srmt"
CONFIG_FILE="$DEPLOY_DIR/config/prod.yaml"

echo "🚀 Deploying SRMT Admin $VERSION"

# 1. Pre-flight checks
echo "📋 Pre-flight checks..."
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Config file not found: $CONFIG_FILE"
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running"
    exit 1
fi

# 2. Backup current config (optional)
echo "💾 Backing up config..."
cp "$CONFIG_FILE" "$CONFIG_FILE.backup.$(date +%Y%m%d_%H%M%S)"

# 3. Pull new image
echo "⬇️  Pulling image $IMAGE:$VERSION..."
docker pull "$IMAGE:$VERSION"

# 4. Stop old container
echo "🛑 Stopping old container..."
docker stop srmt-admin 2>/dev/null || true
docker rm srmt-admin 2>/dev/null || true

# 5. Run new container
echo "▶️  Starting new container..."
docker run -d \
  --name srmt-admin \
  --restart unless-stopped \
  -p 9010:9010 \
  -v "$DEPLOY_DIR/config:/app/config:ro" \
  -e CONFIG_PATH=/app/config/prod.yaml \
  "$IMAGE:$VERSION"

# 6. Wait for health check
echo "🏥 Waiting for health check..."
sleep 5
for i in {1..30}; do
    if curl -f http://localhost:9010/api/v3/analytics > /dev/null 2>&1; then
        echo "✅ Deployment successful!"
        docker logs --tail 20 srmt-admin
        exit 0
    fi
    sleep 2
done

echo "❌ Health check failed!"
docker logs --tail 50 srmt-admin
exit 1
```

**Usage:**
```bash
# Deploy latest
./deploy.sh

# Deploy specific version
./deploy.sh v1.2.3

# Make executable
chmod +x deploy.sh
```

---

## Troubleshooting

### Container Starts But Crashes

```bash
# Check logs
docker logs srmt-admin

# Common issues:
# - Config file not found → Check volume mount
# - Permission denied → Check file permissions (chmod 644)
# - Database connection failed → Check network, credentials
```

### Config Changes Not Applied

```bash
# Restart container to reload config
docker restart srmt-admin

# Or recreate container
docker stop srmt-admin && docker rm srmt-admin
docker run ... # same command
```

### Image Pull Failed

```bash
# Login to Docker Hub
docker login

# Check image exists
docker pull yourusername/srmt-admin:latest

# Use digest if tag is mutable
docker pull yourusername/srmt-admin@sha256:abc123...
```

---

## Quick Reference

### Push to Docker Hub
```bash
docker build -t yourusername/srmt-admin:v1.0.0 .
docker push yourusername/srmt-admin:v1.0.0
```

### Deploy on Server (Volume Mount)
```bash
# Setup
mkdir -p /opt/srmt/config
scp config/prod.yaml server:/opt/srmt/config/

# Deploy
docker pull yourusername/srmt-admin:v1.0.0
docker run -d \
  -p 9010:9010 \
  -v /opt/srmt/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  yourusername/srmt-admin:v1.0.0
```

### Update Deployment
```bash
docker pull yourusername/srmt-admin:latest
docker stop srmt-admin
docker rm srmt-admin
docker run ... # same command
```

---

## Recommendation

**For most production scenarios, use Strategy 1 (Volume Mounts):**

✅ Simple and reliable
✅ Config updates without rebuilding
✅ Same image for all environments
✅ Works on any server (VPS, dedicated, cloud)

Combine with proper secrets management (encrypted filesystem, restricted permissions) for security.
