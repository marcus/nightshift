#!/usr/bin/env bash
# commit-msg hook for nightshift
# Enforces Conventional Commits subject lines: https://www.conventionalcommits.org/
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

MSG_FILE="$1"
SUBJECT=$(head -n1 "$MSG_FILE")

# Allow merge and revert commits through untouched.
if [[ "$SUBJECT" =~ ^Merge\  ]] || [[ "$SUBJECT" =~ ^Revert\  ]]; then
  exit 0
fi

TYPES="feat|fix|chore|docs|refactor|perf|ci|test|build|style"
PATTERN="^(${TYPES})(\([a-zA-Z0-9_.-]+\))?!?: .+"

printf "🪡 commit-msg check\n"
printf "  %-20s" "conventional commit"

if [[ "$SUBJECT" =~ $PATTERN ]]; then
  echo "✓"
else
  echo "✗ FAILED"
  echo "    Subject must match: <type>(<scope>)?: <description>"
  echo "    Allowed types: ${TYPES//|/, }"
  echo "    Got: \"$SUBJECT\""
  echo ""
  echo "❌ commit message does not follow Conventional Commits. Fix it or use --no-verify to skip."
  exit 1
fi
