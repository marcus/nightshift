#!/usr/bin/env bash
# pre-commit hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit)
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

PASS=0
FAIL=0

echo "🪡 pre-commit checks"

run_check() {
  local target="$1"
  local output

  printf "  %-20s" "$target"
  if output=$(make "$target" 2>&1); then
    echo "✓"
    PASS=$((PASS+1))
    return 0
  fi

  echo "✗ FAILED"
  if [[ -n "$output" ]]; then
    echo "$output" | sed 's/^/    /'
  fi
  FAIL=$((FAIL+1))
  return 0
}

run_check "fmt-check"
run_check "vet"
run_check "lint"

echo ""
if [[ $FAIL -gt 0 ]]; then
  echo "❌ $FAIL check(s) failed. Fix issues or use --no-verify to skip."
  exit 1
else
  echo "✅ All checks passed ($PASS)"
fi
