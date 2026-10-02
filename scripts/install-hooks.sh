#!/usr/bin/env bash
# Idempotently installs nightshift's git hooks by symlinking them into
# .git/hooks/. Safe to re-run.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"

mkdir -p "$HOOKS_DIR"

# install_hook <hook-name> <script-file>
# hook-name is the name git expects in .git/hooks (no extension).
install_hook() {
  local hook_name="$1"
  local script_file="$2"

  chmod +x "$REPO_ROOT/scripts/$script_file"
  ln -sf "../../scripts/$script_file" "$HOOKS_DIR/$hook_name"
  echo "✓ ${hook_name} hook installed (.git/hooks/${hook_name} → scripts/${script_file})"
}

install_hook "pre-commit" "pre-commit.sh"
install_hook "commit-msg" "commit-msg.sh"

echo "Done. Bypass any hook in an emergency with: git commit --no-verify"
