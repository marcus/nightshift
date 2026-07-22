#!/bin/sh

# Enable this repository's version-controlled hooks without changing global Git
# configuration.
set -eu

repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)

if ! git -C "$repo_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "not a Git worktree: $repo_root" >&2
  exit 1
fi

if existing_hooks_path=$(git -C "$repo_root" config --local --get core.hooksPath 2>/dev/null); then
  if [ "$existing_hooks_path" != ".githooks" ]; then
    cat >&2 <<EOF
repository already uses core.hooksPath=$existing_hooks_path
merge those hooks into .githooks or remove the local setting, then rerun this installer
EOF
    exit 1
  fi

  echo "repository hooks already enabled (.githooks)"
  exit 0
fi

git -C "$repo_root" config --local core.hooksPath .githooks
echo "repository hooks enabled (.githooks)"
