#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
WEB_DIR="${REPO_ROOT}/web"
OUT_DIR=$(realpath -m "${1:-${REPO_ROOT}/releases/h618}")
OUTPUT_NAME="${2:-picoclaw-web-linux-arm64}"
GATEWAY_OUTPUT_NAME="${3:-picoclaw}"
ONBOARD_WORKSPACE_DIR="${REPO_ROOT}/cmd/picoclaw/internal/onboard/workspace"
GATEWAY_GO_TAGS="${GATEWAY_GO_TAGS:-goolm}"

echo "==> Installing frontend dependencies"
cd "${WEB_DIR}/frontend"
corepack pnpm install --frozen-lockfile

echo "==> Building embedded frontend assets"
corepack pnpm build:backend

mkdir -p "${OUT_DIR}"

echo "==> Building linux/arm64 launcher binary"
cd "${WEB_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "${OUT_DIR}/${OUTPUT_NAME}" ./backend

echo "==> Building linux/arm64 gateway binary"
cd "${REPO_ROOT}"
cleanup_onboard_workspace() {
  if [[ -d "${ONBOARD_WORKSPACE_DIR}" ]]; then
    rm -rf "${ONBOARD_WORKSPACE_DIR}"
  fi
}
trap cleanup_onboard_workspace EXIT
cp -R "${REPO_ROOT}/workspace" "${ONBOARD_WORKSPACE_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "${GATEWAY_GO_TAGS}" -o "${OUT_DIR}/${GATEWAY_OUTPUT_NAME}" ./cmd/picoclaw

chmod +x "${OUT_DIR}/${OUTPUT_NAME}"
chmod +x "${OUT_DIR}/${GATEWAY_OUTPUT_NAME}"

cat <<EOF
Built:
  ${OUT_DIR}/${OUTPUT_NAME}
  ${OUT_DIR}/${GATEWAY_OUTPUT_NAME}

Suggested deploy layout on H618:
  /opt/picoclaw/current/${OUTPUT_NAME}
  /opt/picoclaw/current/${GATEWAY_OUTPUT_NAME}
  /data/picoclaw/config.json
  /data/picoclaw/launcher-config.json

Run example:
  /opt/picoclaw/current/${OUTPUT_NAME} --no-browser /data/picoclaw/config.json
EOF
