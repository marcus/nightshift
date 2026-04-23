#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "commit-msg hook expects the commit message file path" >&2
  exit 1
fi

MESSAGE_FILE=$1
ALLOWED_TYPES="feat|fix|docs|refactor|test|build|ci|chore"
HEADER_PATTERN='^([[:alpha:]]+)(\([^)]+\))?:[[:space:]]+(.+)$'
VALID_PATTERN="^(${ALLOWED_TYPES})(\\([^)]+\\))?: .+$"

trim() {
  sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//'
}

normalize_subject() {
  local subject=$1
  local normalized
  normalized=$(printf '%s' "$subject" | trim | sed -E 's/\.$//')

  if [[ $normalized =~ $HEADER_PATTERN ]]; then
    local raw_type=${BASH_REMATCH[1]}
    local scope=${BASH_REMATCH[2]:-}
    local summary=${BASH_REMATCH[3]}
    local lower_type
    lower_type=$(printf '%s' "$raw_type" | tr '[:upper:]' '[:lower:]')

    case "$lower_type" in
      feat|fix|docs|refactor|test|build|ci|chore)
        normalized="${lower_type}${scope}: ${summary}"
        ;;
    esac
  fi

  printf '%s' "$normalized"
}

rewrite_first_line() {
  local subject=$1
  local tmp
  tmp=$(mktemp)
  awk -v subject="$subject" 'NR == 1 { $0 = subject } { print }' "$MESSAGE_FILE" > "$tmp"
  mv "$tmp" "$MESSAGE_FILE"
}

subject=$(sed -n '1p' "$MESSAGE_FILE")
trimmed_subject=$(printf '%s' "$subject" | trim)

case "$trimmed_subject" in
  Merge\ *|Revert\ *|fixup!\ *|squash!\ *)
    exit 0
    ;;
esac

normalized_subject=$(normalize_subject "$subject")

if [[ "$normalized_subject" != "$subject" ]]; then
  rewrite_first_line "$normalized_subject"
fi

if [[ $normalized_subject =~ $VALID_PATTERN ]]; then
  exit 0
fi

cat >&2 <<'EOF'
commit subject must match: type: subject
or: type(scope): subject
allowed types: feat, fix, docs, refactor, test, build, ci, chore
examples:
  feat: add pause command
  fix(budget): handle weekly reset
use --no-verify to bypass
EOF
exit 1
