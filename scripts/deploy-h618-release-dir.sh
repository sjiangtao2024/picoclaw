#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
RELEASE_DIR=$(realpath -m "${1:-${REPO_ROOT}/releases/h618}")
WEB_BINARY="${RELEASE_DIR}/picoclaw-web-linux-arm64"
GATEWAY_BINARY="${RELEASE_DIR}/picoclaw"

exec "${SCRIPT_DIR}/upgrade-h618-web.sh" "${WEB_BINARY}" "${GATEWAY_BINARY}"
