#!/usr/bin/env bash
# Plain-bash test suite for scripts/commit-msg-normalize.sh (bats is not
# assumed to be installed). Run with: bash scripts/test-commit-msg-normalize.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NORMALIZE="$SCRIPT_DIR/commit-msg-normalize.sh"
HOOK="$SCRIPT_DIR/commit-msg.sh"

PASS=0
FAIL=0

tmpfile() {
  mktemp "${TMPDIR:-/tmp}/commit-msg-test.XXXXXX"
}

assert_eq() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    echo "  ✓ $desc"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $desc"
    echo "    expected: $(printf '%q' "$expected")"
    echo "    actual:   $(printf '%q' "$actual")"
    FAIL=$((FAIL + 1))
  fi
}

assert_status() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    echo "  ✓ $desc"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $desc (expected exit $expected, got $actual)"
    FAIL=$((FAIL + 1))
  fi
}

echo "already-conventional message passes unchanged"
f=$(tmpfile)
printf 'feat: add widget support\n' > "$f"
"$NORMALIZE" "$f" >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
assert_eq "content unchanged" "feat: add widget support" "$(cat "$f")"
rm -f "$f"

echo "missing colon after type gets one inserted"
f=$(tmpfile)
printf 'fix add missing colon\n' > "$f"
"$NORMALIZE" "$f" >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
assert_eq "colon inserted" "fix: add missing colon" "$(cat "$f")"
rm -f "$f"

echo "unknown type is rejected with non-zero exit"
f=$(tmpfile)
printf 'oops: this is not a real type\n' > "$f"
"$NORMALIZE" "$f" >/dev/null 2>&1
status=$?
assert_status "exits non-zero" 1 "$([[ $status -ne 0 ]] && echo 1 || echo 0)"
rm -f "$f"

echo "trailing whitespace and extra blank lines are stripped"
f=$(tmpfile)
printf 'chore: tidy up   \n\n\n\nbody line one\n\n\n\nbody line two\n' > "$f"
"$NORMALIZE" "$f" >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
assert_eq "collapsed blanks and trimmed" \
  "chore: tidy up

body line one

body line two" \
  "$(cat "$f")"
rm -f "$f"

echo "message with a body preserves the body"
f=$(tmpfile)
printf 'docs: update readme\n\nExplain the new install flow in more detail.\n' > "$f"
"$NORMALIZE" "$f" >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
assert_eq "body preserved" \
  "docs: update readme

Explain the new install flow in more detail." \
  "$(cat "$f")"
rm -f "$f"

echo "hook exempts merge commits (source arg) from type-prefix validation"
f=$(tmpfile)
printf "Merge branch 'main' into feat/x\n" > "$f"
"$HOOK" "$f" merge >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
rm -f "$f"

echo "hook exempts revert commits (subject sniff) from type-prefix validation"
f=$(tmpfile)
printf 'Revert "feat: add widget support"\n' > "$f"
"$HOOK" "$f" >/dev/null 2>&1
status=$?
assert_status "exits 0" 0 "$status"
rm -f "$f"

echo "hook still rejects a non-merge message with an unknown type"
f=$(tmpfile)
printf 'oops: this is not a real type\n' > "$f"
"$HOOK" "$f" >/dev/null 2>&1
status=$?
assert_status "exits non-zero" 1 "$([[ $status -ne 0 ]] && echo 1 || echo 0)"
rm -f "$f"

echo ""
echo "$PASS passed, $FAIL failed"
[[ $FAIL -eq 0 ]]
