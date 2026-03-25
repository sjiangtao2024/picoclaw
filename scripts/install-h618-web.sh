#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-picoclaw-web-linux-arm64> [path-to-picoclaw]"
  exit 1
fi

BINARY_SRC=$(realpath "$1")
GATEWAY_SRC=${2:-}
SERVICE_NAME=${SERVICE_NAME:-picoclaw-web}
INSTALL_ROOT=${INSTALL_ROOT:-/root/picoclaw}
BIN_DIR=${BIN_DIR:-${INSTALL_ROOT}/bin}
CONFIG_DIR=${CONFIG_DIR:-${INSTALL_ROOT}/config}
WORKSPACE_DIR=${WORKSPACE_DIR:-${INSTALL_ROOT}/workspace}
LOG_DIR=${LOG_DIR:-${INSTALL_ROOT}/logs}
COMMAND_BIN_DIR=${COMMAND_BIN_DIR:-/usr/local/bin}
SERVICE_PATH=${SERVICE_PATH:-/etc/systemd/system/${SERVICE_NAME}.service}
TARGET_BINARY="${BIN_DIR}/picoclaw-web"
TARGET_GATEWAY="${BIN_DIR}/picoclaw"
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

mkdir -p "${BIN_DIR}" "${CONFIG_DIR}" "${WORKSPACE_DIR}" "${LOG_DIR}"
install -m 0755 "${BINARY_SRC}" "${TARGET_BINARY}"
install -m 0755 "${GATEWAY_SRC}" "${TARGET_GATEWAY}"

mkdir -p "${COMMAND_BIN_DIR}"
cat > "${COMMAND_BIN_DIR}/picoclaw" <<EOF
#!/usr/bin/env bash
export PICOCLAW_HOME="${INSTALL_ROOT}"
export PICOCLAW_CONFIG="${CONFIG_DIR}/config.json"
exec "${TARGET_GATEWAY}" "\$@"
EOF
chmod 0755 "${COMMAND_BIN_DIR}/picoclaw"

cat > "${COMMAND_BIN_DIR}/picoclaw-web" <<EOF
#!/usr/bin/env bash
export PICOCLAW_HOME="${INSTALL_ROOT}"
export PICOCLAW_CONFIG="${CONFIG_DIR}/config.json"
exec "${TARGET_BINARY}" "\$@"
EOF
chmod 0755 "${COMMAND_BIN_DIR}/picoclaw-web"

if [[ ! -f "${CONFIG_DIR}/launcher-config.json" ]]; then
  cat > "${CONFIG_DIR}/launcher-config.json" <<'EOF'
{
  "port": 18800,
  "public": true,
  "allowed_cidrs": [
    "192.168.1.0/24"
  ]
}
EOF
fi

if [[ ! -f "${CONFIG_DIR}/config.json" ]]; then
  cat > "${CONFIG_DIR}/config.json" <<EOF
{
  "version": 1,
  "agents": {
    "defaults": {
      "workspace": "${WORKSPACE_DIR}"
    }
  }
}
EOF
  echo "Created placeholder config at ${CONFIG_DIR}/config.json"
  echo "Edit it before enabling production channels and models."
fi

mkdir -p "$(dirname "${SERVICE_PATH}")"
sed \
  -e "s|__PICOCLAW_ROOT__|${INSTALL_ROOT}|g" \
  -e "s|__PICOCLAW_WEB_BIN__|${TARGET_BINARY}|g" \
  -e "s|__PICOCLAW_CONFIG__|${CONFIG_DIR}/config.json|g" \
  "${SERVICE_TEMPLATE}" > "${SERVICE_PATH}"
chmod 0644 "${SERVICE_PATH}"
systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
