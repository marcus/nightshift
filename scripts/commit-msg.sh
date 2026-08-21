#!/usr/bin/env bash
# commit-msg hook for nightshift
#
# Enforces Conventional Commits on every commit message and rewrites the
# message file into canonical form before the commit is created. Messages that
# cannot be normalized (missing/unknown type, missing or overlong subject) are
# rejected with a non-zero exit so the commit is aborted.
#
# Install:
#   make install-hooks
#   # or manually:
#   ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg
#   chmod +x scripts/commit-msg.sh
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: commit-msg <message-file>" >&2
  exit 1
fi

MSG_FILE="$1"

# Resolve the nightshift command: prefer the binary on $PATH, fall back to
# running from the source tree via go run. Kept as an array so the multi-word
# fallback is expanded as separate words when invoked.
NIGHTSHIFT=(nightshift)
if ! command -v nightshift >/dev/null 2>&1; then
  if ! command -v go >/dev/null 2>&1; then
    echo "🪡 commit-msg: neither 'nightshift' nor 'go' found on PATH;" >&2
    echo "    cannot check the commit message" >&2
    exit 1
  fi
  NIGHTSHIFT=(go run github.com/marcus/nightshift/cmd/nightshift)
fi

# Make sure the tool itself can run (binary present, go run compiles) before
# blaming the commit message for any failure.
probe_status=0
"${NIGHTSHIFT[@]}" commit normalize --help >/dev/null 2>&1 || probe_status=$?
if [[ $probe_status -ne 0 ]]; then
  echo "🪡 commit-msg: failed to run nightshift (exit $probe_status);" >&2
  echo "    this is a tooling problem, not a message-format problem" >&2
  exit 1
fi

ERR_FILE="$(mktemp "${TMPDIR:-/tmp}/nightshift-commit-msg.XXXXXX")"
trap 'rm -f "$ERR_FILE"' EXIT

status=0
"${NIGHTSHIFT[@]}" commit normalize --file "$MSG_FILE" 2>"$ERR_FILE" || status=$?

if [[ $status -ne 0 ]]; then
  echo "🪡 commit-msg: message does not follow Conventional Commits" >&2
  sed 's/^/    /' "$ERR_FILE" >&2 || true
  echo "" >&2
  echo "    Expected format: <type>(<scope>)!: <subject>" >&2
  echo "    Types: feat fix docs style refactor test chore perf build ci revert" >&2
  echo "    (rewrite your message, or bypass with: git commit --no-verify)" >&2
  exit 1
fi

echo "🪡 commit-msg: normalized"
exit 0
