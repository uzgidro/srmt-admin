#!/bin/bash
# Production Deployment Script for SRMT Admin
# Usage: ./deploy-production.sh [version]

set -e

# Configuration
IMAGE_NAME="${DOCKER_IMAGE:-yourusername/srmt-admin}"
VERSION="${1:-latest}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/srmt}"
CONFIG_FILE="$DEPLOY_DIR/config/prod.yaml"
CONTAINER_NAME="srmt-admin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Deploying SRMT Admin $VERSION"
echo "📦 Image: $IMAGE_NAME:$VERSION"
echo "📁 Deploy directory: $DEPLOY_DIR"
echo ""

# Pre-flight checks
echo "📋 Running pre-flight checks..."

if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}❌ Config file not found: $CONFIG_FILE${NC}"
    echo "Please create the config file first:"
    echo "  sudo mkdir -p $DEPLOY_DIR/config"
    echo "  sudo nano $DEPLOY_DIR/config/prod.yaml"
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Pre-flight checks passed${NC}\n"

# Backup current config
echo "💾 Backing up current config..."
BACKUP_FILE="$CONFIG_FILE.backup.$(date +%Y%m%d_%H%M%S)"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo -e "${GREEN}✅ Config backed up to: $BACKUP_FILE${NC}\n"

# Pull new image
echo "⬇️  Pulling image $IMAGE_NAME:$VERSION..."
if docker pull "$IMAGE_NAME:$VERSION"; then
    echo -e "${GREEN}✅ Image pulled successfully${NC}\n"
else
    echo -e "${RED}❌ Failed to pull image${NC}"
    exit 1
fi

# Stop and remove old container
echo "🛑 Stopping old container..."
if docker stop "$CONTAINER_NAME" 2>/dev/null; then
    echo -e "${GREEN}✅ Old container stopped${NC}"
else
    echo -e "${YELLOW}⚠️  No old container found${NC}"
fi

if docker rm "$CONTAINER_NAME" 2>/dev/null; then
    echo -e "${GREEN}✅ Old container removed${NC}\n"
else
    echo -e "${YELLOW}⚠️  No old container to remove${NC}\n"
fi

# Run new container
echo "▶️  Starting new container..."
docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p 9010:9010 \
  -v "$DEPLOY_DIR/config:/app/config:ro" \
  -e CONFIG_PATH=/app/config/prod.yaml \
  "$IMAGE_NAME:$VERSION"

echo -e "${GREEN}✅ Container started${NC}\n"

# Wait for container to be healthy
echo "🏥 Waiting for application to be healthy..."
sleep 5

# Health check
HEALTH_CHECK_URL="http://localhost:9010/api/v3/analytics"
for i in {1..30}; do
    if curl -f -s "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Health check passed!${NC}\n"
        echo "🎉 Deployment successful!"
        echo ""
        echo "📊 Container Status:"
        docker ps -f name="$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        echo ""
        echo "📝 Recent logs:"
        docker logs --tail 20 "$CONTAINER_NAME"
        echo ""
        echo "🌐 Application is available at: http://localhost:9010"
        exit 0
    fi
    echo -n "."
    sleep 2
done

echo -e "\n${RED}❌ Health check failed!${NC}"
echo "Container logs:"
docker logs --tail 50 "$CONTAINER_NAME"
echo ""
echo "Container status:"
docker ps -a -f name="$CONTAINER_NAME"
exit 1
