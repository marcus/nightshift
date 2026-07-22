#!/bin/sh

# Enable this repository's version-controlled hooks without changing global Git
# configuration.
set -eu

repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)

if ! git -C "$repo_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "not a Git worktree: $repo_root" >&2
  exit 1
fi

git -C "$repo_root" config --local core.hooksPath .githooks
echo "repository hooks enabled (.githooks)"
