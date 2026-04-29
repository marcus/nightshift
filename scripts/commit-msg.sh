#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "commit-msg: missing commit message file"
  exit 1
fi

MSG_FILE=$1
if [[ ! -f "$MSG_FILE" ]]; then
  echo "commit-msg: message file not found: $MSG_FILE"
  exit 1
fi

SUBJECT=$(sed -n '/^[[:space:]]*#/d; /^[[:space:]]*$/d; {p; q;}' "$MSG_FILE")

fail() {
  echo "commit-msg: invalid subject"
  echo "  found: ${SUBJECT:-<empty>}"
  echo "  want:  type(scope): description"
  echo "  types: build chore ci docs feat fix perf refactor style test"
  echo "  rules: <=72 chars, non-empty description; merge/revert allowed"
  exit 1
}

if [[ -z "$SUBJECT" ]]; then
  fail
fi

if [[ "$SUBJECT" =~ ^Merge[[:space:]] ]] || [[ "$SUBJECT" =~ ^Revert[[:space:]] ]]; then
  exit 0
fi

if (( ${#SUBJECT} > 72 )); then
  fail
fi

TYPE='(build|chore|ci|docs|feat|fix|perf|refactor|style|test)'
SCOPE='(\([A-Za-z0-9._/#-]+\))?'

if [[ ! "$SUBJECT" =~ ^${TYPE}${SCOPE}:[[:space:]][^[:space:]].* ]]; then
  fail
fi
