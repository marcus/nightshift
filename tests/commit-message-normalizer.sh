#!/bin/sh
set -eu

repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
normalizer="$repo_root/scripts/normalize-commit-message.sh"
hook="$repo_root/.githooks/commit-msg"
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/nightshift-commit-msg-test.XXXXXX")
cleanup() {
  rm -rf "$test_dir"
}
trap cleanup 0
trap 'exit 1' 1 2 15

tests_run=0

fail() {
  echo "not ok - $1" >&2
  exit 1
}

assert_normalizes() {
  name=$1
  input=$2
  expected=$3
  message="$test_dir/message"
  expected_file="$test_dir/expected"

  printf '%b' "$input" > "$message"
  printf '%b' "$expected" > "$expected_file"
  "$normalizer" "$message" || fail "$name (normalizer failed)"
  cmp -s "$expected_file" "$message" || {
    diff -u "$expected_file" "$message" >&2 || true
    fail "$name"
  }
  tests_run=$((tests_run + 1))
  echo "ok - $name"
}

assert_preserved() {
  name=$1
  input=$2
  assert_normalizes "$name" "$input" "$input"
}

assert_rejected() {
  name=$1
  input=$2
  message="$test_dir/message"
  original="$test_dir/original"

  printf '%b' "$input" > "$message"
  cp "$message" "$original"
  if "$normalizer" "$message" >/dev/null 2>&1; then
    fail "$name (unexpected success)"
  fi
  cmp -s "$original" "$message" || fail "$name (message changed)"
  tests_run=$((tests_run + 1))
  echo "ok - $name"
}

assert_normalizes \
  "already-valid subject" \
  'feat(cli): add status output
' \
  'feat(cli): add status output
'

assert_normalizes \
  "capitalization and spacing" \
  '
  FEAT  ( cli ) ! :   add   status output\040\040\040

' \
  'feat(cli)!: add status output
'

assert_normalizes \
  "scoped breaking change" \
  'FIX (config)!:   reject unknown keys
' \
  'fix(config)!: reject unknown keys
'

assert_normalizes \
  "body paragraphs and trailers" \
  'DOCS :   explain   provider setup\040\040\040

Keep  repeated  body spacing.\040\040\040

Nightshift-Task: commit-normalize\040\040\040
Nightshift-Ref: https://github.com/marcus/nightshift

' \
  'docs: explain provider setup

Keep  repeated  body spacing.

Nightshift-Task: commit-normalize
Nightshift-Ref: https://github.com/marcus/nightshift
'

assert_rejected "blank input" '
\040\040\040
'
assert_rejected "unsupported type" 'release: prepare 1.0
'

assert_preserved "merge message" 'Merge branch '\''main'\'' into feature\040\040\040

Generated merge body.\040\040\040
'
assert_preserved "revert message" 'Revert "feat: remove legacy API"

This reverts commit abc123.\040\040\040
'
assert_preserved "fixup commit" 'fixup! feat(cli): add status output\040\040\040
'
assert_preserved "squash commit" 'squash! fix(config): reject unknown keys\040\040\040
'
assert_preserved "generated rebase message" '# This is a combination of 2 commits.
# This is the 1st commit message:

feat: first message\040\040\040
'

message="$test_dir/message"
printf '%b' ' CHORE :   normalize messages\040
' > "$message"
"$hook" "$message" || fail "commit-msg hook"
printf '%s' 'chore: normalize messages
' > "$test_dir/expected"
cmp -s "$test_dir/expected" "$message" || fail "commit-msg hook"
if "$hook" "$test_dir/missing" >/dev/null 2>&1; then
  fail "commit-msg hook propagates failures"
fi
tests_run=$((tests_run + 2))
echo "ok - commit-msg hook"
echo "ok - commit-msg hook propagates failures"

cp "$message" "$test_dir/once"
"$normalizer" "$message" || fail "idempotence (second run failed)"
cmp -s "$test_dir/once" "$message" || fail "idempotence"
tests_run=$((tests_run + 1))
echo "ok - idempotence"

echo "$tests_run tests passed"
