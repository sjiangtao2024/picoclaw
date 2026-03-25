#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-new-picoclaw-web-linux-arm64>"
  exit 1
fi

BINARY_SRC=$(realpath "$1")
SERVICE_NAME=${SERVICE_NAME:-picoclaw-web}
INSTALL_ROOT=${INSTALL_ROOT:-/opt/picoclaw/current}
TARGET_BINARY="${INSTALL_ROOT}/picoclaw-web-linux-arm64"
BACKUP_DIR=${BACKUP_DIR:-/opt/picoclaw/backups}
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

if [[ ! -f "${BINARY_SRC}" ]]; then
  echo "Binary not found: ${BINARY_SRC}"
  exit 1
fi

mkdir -p "${INSTALL_ROOT}" "${BACKUP_DIR}"

if [[ -f "${TARGET_BINARY}" ]]; then
  cp "${TARGET_BINARY}" "${BACKUP_DIR}/picoclaw-web-linux-arm64.${TIMESTAMP}"
fi

systemctl stop "${SERVICE_NAME}"
install -m 0755 "${BINARY_SRC}" "${TARGET_BINARY}"
systemctl start "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager

echo "Upgrade complete."
echo "Backup directory: ${BACKUP_DIR}"
