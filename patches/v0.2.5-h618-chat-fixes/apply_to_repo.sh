#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 REPO_DIR" >&2
  exit 1
fi

repo_dir=$1
patch_dir=/home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes

cd "$repo_dir"

git apply --check "$patch_dir/0001-web-config-validation.patch"
git apply --check "$patch_dir/0002-frontend-chat-reconnect-and-polling.patch"
git apply --check "$patch_dir/0003-pico-agent-diagnostics-and-allowlist.patch"
git apply --check "$patch_dir/0004-skillhub-defaults-and-compat.patch"

git apply "$patch_dir/0001-web-config-validation.patch"
git apply "$patch_dir/0002-frontend-chat-reconnect-and-polling.patch"
git apply "$patch_dir/0003-pico-agent-diagnostics-and-allowlist.patch"
git apply "$patch_dir/0004-skillhub-defaults-and-compat.patch"

echo "patches applied to $repo_dir"
