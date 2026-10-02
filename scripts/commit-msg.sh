#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks  (or: ln -sf ../../scripts/commit-msg.sh .git/hooks/commit-msg)
set -euo pipefail

msg_file=${1:?commit message file required}
subject=""

IFS= read -r subject < "$msg_file" || true
subject=${subject%$'\r'}

case "$subject" in
  Merge\ *|Revert\ *|fixup!\ *|squash!\ *)
    exit 0
    ;;
esac

pattern='^(feat|fix|docs|refactor|test|build|chore|ci|perf)(\([a-z0-9._/-]+\))?(!)?: .+'

if [[ "$subject" =~ $pattern ]]; then
  exit 0
fi

if [[ -n "$subject" ]]; then
  echo "invalid commit subject: $subject" >&2
else
  echo "invalid commit subject: <empty>" >&2
fi

cat >&2 <<'EOF'
use: type(scope): summary
or:  type: summary
types: feat fix docs refactor test build chore ci perf
examples:
  feat(tasks): add commit message validator
  docs: document commit hooks
  build(release): bump version to v0.3.5
allowed: Merge..., Revert..., fixup! ..., squash! ...
EOF

exit 1
