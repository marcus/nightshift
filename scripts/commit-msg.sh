#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks
set -euo pipefail

msg_file=${1:-}

if [[ -z "$msg_file" || ! -f "$msg_file" ]]; then
  echo "commit-msg: missing commit message file" >&2
  exit 1
fi

normalize_line() {
  local line=$1
  line=${line%$'\r'}
  printf '%s' "$line"
}

subject=
while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line=$(normalize_line "$raw_line")
  [[ -z "$line" ]] && continue
  [[ $line == \#* ]] && continue
  subject=$line
  break
done <"$msg_file"

if [[ -z "$subject" ]]; then
  cat >&2 <<'EOF'
commit-msg: empty commit message
Use: type: summary
EOF
  exit 1
fi

case "$subject" in
  Merge\ *|Revert\ *|fixup!\ *|squash!\ *)
    exit 0
    ;;
esac

pattern='^(build|chore|ci|docs|feat|fix|perf|refactor|style|test)(\([a-z0-9#][a-z0-9._/#-]*\))?(!)?: [^[:space:]](.*[^[:space:]])?$'

if [[ $subject =~ $pattern ]]; then
  exit 0
fi

cat >&2 <<EOF
commit-msg: invalid subject
Expected: type: summary
Example: fix: normalize commit message hook install
Allowed: optional scope like feat(tasks): ..., plus Merge/Revert/fixup!/squash! subjects
Bypass: git commit --no-verify
Found: $subject
EOF
exit 1
