#!/bin/sh
# setup-hooks.sh — configure local git to use repo-tracked hooks.
#
# Sets `core.hooksPath` to .githooks and `commit.template` to .gitmessage so
# that the commit-msg normalizer runs and contributors see the template when
# they `git commit` without `-m`.

set -eu

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "setup-hooks: not inside a git repository" >&2
    exit 1
}

cd "$REPO_ROOT"

chmod +x .githooks/commit-msg scripts/commit-normalizer.sh 2>/dev/null || true

git config core.hooksPath .githooks
git config commit.template .gitmessage

cat <<EOF
✓ core.hooksPath  = .githooks
✓ commit.template = .gitmessage

The commit-msg hook will normalize messages to Conventional Commits.
See docs/commit-conventions.md for the full standard.

To bypass once:   COMMIT_NORMALIZER=0 git commit -m "..."
To disable:       git config --unset core.hooksPath
EOF
