#!/bin/bash
# NOFX - start containers with local data persistence
# Usage: ./docker-run.sh [backend|frontend|all]

set -e

cd "$(dirname "$0")"

# Source env vars
set -a
source .env 2>/dev/null || true
set +a

BACKEND_IMAGE="${BACKEND_IMAGE:-ghcr.io/zongwei-wu/nofx-1/nofx-backend:dev-cursor-amd64}"
FRONTEND_IMAGE="${FRONTEND_IMAGE:-localhost/nofx-frontend:latest}"
BACKEND_PORT="${NOFX_BACKEND_PORT:-8080}"
FRONTEND_PORT="${NOFX_FRONTEND_PORT:-3000}"
TZ="${NOFX_TIMEZONE:-Asia/Shanghai}"
DATA_KEY="${DATA_ENCRYPTION_KEY}"
JWT="${JWT_SECRET}"

start_backend() {
  echo "Starting backend container..."
  podman rm -f nofx-trading 2>/dev/null || true
  podman run -d --name nofx-trading \
    -p ${BACKEND_PORT}:8080 \
    -v ./config.json:/app/config.json:ro \
    -v ./config.db:/app/config.db \
    -v ./decision_logs:/app/decision_logs \
    -v ./data/copy-trading:/app/data/copy-trading \
    -v ./prompts:/app/prompts \
    -v ./secrets:/app/secrets:ro \
    -v /etc/localtime:/etc/localtime:ro \
    -e TZ=${TZ} \
    -e DATA_ENCRYPTION_KEY="${DATA_KEY}" \
    -e JWT_SECRET="${JWT}" \
    -e COPY_TRADING_CACHE_DIR=/app/data/copy-trading \
    ${BACKEND_IMAGE}
  echo "Backend started on port ${BACKEND_PORT}"
}

start_frontend() {
  echo "Starting frontend container..."
  podman rm -f nofx-frontend 2>/dev/null || true
  podman run -d --name nofx-frontend \
    -p ${FRONTEND_PORT}:80 \
    localhost/nofx-frontend:latest
  echo "Frontend started on port ${FRONTEND_PORT}"
}

case "${1:-all}" in
  backend) start_backend ;;
  frontend) start_frontend ;;
  all)
    start_backend
    start_frontend
    ;;
  *)
    echo "Usage: $0 [backend|frontend|all]"
    exit 1
    ;;
esac

echo "Done. Containers:"
podman ps --filter name=nofx
