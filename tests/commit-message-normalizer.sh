#!/bin/sh
set -eu

repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
normalizer="$repo_root/scripts/normalize-commit-message.sh"
hook="$repo_root/.githooks/commit-msg"
installer="$repo_root/scripts/install-git-hooks.sh"
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/nightshift-commit-msg-test.XXXXXX")
cleanup() {
  rm -rf "$test_dir"
}
trap cleanup 0
trap 'exit 1' 1 2 15

# Keep the default-prefix fixtures independent of system, user, repository, and
# inherited command-scope Git configuration. Individual tests override this
# command-scope value when exercising core.commentChar or core.commentString.
GIT_CONFIG_NOSYSTEM=1
GIT_CONFIG_GLOBAL="$test_dir/empty-gitconfig"
GIT_CONFIG_COUNT=1
GIT_CONFIG_KEY_0=core.commentChar
GIT_CONFIG_VALUE_0='#'
export GIT_CONFIG_NOSYSTEM GIT_CONFIG_GLOBAL GIT_CONFIG_COUNT
export GIT_CONFIG_KEY_0 GIT_CONFIG_VALUE_0

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

assert_preserved_with_comment_prefix() {
  name=$1
  comment_key=$2
  comment_prefix=$3
  input=$4
  message="$test_dir/message"
  expected_file="$test_dir/expected"

  printf '%b' "$input" > "$message"
  printf '%b' "$input" > "$expected_file"
  GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0="core.$comment_key" \
    GIT_CONFIG_VALUE_0="$comment_prefix" \
    "$normalizer" "$message" || fail "$name (normalizer failed)"
  cmp -s "$expected_file" "$message" || {
    diff -u "$expected_file" "$message" >&2 || true
    fail "$name"
  }
  tests_run=$((tests_run + 1))
  echo "ok - $name"
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

assert_rejected_with_comment_prefix() {
  name=$1
  comment_key=$2
  comment_prefix=$3
  input=$4
  message="$test_dir/message"
  original="$test_dir/original"

  printf '%b' "$input" > "$message"
  cp "$message" "$original"
  if GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0="core.$comment_key" \
    GIT_CONFIG_VALUE_0="$comment_prefix" \
    "$normalizer" "$message" >/dev/null 2>&1; then
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

assert_normalizes \
  "Git comments after subject" \
  ' CHORE :   normalize messages

# Please enter the commit message for your changes.
# Lines starting with '\''#'\'' will be ignored.
' \
  'chore: normalize messages

# Please enter the commit message for your changes.
# Lines starting with '\''#'\'' will be ignored.
'

assert_rejected "blank input" '
\040\040\040
'
assert_rejected "unsupported type" 'release: prepare 1.0
'
assert_rejected "missing separator" 'feat add status output
'
assert_rejected "empty scope" 'feat(): add status output
'
assert_rejected "missing summary" 'feat(cli):
'
assert_rejected "manual comment subject" '# invalid manual subject
'
assert_rejected "lookalike generated comment subject" '# This is a combination of banana commits.
'
assert_rejected_with_comment_prefix \
  "manual subject with configured comment string" \
  "commentString" \
  "//" \
  '// invalid manual subject
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
assert_preserved_with_comment_prefix \
  "generated message with configured comment character" \
  "commentChar" \
  ";" \
  '; This is a combination of 2 commits.
; This is the 1st commit message:

feat: first message\040\040\040
'
assert_preserved_with_comment_prefix \
  "generated message with configured comment string" \
  "commentString" \
  "//" \
  '// This is a combination of 2 commits.
// This is the 1st commit message:

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

installer_repo="$test_dir/installer-repo"
mkdir -p "$installer_repo/scripts" "$installer_repo/.githooks"
cp "$installer" "$installer_repo/scripts/install-git-hooks.sh"
git -C "$installer_repo" init -q
global_config="$test_dir/global-gitconfig"
GIT_CONFIG_GLOBAL="$global_config" git config --global core.hooksPath /user/hooks
GIT_CONFIG_GLOBAL="$global_config" "$installer_repo/scripts/install-git-hooks.sh" >/dev/null ||
  fail "hook installer"
installed_path=$(GIT_CONFIG_GLOBAL="$global_config" git -C "$installer_repo" config --local --get core.hooksPath)
[ "$installed_path" = ".githooks" ] || fail "hook installer writes local config"
global_path=$(GIT_CONFIG_GLOBAL="$global_config" git config --global --get core.hooksPath)
[ "$global_path" = "/user/hooks" ] || fail "hook installer preserves user config"
GIT_CONFIG_GLOBAL="$global_config" "$installer_repo/scripts/install-git-hooks.sh" >/dev/null ||
  fail "hook installer idempotence"
tests_run=$((tests_run + 4))
echo "ok - hook installer"
echo "ok - hook installer writes local config"
echo "ok - hook installer preserves user config"
echo "ok - hook installer idempotence"

conflict_repo="$test_dir/conflict-repo"
mkdir -p "$conflict_repo/scripts" "$conflict_repo/.githooks"
cp "$installer" "$conflict_repo/scripts/install-git-hooks.sh"
git -C "$conflict_repo" init -q
git -C "$conflict_repo" config --local core.hooksPath .custom-hooks
if "$conflict_repo/scripts/install-git-hooks.sh" >/dev/null 2>&1; then
  fail "hook installer rejects conflicting local config"
fi
conflict_path=$(git -C "$conflict_repo" config --local --get core.hooksPath)
[ "$conflict_path" = ".custom-hooks" ] || fail "hook installer preserves conflicting local config"
tests_run=$((tests_run + 2))
echo "ok - hook installer rejects conflicting local config"
echo "ok - hook installer preserves conflicting local config"

echo "$tests_run tests passed"
