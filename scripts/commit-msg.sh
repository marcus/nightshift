#!/usr/bin/env bash
# commit-msg hook for nightshift
#
# Enforces Conventional Commits on every commit message and rewrites the
# message file into canonical form before the commit is created. Messages that
# cannot be normalized (missing/unknown type, capitalized or overlong subject)
# are rejected with a non-zero exit so the commit is aborted.
#
# Install:
#   make install-hooks
#   # or manually:
#   ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg
set -u

if [[ $# -lt 1 ]]; then
  echo "usage: commit-msg <message-file>" >&2
  exit 1
fi

MSG_FILE="$1"

# Resolve the nightshift binary: prefer the one on $PATH, fall back to
# running the current source tree.
NIGHTSHIFT="$(command -v nightshift || true)"
if [[ -z "$NIGHTSHIFT" ]]; then
  NIGHTSHIFT="go run github.com/marcus/nightshift/cmd/nightshift"
fi

ERR_FILE="$(mktemp -t nightshift-commit-msg)"
trap 'rm -f "$ERR_FILE"' EXIT

NORMALIZED="$($NIGHTSHIFT commit normalize --file "$MSG_FILE" 2>"$ERR_FILE")"
STATUS=$?

if [[ $STATUS -ne 0 ]]; then
  echo "🪡 commit-msg: message does not follow Conventional Commits" >&2
  sed 's/^/    /' "$ERR_FILE" >&2
  echo "" >&2
  echo "    Expected format: <type>(<scope>): <subject>" >&2
  echo "    Types: feat fix docs style refactor perf test build ci chore revert" >&2
  echo "    (rewrite your message, or bypass with: git commit --no-verify)" >&2
  exit 1
fi

# Rewrite the message file into canonical form.
printf '%s\n' "$NORMALIZED" > "$MSG_FILE"
echo "🪡 commit-msg: normalized"
exit 0
