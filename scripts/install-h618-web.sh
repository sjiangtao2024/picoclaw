#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-picoclaw-web-linux-arm64> [path-to-picoclaw]"
  exit 1
fi

BINARY_SRC=$(realpath "$1")
GATEWAY_SRC=${2:-}
SERVICE_NAME=${SERVICE_NAME:-picoclaw-web}
INSTALL_ROOT=${INSTALL_ROOT:-/opt/picoclaw/current}
DATA_ROOT=${DATA_ROOT:-/data/picoclaw}
SERVICE_PATH=${SERVICE_PATH:-/etc/systemd/system/${SERVICE_NAME}.service}
TARGET_BINARY="${INSTALL_ROOT}/picoclaw-web-linux-arm64"
TARGET_GATEWAY="${INSTALL_ROOT}/picoclaw"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
SERVICE_TEMPLATE="${REPO_ROOT}/deploy/systemd/picoclaw-web.service"

if [[ -z "${GATEWAY_SRC}" ]]; then
  for candidate in \
    "$(dirname "${BINARY_SRC}")/picoclaw" \
    "$(dirname "${BINARY_SRC}")/picoclaw-linux-arm64"; do
    if [[ -f "${candidate}" ]]; then
      GATEWAY_SRC="${candidate}"
      break
    fi
  done
fi

if [[ ! -f "${BINARY_SRC}" ]]; then
  echo "Binary not found: ${BINARY_SRC}"
  exit 1
fi
if [[ -z "${GATEWAY_SRC}" ]]; then
  echo "Gateway binary not found. Provide path to picoclaw as the second argument."
  exit 1
fi
GATEWAY_SRC=$(realpath "${GATEWAY_SRC}")
if [[ ! -f "${GATEWAY_SRC}" ]]; then
  echo "Gateway binary not found: ${GATEWAY_SRC}"
  exit 1
fi

mkdir -p "${INSTALL_ROOT}" "${DATA_ROOT}" "${DATA_ROOT}/logs"
install -m 0755 "${BINARY_SRC}" "${TARGET_BINARY}"
install -m 0755 "${GATEWAY_SRC}" "${TARGET_GATEWAY}"

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

mkdir -p "$(dirname "${SERVICE_PATH}")"
install -m 0644 "${SERVICE_TEMPLATE}" "${SERVICE_PATH}"
systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
