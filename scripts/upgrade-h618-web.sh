#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-new-picoclaw-web-linux-arm64> [path-to-new-picoclaw]"
  exit 1
fi

BINARY_SRC=$(realpath "$1")
GATEWAY_SRC=${2:-}
SERVICE_NAME=${SERVICE_NAME:-picoclaw-web}
INSTALL_ROOT=${INSTALL_ROOT:-/root/picoclaw}
BIN_DIR=${BIN_DIR:-${INSTALL_ROOT}/bin}
TARGET_BINARY="${BIN_DIR}/picoclaw-web"
TARGET_GATEWAY="${BIN_DIR}/picoclaw"
BACKUP_DIR=${BACKUP_DIR:-${INSTALL_ROOT}/backups}
COMMAND_BIN_DIR=${COMMAND_BIN_DIR:-/usr/local/bin}
CONFIG_DIR=${CONFIG_DIR:-${INSTALL_ROOT}/config}
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

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
if [[ -n "${GATEWAY_SRC}" ]]; then
  GATEWAY_SRC=$(realpath "${GATEWAY_SRC}")
  if [[ ! -f "${GATEWAY_SRC}" ]]; then
    echo "Gateway binary not found: ${GATEWAY_SRC}"
    exit 1
  fi
elif [[ ! -f "${TARGET_GATEWAY}" ]]; then
  echo "Gateway binary not found. Provide path to picoclaw as the second argument."
  exit 1
fi

mkdir -p "${BIN_DIR}" "${BACKUP_DIR}"

if [[ -f "${TARGET_BINARY}" ]]; then
  cp "${TARGET_BINARY}" "${BACKUP_DIR}/picoclaw-web-linux-arm64.${TIMESTAMP}"
fi
if [[ -f "${TARGET_GATEWAY}" ]]; then
  cp "${TARGET_GATEWAY}" "${BACKUP_DIR}/picoclaw.${TIMESTAMP}"
fi

systemctl stop "${SERVICE_NAME}"
install -m 0755 "${BINARY_SRC}" "${TARGET_BINARY}"
if [[ -n "${GATEWAY_SRC}" ]]; then
  install -m 0755 "${GATEWAY_SRC}" "${TARGET_GATEWAY}"
fi
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
systemctl start "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager

echo "Upgrade complete."
echo "Backup directory: ${BACKUP_DIR}"
