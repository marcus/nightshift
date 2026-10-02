#!/usr/bin/env bash
# Normalizes a commit message file in place to the Conventional Commits
# subset used by nightshift: "<type>: <subject>" with an optional body.
#
# Usage: commit-msg-normalize.sh <path-to-commit-msg-file>
#
# Exit codes:
#   0 - message normalized successfully
#   1 - usage error (bad args / missing file)
#   2 - subject line has no recognized type prefix after normalization
set -euo pipefail

VALID_TYPES="feat fix docs refactor test chore perf ci"
SUBJECT_MAX_LEN=72

usage() {
  echo "Usage: $(basename "$0") <path-to-commit-msg-file>" >&2
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

MSG_FILE="$1"

if [[ ! -f "$MSG_FILE" ]]; then
  echo "commit-msg-normalize: no such file: $MSG_FILE" >&2
  exit 1
fi

is_valid_type() {
  local candidate="$1"
  local t
  for t in $VALID_TYPES; do
    [[ "$t" == "$candidate" ]] && return 0
  done
  return 1
}

# Read the file, strip trailing whitespace per line, drop comment lines
# (git includes "# ..." guidance lines), and collapse runs of 2+ blank
# lines down to a single blank line.
NORMALIZED=$(
  awk '
    /^#/ { next }
    {
      sub(/[ \t]+$/, "")
      if ($0 == "") {
        blank++
        if (blank > 1) next
      } else {
        blank = 0
      }
      print
    }
  ' "$MSG_FILE"
)

# Trim leading/trailing blank lines as a whole.
NORMALIZED=$(printf '%s\n' "$NORMALIZED" | sed -e '/./,$!d' -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')

SUBJECT=$(printf '%s\n' "$NORMALIZED" | head -n1)
REST=$(printf '%s\n' "$NORMALIZED" | tail -n +2)

# Extract "type" and "type(scope)" prefixes, tolerating a missing colon
# or missing space after the colon.
if [[ "$SUBJECT" =~ ^([a-zA-Z]+)(\([a-zA-Z0-9_-]+\))?:?[[:space:]]*(.*)$ ]]; then
  RAW_TYPE="${BASH_REMATCH[1]}"
  SCOPE="${BASH_REMATCH[2]}"
  DESCRIPTION="${BASH_REMATCH[3]}"
else
  RAW_TYPE=""
  SCOPE=""
  DESCRIPTION="$SUBJECT"
fi

TYPE=$(printf '%s' "$RAW_TYPE" | tr '[:upper:]' '[:lower:]')

if [[ -z "$TYPE" ]] || ! is_valid_type "$TYPE"; then
  echo "commit-msg-normalize: subject must start with one of: $VALID_TYPES" >&2
  echo "commit-msg-normalize: got subject: \"$SUBJECT\"" >&2
  exit 2
fi

if [[ -z "$DESCRIPTION" ]]; then
  echo "commit-msg-normalize: subject has a type but no description" >&2
  exit 2
fi

SUBJECT="${TYPE}${SCOPE}: ${DESCRIPTION}"

if [[ ${#SUBJECT} -gt $SUBJECT_MAX_LEN ]]; then
  echo "commit-msg-normalize: warning: subject line is ${#SUBJECT} chars (guideline: ${SUBJECT_MAX_LEN})" >&2
fi

{
  printf '%s\n' "$SUBJECT"
  if [[ -n "$REST" ]]; then
    printf '%s\n' "$REST"
  fi
} > "$MSG_FILE"

exit 0
