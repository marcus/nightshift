#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks
set -euo pipefail

readonly CONVENTIONAL_TYPES='build|chore|ci|docs|feat|fix|perf|refactor|style|test'
readonly CONVENTIONAL_SCOPE='[[:alnum:]#./_-]+'
readonly CONVENTIONAL_PATTERN="^(${CONVENTIONAL_TYPES})(\\(${CONVENTIONAL_SCOPE}\\))?(!)?: [^[:space:]].*"

usage() {
  echo "Usage: scripts/commit-msg.sh <commit-msg-file> | scripts/commit-msg.sh --title \"<subject>\"" >&2
}

read_subject() {
  if [[ $# -eq 2 && "$1" == "--title" ]]; then
    awk '
      {
        line = $0
        sub(/\r$/, "", line)
        sub(/^[[:space:]]+/, "", line)
        sub(/[[:space:]]+$/, "", line)
        if (line != "" && line !~ /^#/) {
          print line
          exit
        }
      }
    ' <<<"$2"
    return
  fi

  if [[ $# -eq 1 ]]; then
    awk '
      {
        line = $0
        sub(/\r$/, "", line)
        sub(/^[[:space:]]+/, "", line)
        sub(/[[:space:]]+$/, "", line)
        if (line != "" && line !~ /^#/) {
          print line
          exit
        }
      }
    ' "$1"
    return
  fi

  usage
  exit 2
}

print_failure() {
  cat >&2 <<'EOF'
Commit subject must use Conventional Commits:
  type: summary
  type(scope): summary
  type!: summary
  type(scope)!: summary

Accepted types: build, chore, ci, docs, feat, fix, perf, refactor, style, test
Allowed exceptions: Merge ..., Revert ...

Examples:
  feat(run): add pause command
  feat!: drop legacy API
  fix(config): preserve provider YAML keys
  docs(readme): explain hook installation
EOF
}

if [[ $# -eq 1 && ( "$1" == "-h" || "$1" == "--help" ) ]]; then
  usage
  exit 0
fi

subject="$(read_subject "$@")"

if [[ -z "$subject" ]]; then
  echo "Commit subject is empty." >&2
  print_failure
  exit 1
fi

if [[ "$subject" =~ ^(Merge|Revert)\  ]]; then
  exit 0
fi

if [[ "$subject" =~ $CONVENTIONAL_PATTERN ]]; then
  exit 0
fi

echo "Invalid commit subject: $subject" >&2
print_failure
exit 1
