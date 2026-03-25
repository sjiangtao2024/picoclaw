#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-picoclaw-web-linux-arm64>"
  exit 1
fi

BINARY_SRC=$(realpath "$1")
SERVICE_NAME=${SERVICE_NAME:-picoclaw-web}
INSTALL_ROOT=${INSTALL_ROOT:-/opt/picoclaw/current}
DATA_ROOT=${DATA_ROOT:-/data/picoclaw}
SERVICE_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
TARGET_BINARY="${INSTALL_ROOT}/picoclaw-web-linux-arm64"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
SERVICE_TEMPLATE="${REPO_ROOT}/deploy/systemd/picoclaw-web.service"

if [[ ! -f "${BINARY_SRC}" ]]; then
  echo "Binary not found: ${BINARY_SRC}"
  exit 1
fi

mkdir -p "${INSTALL_ROOT}" "${DATA_ROOT}" "${DATA_ROOT}/logs"
install -m 0755 "${BINARY_SRC}" "${TARGET_BINARY}"

if [[ ! -f "${DATA_ROOT}/launcher-config.json" ]]; then
  cat > "${DATA_ROOT}/launcher-config.json" <<'EOF'
{
  "port": 18800,
  "public": true,
  "allowed_cidrs": [
    "192.168.1.0/24"
  ]
}
EOF
fi

if [[ ! -f "${DATA_ROOT}/config.json" ]]; then
  cat > "${DATA_ROOT}/config.json" <<'EOF'
{
  "version": 1
}
EOF
  echo "Created placeholder config at ${DATA_ROOT}/config.json"
  echo "Edit it before enabling production channels and models."
fi

install -m 0644 "${SERVICE_TEMPLATE}" "${SERVICE_PATH}"
systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
