#!/usr/bin/env bash
# pre-commit hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit)
set -euo pipefail

PASS=0
FAIL=0

echo "🪡 pre-commit checks"

run_check() {
  local label="$1"
  shift

  printf "  %-20s" "$label"
  if OUTPUT=$("$@" 2>&1); then
    echo "✓"
    PASS=$((PASS+1))
    return 0
  fi

  echo "✗ FAILED"
  if [[ -n "$OUTPUT" ]]; then
    echo "$OUTPUT" | sed 's/^/    /'
  fi
  FAIL=$((FAIL+1))
}

run_check "gofmt" make fmt-check
run_check "go vet" make vet
run_check "golangci-lint" make lint

echo ""
if [[ $FAIL -gt 0 ]]; then
  echo "❌ $FAIL check(s) failed. Fix issues or use --no-verify to skip."
  exit 1
else
  echo "✅ All checks passed ($PASS)"
fi
