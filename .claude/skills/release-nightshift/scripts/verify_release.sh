#!/usr/bin/env bash
# Verify a nightshift release completed successfully.
# Usage: verify_release.sh vX.Y.Z
set -euo pipefail

TAG="${1:?Usage: verify_release.sh vX.Y.Z}"
REPO="marcus/nightshift"
EXPECTED_ARTIFACTS=("darwin_amd64" "darwin_arm64" "linux_amd64" "linux_arm64" "checksums.txt")

echo "=== Verifying release ${TAG} ==="

# 1. Check the release exists
echo "Checking release exists..."
if ! gh release view "$TAG" --repo "$REPO" > /dev/null 2>&1; then
  echo "FAIL: Release ${TAG} not found"
  exit 1
fi
echo "OK: Release exists"

# 2. Check all expected artifacts are present
echo "Checking artifacts..."
ASSETS=$(gh release view "$TAG" --repo "$REPO" --json assets -q '.assets[].name')
MISSING=0
for PATTERN in "${EXPECTED_ARTIFACTS[@]}"; do
  if ! echo "$ASSETS" | grep -q "$PATTERN"; then
    echo "FAIL: Missing artifact matching '${PATTERN}'"
    MISSING=1
  fi
done
if [ "$MISSING" -eq 0 ]; then
  echo "OK: All expected artifacts present"
fi

# 3. Check the most recent release workflow run
echo "Checking Actions workflow..."
RUN_STATUS=$(gh run list --workflow=release.yml --repo "$REPO" --limit 1 --json conclusion -q '.[0].conclusion')
if [ "$RUN_STATUS" = "success" ]; then
  echo "OK: Release workflow succeeded"
else
  echo "WARN: Latest release workflow status: ${RUN_STATUS:-unknown}"
fi

# 4. Summary
echo ""
if [ "$MISSING" -eq 0 ] && [ "$RUN_STATUS" = "success" ]; then
  echo "Release ${TAG} verified successfully."
else
  echo "Release ${TAG} has issues — review above."
  exit 1
fi
