#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "commit-msg: expected path to commit message file" >&2
  exit 1
fi

MSG_FILE=$1
MAX_SUBJECT_LEN=72

subject=$(sed -n '/^[[:space:]]*#/d; /^[[:space:]]*$/d; p; q' "$MSG_FILE")

if [[ -z "$subject" ]]; then
  echo "commit-msg: empty commit subject" >&2
  exit 1
fi

# Let Git-generated messages and autosquash commits through unchanged.
if [[ "$subject" =~ ^Merge[[:space:]] ]] ||
  [[ "$subject" =~ ^Revert[[:space:]]\".*\"$ ]] ||
  [[ "$subject" =~ ^(fixup|squash)![[:space:]] ]]; then
  exit 0
fi

if (( ${#subject} > MAX_SUBJECT_LEN )); then
  echo "commit-msg: subject must be ${MAX_SUBJECT_LEN} characters or fewer" >&2
  echo "  $subject" >&2
  exit 1
fi

if [[ "$subject" == *. ]]; then
  echo "commit-msg: subject must not end with a period" >&2
  echo "  $subject" >&2
  exit 1
fi

pattern='^(build|chore|ci|docs|feat|fix|perf|refactor|style|test)(\([a-z0-9._/-]+\))?(!)?: [^[:space:]].*$'

if [[ ! "$subject" =~ $pattern ]]; then
  cat >&2 <<'EOF'
commit-msg: subject must use Conventional Commit format

Format:
  type(scope): concise subject

Allowed types:
  build chore ci docs feat fix perf refactor style test

Examples:
  feat(tasks): add queue filtering
  fix: handle empty budget snapshots
  docs: clarify hook installation
EOF
  echo "" >&2
  echo "Got:" >&2
  echo "  $subject" >&2
  exit 1
fi
