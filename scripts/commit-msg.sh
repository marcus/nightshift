#!/usr/bin/env bash
# commit-msg hook and CI validator for Nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

readonly ALLOWED_TYPES='build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test'
readonly CONVENTIONAL_PATTERN="^(${ALLOWED_TYPES})(\\([a-z0-9][a-z0-9._/-]*\\))?(!)?: [^ ].*$"
readonly MERGE_PATTERN='^Merge '
readonly REVERT_PATTERN='^Revert ".+"$'
readonly VERSION_BUMP_PATTERN='^Bump version to v[0-9]+(\.[0-9]+)+([.-][0-9A-Za-z.-]+)?$'
readonly RELEASE_PATTERN='^Release v[0-9]+(\.[0-9]+)+([.-][0-9A-Za-z.-]+)?(: .+)?$'

usage() {
  cat <<'EOF'
Usage:
  scripts/commit-msg.sh <commit-message-file>
  scripts/commit-msg.sh --rev-range <git-revision-or-range>

Expected subject format:
  type(scope?): summary

Allowed types:
  build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test
EOF
}

print_error() {
  local source="$1"
  local subject="$2"

  {
    echo "Invalid commit subject in ${source}:"
    echo "  ${subject}"
    echo ""
    echo "Expected first line to match:"
    echo "  type(scope?): summary"
    echo ""
    echo "Allowed types:"
    echo "  build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test"
    echo ""
    echo "Examples:"
    echo "  feat(cli): add commit message validator"
    echo "  fix: handle pull request commit ranges in CI"
    echo "  docs(readme): document local hook installation"
    echo ""
    echo "Allowed exceptions:"
    echo "  Merge pull request #123 from owner/branch"
    echo "  Revert \"feat(cli): add commit message validator\""
    echo "  Bump version to v0.3.4"
    echo "  Release v0.3.4: changelog and binaries"
  } >&2
}

is_valid_subject() {
  local subject="$1"

  [[ "$subject" =~ $CONVENTIONAL_PATTERN ]] && return 0
  [[ "$subject" =~ $MERGE_PATTERN ]] && return 0
  [[ "$subject" =~ $REVERT_PATTERN ]] && return 0
  [[ "$subject" =~ $VERSION_BUMP_PATTERN ]] && return 0
  [[ "$subject" =~ $RELEASE_PATTERN ]] && return 0
  return 1
}

validate_subject() {
  local source="$1"
  local subject="$2"

  if ! is_valid_subject "$subject"; then
    print_error "$source" "$subject"
    return 1
  fi
}

validate_message_file() {
  local message_file="$1"
  local subject

  if [[ ! -f "$message_file" ]]; then
    echo "Commit message file not found: $message_file" >&2
    return 1
  fi

  subject=$(sed -n '1{s/[[:space:]]*$//;p;}' "$message_file")
  validate_subject "$message_file" "$subject"
}

validate_rev_range() {
  local rev_spec="$1"
  local commits=()
  local rev
  local subject

  if [[ "$rev_spec" == *..* ]]; then
    while IFS= read -r rev; do
      commits+=("$rev")
    done < <(git rev-list --reverse "$rev_spec")
  else
    commits=("$rev_spec")
  fi

  if [[ ${#commits[@]} -eq 0 ]]; then
    echo "No commits found for revision range: $rev_spec" >&2
    return 1
  fi

  for rev in "${commits[@]}"; do
    subject=$(git log -1 --format=%s "$rev")
    validate_subject "$rev" "$subject"
    printf '✓ %s %s\n' "${rev:0:7}" "$subject"
  done
}

main() {
  case "${1-}" in
    "")
      usage >&2
      exit 1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --rev-range)
      if [[ $# -ne 2 ]]; then
        usage >&2
        exit 1
      fi
      validate_rev_range "$2"
      ;;
    *)
      if [[ $# -ne 1 ]]; then
        usage >&2
        exit 1
      fi
      validate_message_file "$1"
      ;;
  esac
}

main "$@"
