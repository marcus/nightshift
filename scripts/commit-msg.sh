#!/usr/bin/env bash
# commit-msg hook for Nightshift
# Install: make install-hooks
set -euo pipefail

MESSAGE_FILE=${1:-}
ALLOWED_TYPES="feat|fix|docs|style|refactor|perf|test|build|ci|chore"

fail() {
  local subject=${1:-"<empty>"}

  cat >&2 <<EOF
Invalid commit subject: ${subject}
Expected: type: summary or type(scope): summary
Breaking changes: type!: summary or type(scope)!: summary
Allowed types: ${ALLOWED_TYPES//|/, }
Examples: feat: add task filtering
          fix(scheduler): handle empty queues
EOF
  exit 1
}

if [[ -z "$MESSAGE_FILE" || ! -f "$MESSAGE_FILE" ]]; then
  fail "<missing commit message file>"
fi

subject=""
while IFS= read -r line || [[ -n "$line" ]]; do
  # Git may provide a CRLF-formatted message (for example, from an editor).
  line=${line%$'\r'}

  [[ "$line" =~ ^[[:space:]]*$ ]] && continue
  [[ "$line" =~ ^[[:space:]]*# ]] && continue

  subject=$line
  break
done < "$MESSAGE_FILE"

[[ -n "$subject" ]] || fail

merge_re='^Merge[[:space:]].+$'
revert_re='^Revert[[:space:]]".+"$'
autosquash_re='^(fixup|squash)![[:space:]].+$'

if [[ "$subject" =~ $merge_re || "$subject" =~ $revert_re || "$subject" =~ $autosquash_re ]]; then
  exit 0
fi

subject_re="^(${ALLOWED_TYPES})(\\([a-z0-9][a-z0-9._/-]*\\))?!?: [^[:space:]](.*[^[:space:]])?$"
[[ "$subject" =~ $subject_re ]] || fail "$subject"
