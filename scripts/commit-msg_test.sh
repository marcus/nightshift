#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
HOOK="$SCRIPT_DIR/commit-msg.sh"
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

pass=0
fail=0

run_case() {
  local expectation=$1
  local name=$2
  local message=$3
  local message_file="$TEST_DIR/message"

  printf '%s' "$message" > "$message_file"

  if "$HOOK" "$message_file" >/dev/null 2>&1; then
    actual=valid
  else
    actual=invalid
  fi

  if [[ "$actual" == "$expectation" ]]; then
    printf 'ok - %s\n' "$name"
    pass=$((pass + 1))
  else
    printf 'not ok - %s (expected %s, got %s)\n' "$name" "$expectation" "$actual"
    fail=$((fail + 1))
  fi
}

run_case valid "simple subject" 'feat: add task filtering'
run_case valid "scoped subject" 'fix(scheduler): handle empty queues'
run_case valid "breaking change without scope" 'feat!: replace task configuration'
run_case valid "breaking change with scope" 'refactor(config)!: remove legacy keys'
run_case valid "body and Nightshift trailers" $'docs: explain task selection\n\nMore detail.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift\n'
run_case valid "leading comments and blank lines" $'# Please enter a commit message.\n\nchore: refresh generated files\n'
run_case valid "CRLF message" $'test(parser): cover invalid input\r\n\r\nBody.\r\n'
run_case valid "generated merge subject" "Merge branch 'main' into feature"
run_case valid "generated revert subject" 'Revert "feat: add task filtering"'
run_case valid "generated fixup subject" 'fixup! feat: add task filtering'
run_case valid "generated squash subject" 'squash! fix(scheduler): handle empty queues'

run_case invalid "empty message" ''
run_case invalid "comments-only message" $'# Please enter a commit message.\n# Lines starting with # are ignored.\n'
run_case invalid "unknown type" 'feature: add task filtering'
run_case invalid "uppercase type" 'Feat: add task filtering'
run_case invalid "scope with whitespace" 'fix(task queue): handle empty queues'
run_case invalid "missing summary" 'fix:'
run_case invalid "missing space after colon" 'fix:no separating space'
run_case invalid "extra space after colon" 'fix:  too much space'
run_case invalid "leading whitespace" ' fix: leading whitespace'
run_case invalid "trailing whitespace" 'fix: trailing whitespace '
run_case invalid "malformed breaking marker" 'fix(!): malformed marker'

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]]
