#!/usr/bin/env bash
set -euo pipefail

tag="${AUX_IMAGE_TAG:-${1:-}}"
if [ -z "$tag" ]; then
  echo "usage: AUX_IMAGE_TAG=aux-YYYYMMDD.N $0"
  echo "   or: $0 aux-YYYYMMDD.N"
  exit 2
fi

service_dir="/root/sub2api/sub2api-sign"
backup_dir="/root/sub2api/rollback/sign-$(date +%Y%m%d-%H%M%S)"

cd "$service_dir"

if [ ! -f .env ]; then
  echo "$service_dir/.env is required and must stay on the VPS"
  exit 1
fi

mkdir -p "$backup_dir"
cp -a docker-compose.sign.yml .env "$backup_dir"/
docker inspect sub2api-sign-api > "$backup_dir/sub2api-sign-api.inspect.json" 2>/dev/null || true
docker inspect sub2api-sign-web > "$backup_dir/sub2api-sign-web.inspect.json" 2>/dev/null || true

export AUX_IMAGE_TAG="$tag"

docker compose -f docker-compose.sign.yml pull
docker compose -f docker-compose.sign.yml up -d

curl -fsS http://127.0.0.1:8092/healthz >/dev/null
curl -fsS http://127.0.0.1:4174/external/sign/ >/dev/null

echo "sub2api-sign is running with AUX_IMAGE_TAG=$AUX_IMAGE_TAG"
echo "rollback material: $backup_dir"
