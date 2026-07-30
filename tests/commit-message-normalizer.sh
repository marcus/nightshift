#!/bin/sh

set -u

repository_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
normalizer="$repository_root/scripts/normalize-commit-message.sh"
installer="$repository_root/scripts/install-hooks.sh"
temporary_root=$(mktemp -d "${TMPDIR:-/tmp}/nightshift-commit-normalizer.XXXXXX") || exit 1
tests_run=0
tests_failed=0

cleanup() {
  rm -rf "$temporary_root"
}
trap cleanup EXIT HUP INT TERM

pass() {
  tests_run=$((tests_run + 1))
  printf 'ok %d - %s\n' "$tests_run" "$1"
}

fail() {
  tests_run=$((tests_run + 1))
  tests_failed=$((tests_failed + 1))
  printf 'not ok %d - %s\n' "$tests_run" "$1" >&2
}

assert_files_equal() {
  description=$1
  expected=$2
  actual=$3

  if cmp -s "$expected" "$actual"; then
    pass "$description"
  else
    fail "$description"
    diff -u "$expected" "$actual" >&2 || true
  fi
}

assert_status() {
  description=$1
  expected=$2
  actual=$3

  if [ "$expected" -eq "$actual" ]; then
    pass "$description"
  else
    fail "$description (expected status $expected, got $actual)"
  fi
}

run_normalizer() {
  message_file=$1
  (
    cd "$repository_root" || exit 1
    "$normalizer" "$message_file"
  )
}

valid_message="$temporary_root/valid-message"
valid_expected="$temporary_root/valid-expected"
printf ' FIX (CLI) ! :   Preserve\t API   IDs \n\nBody  spacing stays.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift\n' > "$valid_message"
printf 'fix(cli)!: Preserve API IDs\n\nBody  spacing stays.\n\nNightshift-Task: commit-normalize\nNightshift-Ref: https://github.com/marcus/nightshift\n' > "$valid_expected"
run_normalizer "$valid_message"
assert_files_equal "normalizes metadata capitalization and whitespace while preserving the body and trailers" "$valid_expected" "$valid_message"

idempotent_copy="$temporary_root/idempotent-copy"
cp "$valid_message" "$idempotent_copy"
run_normalizer "$valid_message"
assert_files_equal "normalization is idempotent" "$idempotent_copy" "$valid_message"

no_newline_message="$temporary_root/no-newline-message"
no_newline_expected="$temporary_root/no-newline-expected"
printf 'DOCS: Explain hooks' > "$no_newline_message"
printf 'docs: Explain hooks' > "$no_newline_expected"
run_normalizer "$no_newline_message"
assert_files_equal "preserves a missing final newline" "$no_newline_expected" "$no_newline_message"

supported_types_message="$temporary_root/supported-types-message"
for supported_type in build chore ci docs feat fix perf refactor revert style test
do
  printf '%s: exercise supported type\n' "$supported_type" > "$supported_types_message"
  if run_normalizer "$supported_types_message" >/dev/null 2>&1; then
    supported_type_status=0
  else
    supported_type_status=$?
  fi
  assert_status "accepts supported type: $supported_type" 0 "$supported_type_status"
done

for invalid_subject in \
  'Fix the hook' \
  'release: publish' \
  'feat(bad scope): add hook' \
  'fix:   '
do
  invalid_slug=$(printf '%s' "$invalid_subject" | cksum | awk '{print $1}')
  invalid_message="$temporary_root/invalid-$invalid_slug"
  invalid_copy="$temporary_root/invalid-$invalid_slug.copy"
  printf '%s\n\nBody must remain unchanged.\n' "$invalid_subject" > "$invalid_message"
  cp "$invalid_message" "$invalid_copy"
  if run_normalizer "$invalid_message" >/dev/null 2>&1; then
    invalid_status=0
  else
    invalid_status=$?
  fi
  assert_status "rejects malformed subject: $invalid_subject" 1 "$invalid_status"
  assert_files_equal "leaves rejected message unchanged: $invalid_subject" "$invalid_copy" "$invalid_message"
done

generated_messages="$temporary_root/generated-messages"
mkdir "$generated_messages"
generated_index=0
for generated_subject in \
  "Merge branch 'main'" \
  'Revert "feat: add hook"' \
  'fixup! feat: add hook' \
  'squash! feat: add hook' \
  'amend! feat: add hook' \
  'WIP on main: 0123456 feat: add hook' \
  'On main: save work' \
  'index on main: 0123456 feat: add hook' \
  'untracked files on main: 0123456 feat: add hook' \
  '# This is a combination of 2 commits.'
do
  generated_index=$((generated_index + 1))
  generated_message="$generated_messages/$generated_index"
  generated_copy="$generated_messages/$generated_index.copy"
  printf '%s\n\nGenerated body.\n' "$generated_subject" > "$generated_message"
  cp "$generated_message" "$generated_copy"
  if run_normalizer "$generated_message" >/dev/null 2>&1; then
    generated_status=0
  else
    generated_status=$?
  fi
  assert_status "bypasses generated message: $generated_subject" 0 "$generated_status"
  assert_files_equal "does not rewrite generated message: $generated_subject" "$generated_copy" "$generated_message"
done

comment_repository="$temporary_root/comment-repository"
git init -q "$comment_repository"
comment_message="$comment_repository/message"
comment_copy="$comment_repository/message.copy"

git -C "$comment_repository" config --local core.commentChar ';'
printf '; This is a combination of 3 commits.\n' > "$comment_message"
cp "$comment_message" "$comment_copy"
(
  cd "$comment_repository" || exit 1
  "$normalizer" "$comment_message"
)
assert_files_equal "honors core.commentChar for combined messages" "$comment_copy" "$comment_message"

git -C "$comment_repository" config --local --unset-all core.commentChar
git -C "$comment_repository" config --local core.commentString '//'
printf '// This is a combination of 10 commits.\n' > "$comment_message"
cp "$comment_message" "$comment_copy"
(
  cd "$comment_repository" || exit 1
  "$normalizer" "$comment_message"
)
assert_files_equal "honors core.commentString for combined messages" "$comment_copy" "$comment_message"

git -C "$comment_repository" config --local --unset-all core.commentString
git -C "$comment_repository" config --local core.commentChar auto
printf '; This is a combination of 4 commits.\n' > "$comment_message"
cp "$comment_message" "$comment_copy"
(
  cd "$comment_repository" || exit 1
  "$normalizer" "$comment_message"
)
assert_files_equal "honors an automatically selected Git comment prefix" "$comment_copy" "$comment_message"

git -C "$comment_repository" config --local --unset-all core.commentChar
git -C "$comment_repository" config --local core.commentString '//'
printf '// ordinary invalid comment subject\n' > "$comment_message"
cp "$comment_message" "$comment_copy"
if (
  cd "$comment_repository" || exit 1
  "$normalizer" "$comment_message"
) >/dev/null 2>&1; then
  ordinary_comment_status=0
else
  ordinary_comment_status=$?
fi
assert_status "rejects ordinary comment-led subjects" 1 "$ordinary_comment_status"
assert_files_equal "leaves rejected comment-led subjects unchanged" "$comment_copy" "$comment_message"

hook_repository="$temporary_root/hook-repository"
git init -q "$hook_repository"
mkdir -p "$hook_repository/.githooks" "$hook_repository/scripts"
cp "$repository_root/.githooks/commit-msg" "$hook_repository/.githooks/commit-msg"
cp "$repository_root/.githooks/pre-commit" "$hook_repository/.githooks/pre-commit"
printf '#!/bin/sh\nexit 37\n' > "$hook_repository/scripts/normalize-commit-message.sh"
printf '#!/bin/sh\nexit 38\n' > "$hook_repository/scripts/pre-commit.sh"
chmod +x "$hook_repository/.githooks/commit-msg" "$hook_repository/.githooks/pre-commit"
chmod +x "$hook_repository/scripts/normalize-commit-message.sh" "$hook_repository/scripts/pre-commit.sh"
printf 'invalid\n' > "$hook_repository/message"

if (
  cd "$hook_repository" || exit 1
  ./.githooks/commit-msg message
) >/dev/null 2>&1; then
  commit_hook_status=0
else
  commit_hook_status=$?
fi
assert_status "commit-msg hook propagates normalizer failures" 37 "$commit_hook_status"

if (
  cd "$hook_repository" || exit 1
  ./.githooks/pre-commit
) >/dev/null 2>&1; then
  precommit_hook_status=0
else
  precommit_hook_status=$?
fi
assert_status "pre-commit hook propagates existing check failures" 38 "$precommit_hook_status"

install_repository="$temporary_root/install-repository"
global_config="$temporary_root/global-gitconfig"
git init -q "$install_repository"
mkdir -p "$install_repository/.githooks"
cp "$repository_root/.githooks/commit-msg" "$install_repository/.githooks/commit-msg"
cp "$repository_root/.githooks/pre-commit" "$install_repository/.githooks/pre-commit"
chmod +x "$install_repository/.githooks/commit-msg" "$install_repository/.githooks/pre-commit"
GIT_CONFIG_GLOBAL="$global_config" git config --global core.hooksPath /global/hooks

(
  cd "$install_repository" || exit 1
  GIT_CONFIG_GLOBAL="$global_config" "$installer"
) >/dev/null
installed_path=$(git -C "$install_repository" config --local --get core.hooksPath)
if [ "$installed_path" = ".githooks" ]; then
  pass "installer configures the repository-local shared hook path"
else
  fail "installer configures the repository-local shared hook path"
fi

global_path=$(GIT_CONFIG_GLOBAL="$global_config" git config --global --get core.hooksPath)
if [ "$global_path" = "/global/hooks" ]; then
  pass "installer preserves global Git configuration"
else
  fail "installer preserves global Git configuration"
fi

if (
  cd "$install_repository" || exit 1
  GIT_CONFIG_GLOBAL="$global_config" "$installer"
) >/dev/null; then
  installer_second_status=0
else
  installer_second_status=$?
fi
assert_status "installer is idempotent" 0 "$installer_second_status"

conflict_repository="$temporary_root/conflict-repository"
git init -q "$conflict_repository"
mkdir -p "$conflict_repository/.githooks"
cp "$repository_root/.githooks/commit-msg" "$conflict_repository/.githooks/commit-msg"
cp "$repository_root/.githooks/pre-commit" "$conflict_repository/.githooks/pre-commit"
chmod +x "$conflict_repository/.githooks/commit-msg" "$conflict_repository/.githooks/pre-commit"
git -C "$conflict_repository" config --local core.hooksPath /custom/hooks

if (
  cd "$conflict_repository" || exit 1
  "$installer"
) >/dev/null 2>&1; then
  conflict_status=0
else
  conflict_status=$?
fi
assert_status "installer rejects a conflicting repository-local hook path" 1 "$conflict_status"
conflict_path=$(git -C "$conflict_repository" config --local --get core.hooksPath)
if [ "$conflict_path" = "/custom/hooks" ]; then
  pass "installer leaves a conflicting hook path unchanged"
else
  fail "installer leaves a conflicting hook path unchanged"
fi

printf '1..%d\n' "$tests_run"
if [ "$tests_failed" -ne 0 ]; then
  printf '%d test(s) failed\n' "$tests_failed" >&2
  exit 1
fi
