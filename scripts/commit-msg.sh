#!/usr/bin/env bash
# git commit-msg hook: normalizes the commit message to nightshift's
# Conventional Commits subset, rejecting it if the type prefix is
# missing or unrecognized.
#
# Install via: scripts/install-hooks.sh (symlinks this into .git/hooks/commit-msg)
set -euo pipefail

# Hooks are invoked by git with CWD set to the repo's top-level working
# directory, so a repo-relative path is more reliable here than resolving
# BASH_SOURCE (which doesn't follow the .git/hooks/ symlink on macOS).
REPO_ROOT="$(git rev-parse --show-toplevel)"
MSG_FILE="$1"
# git passes a second arg (commit source) for merge, squash, commit
# (amend/-c/-C), and template-derived messages. Merge and revert commits
# in particular have their own conventional subject lines ("Merge branch
# ...", "Revert \"...\"") that don't fit the <type>: <subject> shape, so
# normalizing/rejecting them would block routine `git merge --no-ff` and
# `git revert` usage. Skip normalization for those sources.
SOURCE="${2:-}"

case "$SOURCE" in
  merge|squash)
    exit 0
    ;;
esac

if head -n1 "$MSG_FILE" | grep -qE '^(Merge |Revert ")'; then
  exit 0
fi

set +e
"$REPO_ROOT/scripts/commit-msg-normalize.sh" "$MSG_FILE"
STATUS=$?
set -e

if [[ $STATUS -ne 0 ]]; then
  echo "" >&2
  echo "❌ Commit message rejected. Expected: <type>: <subject>" >&2
  echo "   Valid types: feat fix docs refactor test chore perf ci" >&2
  echo "   Bypass in an emergency with: git commit --no-verify" >&2
  exit "$STATUS"
fi

exit 0
