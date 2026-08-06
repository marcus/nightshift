#!/bin/sh
# run-tests.sh — fixture-driven tests for scripts/commit-normalizer.sh.
#
# Each case directory under tests/commit-normalizer/cases/ may contain:
#   input          — the commit message to feed the normalizer
#   expected       — the expected normalized output (if exit is 0)
#   exit           — expected exit code (default 0)
#   expected.err   — substring expected on stderr (optional)

set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
NORMALIZER="$REPO_ROOT/scripts/commit-normalizer.sh"
CASES_DIR="$SCRIPT_DIR/cases"

[ -x "$NORMALIZER" ] || chmod +x "$NORMALIZER"

PASS=0
FAIL=0
FAIL_NAMES=""

for case_dir in "$CASES_DIR"/*/; do
    name=$(basename "$case_dir")
    input="$case_dir/input"
    [ -f "$input" ] || { echo "skip: $name (no input)"; continue; }

    expected_exit=0
    [ -f "$case_dir/exit" ] && expected_exit=$(cat "$case_dir/exit")

    work=$(mktemp "${TMPDIR:-/tmp}/cn-test.XXXXXX")
    err=$(mktemp "${TMPDIR:-/tmp}/cn-test-err.XXXXXX")
    cp "$input" "$work"

    set +e
    "$NORMALIZER" "$work" 2>"$err"
    got_exit=$?
    set -e

    ok=1
    msg=""

    if [ "$got_exit" -ne "$expected_exit" ]; then
        ok=0
        msg="exit $got_exit, expected $expected_exit"
    fi

    if [ "$ok" -eq 1 ] && [ "$expected_exit" -eq 0 ] && [ -f "$case_dir/expected" ]; then
        if ! diff -u "$case_dir/expected" "$work" >/dev/null; then
            ok=0
            msg="output mismatch"
            DIFF=$(diff -u "$case_dir/expected" "$work" || true)
        fi
    fi

    if [ "$ok" -eq 1 ] && [ -f "$case_dir/expected.err" ]; then
        needle=$(cat "$case_dir/expected.err")
        if ! grep -qF "$needle" "$err"; then
            ok=0
            msg="stderr missing: $needle"
        fi
    fi

    if [ "$ok" -eq 1 ]; then
        printf "  ✓ %s\n" "$name"
        PASS=$((PASS + 1))
    else
        printf "  ✗ %s — %s\n" "$name" "$msg"
        if [ -n "${DIFF:-}" ]; then
            printf '%s\n' "$DIFF" | sed 's/^/      /'
            DIFF=""
        fi
        if [ -s "$err" ]; then
            printf "    stderr:\n"
            sed 's/^/      /' "$err"
        fi
        FAIL=$((FAIL + 1))
        FAIL_NAMES="$FAIL_NAMES $name"
    fi

    rm -f "$work" "$err"
done

echo
echo "passed: $PASS  failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then
    echo "failures:$FAIL_NAMES"
    exit 1
fi
