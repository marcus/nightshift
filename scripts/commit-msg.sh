#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks
set -euo pipefail

msg_file="${1:-}"
if [[ -z "${msg_file}" || ! -f "${msg_file}" ]]; then
  echo "commit-msg: expected path to commit message file" >&2
  exit 1
fi

subject=$(
  grep -vE '^[[:space:]]*#' "$msg_file" | awk 'NF { print; exit }'
)

if [[ -z "${subject}" ]]; then
  exit 0
fi

case "${subject}" in
  Merge\ *|Revert\ *|fixup!\ *|squash!\ *)
    exit 0
    ;;
esac

pattern='^(build|chore|ci|docs|feat|fix|perf|refactor|release|revert|style|test)(\([a-z0-9][a-z0-9._/-]*\))?!?: [^ ].+$'
if ! [[ "${subject}" =~ ${pattern} ]]; then
  cat >&2 <<'EOF'
commit-msg: subject must use:
  <type>(<optional-scope>): <imperative summary>

Examples:
  fix(tasks): standardize commit message template
  docs: add commit message guide

Allowed types:
  build, chore, ci, docs, feat, fix, perf, refactor, release, revert, style, test

See docs/guides/commit-messages.md for details.
EOF
  exit 1
fi

if ((${#subject} > 72)); then
  echo "commit-msg: subject is ${#subject} chars; keep it under 72" >&2
  exit 1
fi

if [[ "${subject}" == *. ]]; then
  echo "commit-msg: subject should not end with a period" >&2
  exit 1
fi

nightshift_task_count=$(grep -c '^Nightshift-Task: ' "$msg_file" || true)
nightshift_ref_count=$(grep -c '^Nightshift-Ref: ' "$msg_file" || true)

if ((nightshift_task_count + nightshift_ref_count == 0)); then
  exit 0
fi

if ((nightshift_task_count != 1 || nightshift_ref_count != 1)); then
  cat >&2 <<'EOF'
commit-msg: Nightshift commits must include both trailers:
  Nightshift-Task: <task-id>
  Nightshift-Ref: https://github.com/marcus/nightshift
EOF
  exit 1
fi

if ! grep -qx 'Nightshift-Ref: https://github.com/marcus/nightshift' "$msg_file"; then
  echo "commit-msg: Nightshift-Ref must be https://github.com/marcus/nightshift" >&2
  exit 1
fi

first_trailer_line=$(grep -n '^Nightshift-\(Task\|Ref\): ' "$msg_file" | head -n 1 | cut -d: -f1)
if [[ -n "${first_trailer_line}" ]] && ((first_trailer_line > 1)); then
  line_before=$(sed -n "$((first_trailer_line - 1))p" "$msg_file")
  if [[ -n "${line_before}" ]]; then
    echo "commit-msg: leave a blank line before Nightshift trailers" >&2
    exit 1
  fi
fi
