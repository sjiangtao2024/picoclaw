#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <release-tag> [branch-suffix]"
  echo "Example: $0 v0.2.4 h618"
  exit 1
fi

RELEASE_TAG=$1
BRANCH_SUFFIX=${2:-h618}
UPSTREAM_REMOTE=${UPSTREAM_REMOTE:-upstream}
ORIGIN_REMOTE=${ORIGIN_REMOTE:-origin}
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
WORKTREE_PARENT=${WORKTREE_PARENT:-$(cd "${REPO_ROOT}/.." && pwd)}
BRANCH_NAME=${BRANCH_NAME:-custom/release-${RELEASE_TAG}-${BRANCH_SUFFIX}}
WORKTREE_DIR=${WORKTREE_DIR:-${WORKTREE_PARENT}/picoclaw-${RELEASE_TAG}-${BRANCH_SUFFIX}}

cd "${REPO_ROOT}"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Not inside a git repository: ${REPO_ROOT}"
  exit 1
fi

if [[ -n "$(git worktree list --porcelain | awk '/^worktree /{print $2}' | grep -F "${WORKTREE_DIR}" || true)" ]]; then
  echo "Worktree already exists: ${WORKTREE_DIR}"
  exit 1
fi

if git show-ref --verify --quiet "refs/heads/${BRANCH_NAME}"; then
  echo "Branch already exists: ${BRANCH_NAME}"
  exit 1
fi

if [[ "${SKIP_FETCH:-0}" != "1" ]]; then
  echo "Fetching ${UPSTREAM_REMOTE} tags and ${ORIGIN_REMOTE} branches..."
  git fetch "${UPSTREAM_REMOTE}" --tags
  git fetch "${ORIGIN_REMOTE}"
fi

if ! git rev-parse --verify --quiet "refs/tags/${RELEASE_TAG}" >/dev/null; then
  echo "Release tag not found after fetch: ${RELEASE_TAG}"
  exit 1
fi

mkdir -p "${WORKTREE_PARENT}"
git worktree add "${WORKTREE_DIR}" -b "${BRANCH_NAME}" "${RELEASE_TAG}"

cat <<EOF
Created worktree: ${WORKTREE_DIR}
Created branch:   ${BRANCH_NAME}

Recommended next steps:
  cd ${WORKTREE_DIR}
  git cherry-pick <patch-commit>...
  git push -u ${ORIGIN_REMOTE} ${BRANCH_NAME}
EOF
