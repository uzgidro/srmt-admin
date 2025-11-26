# Production Deployment Quick Start

Choose your deployment strategy:

## Strategy 1: Docker Hub → Server (Volume Mount) ⭐ Recommended

**When to use:** VPS, dedicated servers, simple deployments

### Build & Push
```bash
# Build image (Wire code generated automatically)
docker build -t yourusername/srmt-admin:v1.0.0 .

# Push to Docker Hub
docker push yourusername/srmt-admin:v1.0.0
docker tag yourusername/srmt-admin:v1.0.0 yourusername/srmt-admin:latest
docker push yourusername/srmt-admin:latest
```

### On Production Server
```bash
# 1. Setup server (one-time)
sudo mkdir -p /opt/srmt/config
cd /opt/srmt

# 2. Create config file
cat > /opt/srmt/config/prod.yaml << 'EOF'
env: "prod"
storage_path: "postgresql://user:password@postgres:5432/srmt"
# ... add your settings
EOF

chmod 600 /opt/srmt/config/prod.yaml

# 3. Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  app:
    image: yourusername/srmt-admin:latest
    restart: unless-stopped
    ports:
      - "9010:9010"
    volumes:
      - ./config:/app/config:ro
    environment:
      - CONFIG_PATH=/app/config/prod.yaml
    depends_on:
      - postgres
      - mongodb

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: srmt
      POSTGRES_USER: srmt_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data

  mongodb:
    image: mongo:7
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
    volumes:
      - mongodb-data:/data/db

volumes:
  postgres-data:
  mongodb-data:
EOF

# 4. Create .env with passwords
cat > .env << 'EOF'
DB_PASSWORD=your_secure_password
MONGO_PASSWORD=your_secure_password
EOF

chmod 600 .env

# 5. Deploy
docker-compose up -d

# 6. Check status
docker-compose logs -f app
```

### Update Deployment
```bash
cd /opt/srmt
docker-compose pull
docker-compose up -d
```

---

## Strategy 2: Docker Hub → Server (Simple Run)

**When to use:** Minimal setup, external databases

```bash
# On server
docker pull yourusername/srmt-admin:latest

# Run with config mounted
docker run -d \
  --name srmt-admin \
  --restart unless-stopped \
  -p 9010:9010 \
  -v /opt/srmt/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  yourusername/srmt-admin:latest

# Update
docker stop srmt-admin
docker rm srmt-admin
docker pull yourusername/srmt-admin:latest
docker run ... # same command
```

---

## Strategy 3: Environment Variables

**When to use:** Cloud platforms (Kubernetes, ECS, etc.)

```bash
docker run -d \
  --name srmt-admin \
  -p 9010:9010 \
  -e DB_HOST=postgres \
  -e DB_USER=srmt_user \
  -e DB_PASSWORD=secret \
  -e JWT_SECRET=your_secret \
  yourusername/srmt-admin:latest
```

**Note:** Requires app modification to read from environment variables.

---

## Strategy 4: Automated Scripts

**When to use:** Simplified deployments, automation

### Setup (one-time)
```bash
# Run setup script
curl -fsSL https://raw.githubusercontent.com/yourusername/srmt-prime/main/scripts/setup-server.sh | bash

# Edit config
nano /opt/srmt/config/prod.yaml
```

### Deploy
```bash
# Using deployment script
curl -fsSL https://raw.githubusercontent.com/yourusername/srmt-prime/main/scripts/deploy-production.sh | bash -s v1.0.0

# Or manual
cd /opt/srmt
./deploy-production.sh v1.0.0
```

---

## Key Points

### ✅ What's in the Image
- Application binary
- Wire-generated code (created during build)
- Database migrations
- Dependencies

### ❌ What's NOT in the Image
- Configuration files (mounted at runtime)
- Secrets (never baked into image)
- User data

### 🔐 Config File Security
```bash
# Always secure your config
chmod 600 /opt/srmt/config/prod.yaml
chown root:root /opt/srmt/config/prod.yaml

# Mount as read-only
-v /opt/srmt/config:/app/config:ro
```

### 🔄 Update Process
1. Build new image with version tag
2. Push to Docker Hub
3. Pull on server
4. Recreate container (config stays the same)

---

## Complete Example (VPS Deployment)

### 1. On Your Machine (Build)
```bash
# Clone repo
git clone https://github.com/yourusername/srmt-prime.git
cd srmt-prime

# Build image (Wire code auto-generated)
docker build -t yourusername/srmt-admin:v1.0.0 .

# Push to Docker Hub
docker login
docker push yourusername/srmt-admin:v1.0.0
docker tag yourusername/srmt-admin:v1.0.0 yourusername/srmt-admin:latest
docker push yourusername/srmt-admin:latest
```

### 2. On Production Server (Deploy)
```bash
# Setup directories
sudo mkdir -p /opt/srmt/config
sudo chown $USER:$USER /opt/srmt

# Copy config (from your machine)
# On local machine:
scp config/prod.yaml user@server:/opt/srmt/config/

# Or create on server:
ssh user@server
cat > /opt/srmt/config/prod.yaml << 'EOF'
env: "prod"
storage_path: "postgresql://srmt_user:PASSWORD@localhost:5432/srmt"
jwt:
  secret: "YOUR_SECURE_SECRET"
# ... add all settings
EOF

# Secure config
chmod 600 /opt/srmt/config/prod.yaml

# Pull and run
docker pull yourusername/srmt-admin:latest
docker run -d \
  --name srmt-admin \
  --restart unless-stopped \
  -p 9010:9010 \
  -v /opt/srmt/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  yourusername/srmt-admin:latest

# Check logs
docker logs -f srmt-admin

# Test
curl http://localhost:9010/api/v3/analytics
```

### 3. Future Updates
```bash
# Pull new version
docker pull yourusername/srmt-admin:v1.1.0

# Stop old
docker stop srmt-admin
docker rm srmt-admin

# Run new (same config)
docker run -d \
  --name srmt-admin \
  --restart unless-stopped \
  -p 9010:9010 \
  -v /opt/srmt/config:/app/config:ro \
  -e CONFIG_PATH=/app/config/prod.yaml \
  yourusername/srmt-admin:v1.1.0
```

---

## Troubleshooting

### Config not found
```bash
# Check file exists
ls -la /opt/srmt/config/prod.yaml

# Check permissions
chmod 644 /opt/srmt/config/prod.yaml
```

### Container crashes
```bash
# Check logs
docker logs srmt-admin

# Check config syntax
cat /opt/srmt/config/prod.yaml | grep -v "^#"
```

### Can't connect to database
```bash
# Check database is running
docker ps

# Check connection from container
docker exec srmt-admin ping postgres
```

---

## Documentation

- **Full Guide:** [docs/PRODUCTION_DEPLOYMENT.md](docs/PRODUCTION_DEPLOYMENT.md)
- **Docker Guide:** [docs/DOCKER.md](docs/DOCKER.md)
- **Config Guide:** [docs/CONFIG.md](docs/CONFIG.md)
- **Wire FAQ:** [docs/WIRE_FAQ.md](docs/WIRE_FAQ.md)

## Scripts

- **Server Setup:** `scripts/setup-server.sh`
- **Deploy:** `scripts/deploy-production.sh`
