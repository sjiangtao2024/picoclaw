#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
WEB_DIR="${REPO_ROOT}/web"
OUT_DIR="${1:-${REPO_ROOT}/releases/h618}"
OUTPUT_NAME="${2:-picoclaw-web-linux-arm64}"

echo "==> Installing frontend dependencies"
cd "${WEB_DIR}/frontend"
corepack pnpm install --frozen-lockfile

echo "==> Building embedded frontend assets"
corepack pnpm build:backend

echo "==> Building linux/arm64 launcher binary"
cd "${WEB_DIR}"
make build WEB_GO='CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go'

mkdir -p "${OUT_DIR}"
cp "${WEB_DIR}/build/picoclaw-launcher" "${OUT_DIR}/${OUTPUT_NAME}"
chmod +x "${OUT_DIR}/${OUTPUT_NAME}"

cat <<EOF
Built:
  ${OUT_DIR}/${OUTPUT_NAME}

Suggested deploy layout on H618:
  /opt/picoclaw/current/${OUTPUT_NAME}
  /data/picoclaw/config.json
  /data/picoclaw/launcher-config.json

Run example:
  /opt/picoclaw/current/${OUTPUT_NAME} --no-browser /data/picoclaw/config.json
EOF
