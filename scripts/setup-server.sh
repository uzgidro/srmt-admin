#!/bin/bash
# Server Setup Script for SRMT Admin
# Run this on your production server to prepare for deployment

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔧 Setting up server for SRMT Admin deployment"
echo ""

# Configuration
DEPLOY_DIR="${DEPLOY_DIR:-/opt/srmt}"
USER="${DEPLOY_USER:-$USER}"

echo "📁 Creating directory structure..."
sudo mkdir -p "$DEPLOY_DIR/config"
sudo mkdir -p "$DEPLOY_DIR/data"
sudo chown -R "$USER:$USER" "$DEPLOY_DIR"
echo -e "${GREEN}✅ Directory structure created at $DEPLOY_DIR${NC}\n"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker not found. Installing Docker...${NC}"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker "$USER"
    rm get-docker.sh
    echo -e "${GREEN}✅ Docker installed${NC}"
    echo "⚠️  Please log out and log back in for Docker permissions to take effect"
else
    echo -e "${GREEN}✅ Docker is already installed${NC}"
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}⚠️  docker-compose not found. Installing...${NC}"
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    echo -e "${GREEN}✅ docker-compose installed${NC}"
else
    echo -e "${GREEN}✅ docker-compose is already installed${NC}"
fi

# Create config file if not exists
CONFIG_FILE="$DEPLOY_DIR/config/prod.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    echo ""
    echo "📝 Creating example config file..."
    cat > "$CONFIG_FILE" << 'EOF'
# Production Configuration for SRMT Admin
# IMPORTANT: Update all values marked with YOUR_* before deploying!

env: "prod"

# PostgreSQL connection
storage_path: "postgresql://srmt_user:YOUR_DB_PASSWORD@localhost:5432/srmt?sslmode=disable"
migrations_path: "file://./migrations/postgres"

# Timezone
timezone: 'UTC'

# MongoDB configuration
mongo:
  host: 'localhost'
  port: '27017'
  username: 'admin'
  password: 'YOUR_MONGO_PASSWORD'
  auth_source: 'admin'

# MinIO object storage
minio:
  endpoint: "YOUR_MINIO_ENDPOINT"
  access_key: "YOUR_MINIO_ACCESS_KEY"
  secret_key: "YOUR_MINIO_SECRET_KEY"
  use_ssl: false

# HTTP server settings
http_server:
  address: "0.0.0.0:9010"
  timeout: 30s
  idle_timeout: 60s
  allowed_origins:
    - "https://your-domain.com"

# JWT authentication - CHANGE THESE!
jwt:
  secret: "YOUR_JWT_SECRET_CHANGE_THIS"
  access_timeout: 15m
  refresh_timeout: 168h

# API key for callbacks
callback_api_key: 'YOUR_API_KEY_CHANGE_THIS'

# MinIO bucket name
bucket: 'srmt-prod'
EOF

    chmod 600 "$CONFIG_FILE"
    echo -e "${GREEN}✅ Example config created at $CONFIG_FILE${NC}"
    echo -e "${YELLOW}⚠️  IMPORTANT: Edit this file and replace all YOUR_* values!${NC}"
    echo "   nano $CONFIG_FILE"
else
    echo -e "${GREEN}✅ Config file already exists${NC}"
fi

# Create docker-compose.yml
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.yml"
if [ ! -f "$COMPOSE_FILE" ]; then
    echo ""
    echo "📝 Creating docker-compose.yml..."
    cat > "$COMPOSE_FILE" << 'EOF'
version: '3.8'

services:
  app:
    image: yourusername/srmt-admin:latest
    container_name: srmt-admin
    restart: unless-stopped
    ports:
      - "9010:9010"
    volumes:
      - ./config:/app/config:ro
    environment:
      - CONFIG_PATH=/app/config/prod.yaml
    depends_on:
      postgres:
        condition: service_healthy
      mongodb:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    container_name: srmt-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: srmt
      POSTGRES_USER: srmt_user
      POSTGRES_PASSWORD: ${DB_PASSWORD:-changeme}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U srmt_user -d srmt"]
      interval: 10s
      timeout: 5s
      retries: 5

  mongodb:
    image: mongo:7
    container_name: srmt-mongodb
    restart: unless-stopped
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD:-changeme}
    volumes:
      - mongodb-data:/data/db
    ports:
      - "127.0.0.1:27017:27017"
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres-data:
  mongodb-data:
EOF

    echo -e "${GREEN}✅ docker-compose.yml created${NC}"
else
    echo -e "${GREEN}✅ docker-compose.yml already exists${NC}"
fi

# Create .env file
ENV_FILE="$DEPLOY_DIR/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo ""
    echo "📝 Creating .env file..."
    cat > "$ENV_FILE" << 'EOF'
# Database passwords
DB_PASSWORD=changeme_secure_password
MONGO_PASSWORD=changeme_secure_password
EOF

    chmod 600 "$ENV_FILE"
    echo -e "${GREEN}✅ .env file created${NC}"
    echo -e "${YELLOW}⚠️  IMPORTANT: Edit .env and set secure passwords!${NC}"
    echo "   nano $ENV_FILE"
else
    echo -e "${GREEN}✅ .env file already exists${NC}"
fi

echo ""
echo "🎉 Server setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit the configuration file:"
echo "   nano $CONFIG_FILE"
echo ""
echo "2. Edit the environment file:"
echo "   nano $ENV_FILE"
echo ""
echo "3. Update docker-compose.yml image name:"
echo "   nano $COMPOSE_FILE"
echo "   Change 'yourusername/srmt-admin' to your Docker Hub username"
echo ""
echo "4. Pull and run the application:"
echo "   cd $DEPLOY_DIR"
echo "   docker-compose pull"
echo "   docker-compose up -d"
echo ""
echo "5. Check logs:"
echo "   docker-compose logs -f app"
