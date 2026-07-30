#!/bin/sh

set -u

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
  echo "install-hooks: run this command inside a Git worktree" >&2
  exit 1
}

if [ ! -x "$repository_root/.githooks/pre-commit" ] ||
  [ ! -x "$repository_root/.githooks/commit-msg" ]; then
  echo "install-hooks: repository-managed hooks are missing or not executable" >&2
  exit 1
fi

# Keep a non-newline marker so command substitution cannot discard output
# separators. This distinguishes one managed value from duplicate, empty, or
# otherwise conflicting local values.
configured_hooks_paths_with_marker=$(
  git config --local --get-all core.hooksPath 2>/dev/null
  config_status=$?
  printf 'x'
  exit "$config_status"
)
config_status=$?

if [ "$config_status" -eq 0 ]; then
  if [ "$configured_hooks_paths_with_marker" = "$(printf '.githooks\nx')" ]; then
    echo "✓ git hooks already configured (core.hooksPath=.githooks)"
    exit 0
  fi

  echo "install-hooks: refusing to replace conflicting repository-local core.hooksPath value(s)" >&2
  exit 1
fi

if [ "$config_status" -ne 1 ]; then
  echo "install-hooks: could not read repository-local core.hooksPath" >&2
  exit 1
fi

if ! git config --local core.hooksPath .githooks; then
  echo "install-hooks: could not configure repository-local core.hooksPath" >&2
  exit 1
fi

echo "✓ git hooks installed (core.hooksPath=.githooks)"
