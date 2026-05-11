#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

MSG_FILE=${1:-}
if [[ -z "$MSG_FILE" || ! -f "$MSG_FILE" ]]; then
  echo "commit-msg: missing commit message file" >&2
  exit 1
fi

TYPES="feat|fix|docs|test|refactor|chore|build|ci|perf|style|revert"
SUBJECT_MAX=72

trim() {
  local value=$1
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

subject_line_number=$(
  awk '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*$/ { next }
    { print NR; exit }
  ' "$MSG_FILE"
)

if [[ -z "$subject_line_number" ]]; then
  echo "commit-msg: empty commit message" >&2
  exit 1
fi

subject=$(
  awk -v line="$subject_line_number" 'NR == line { print; exit }' "$MSG_FILE"
)
subject=$(trim "$subject")

allow_special=false
case "$subject" in
  Merge\ *|Revert\ *|fixup!\ *|squash!\ *)
    allow_special=true
    ;;
esac

normalized=$subject
if [[ "$allow_special" == false ]]; then
  if [[ "$normalized" =~ ^([A-Za-z]+)(\([A-Za-z0-9._/-]+\))?[[:space:]]*[-:][[:space:]]*(.+)$ ]]; then
    type_part=$(printf '%s' "${BASH_REMATCH[1]}" | tr '[:upper:]' '[:lower:]')
    scope_part=${BASH_REMATCH[2]:-}
    summary_part=$(trim "${BASH_REMATCH[3]}")
    normalized="${type_part}${scope_part}: ${summary_part}"
  fi
fi

if [[ "$normalized" != "$subject" ]]; then
  tmp=$(mktemp)
  awk -v line="$subject_line_number" -v replacement="$normalized" '
    NR == line { print replacement; next }
    { print }
  ' "$MSG_FILE" > "$tmp"
  cat "$tmp" > "$MSG_FILE"
  rm -f "$tmp"
  subject=$normalized
fi

if [[ "$allow_special" == true ]]; then
  exit 0
fi

if ! [[ "$subject" =~ ^($TYPES)(\([A-Za-z0-9._/-]+\))?:[[:space:]][^[:space:]].*$ ]]; then
  cat >&2 <<'EOF'
commit-msg: expected commit message format:
  type: summary
  type(scope): summary

Accepted types:
  feat, fix, docs, test, refactor, chore, build, ci, perf, style, revert

Examples:
  feat: add project setup command
  fix(config): preserve default provider
  docs: document hook installation

Merge, Revert, fixup!, and squash! commits are allowed.
EOF
  exit 1
fi

summary=${subject#*: }
if [[ -z "$(trim "$summary")" ]]; then
  echo "commit-msg: summary must not be empty" >&2
  exit 1
fi

if (( ${#subject} > SUBJECT_MAX )); then
  echo "commit-msg: subject must be ${SUBJECT_MAX} characters or fewer" >&2
  echo "  ${subject}" >&2
  exit 1
fi
