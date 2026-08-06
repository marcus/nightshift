#!/usr/bin/env bash
# pre-commit hook for nightshift
# Install: make install-hooks
set -euo pipefail

PASS=0
FAIL=0

echo "🪡 pre-commit checks"

# --- gofmt: only staged .go files ---
STAGED_GO=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [[ -n "$STAGED_GO" ]]; then
  printf "  %-20s" "gofmt"
  UNFORMATTED=$(echo "$STAGED_GO" | xargs gofmt -l 2>&1)
  if [[ -z "$UNFORMATTED" ]]; then
    echo "✓"
    PASS=$((PASS+1))
  else
    echo "✗ FAILED — run: gofmt -w ."
    echo "$UNFORMATTED" | sed 's/^/    /'
    FAIL=$((FAIL+1))
  fi
else
  printf "  %-20s" "gofmt"
  echo "– (no .go files staged)"
fi

# --- go vet ---
printf "  %-20s" "go vet"
VET_OUT=$(go vet ./... 2>&1)
if [[ $? -eq 0 ]]; then
  echo "✓"
  PASS=$((PASS+1))
else
  echo "✗ FAILED"
  echo "$VET_OUT" | sed 's/^/    /'
  FAIL=$((FAIL+1))
fi

# --- go build ---
printf "  %-20s" "go build"
BUILD_OUT=$(go build ./... 2>&1)
if [[ $? -eq 0 ]]; then
  echo "✓"
  PASS=$((PASS+1))
else
  echo "✗ FAILED"
  echo "$BUILD_OUT" | sed 's/^/    /'
  FAIL=$((FAIL+1))
fi

echo ""
if [[ $FAIL -gt 0 ]]; then
  echo "❌ $FAIL check(s) failed. Fix issues or use --no-verify to skip."
  exit 1
else
  echo "✅ All checks passed ($PASS)"
fi
