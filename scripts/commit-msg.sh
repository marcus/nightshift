#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks
set -euo pipefail

TYPES="build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test"

usage() {
  echo "usage: $0 <commit-msg-file>" >&2
  exit 1
}

fail() {
  echo "commit-msg: $1" >&2
  echo "use: type(scope): summary" >&2
  echo "types: build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test" >&2
  echo 'exceptions: merge commits, Revert "...", Bump version..., Release v...' >&2
  exit 1
}

[[ $# -eq 1 ]] || usage

msg_file="$1"
[[ -f "$msg_file" ]] || usage

lines=()
while IFS= read -r line || [[ -n "$line" ]]; do
  line=${line%$'\r'}
  if [[ "$line" =~ ^[[:space:]]*# ]]; then
    continue
  fi
  lines+=("$line")
done < "$msg_file"

while [[ ${#lines[@]} -gt 0 ]]; do
  last_index=$((${#lines[@]} - 1))
  [[ -n "${lines[$last_index]}" ]] && break
  unset 'lines[$last_index]'
done

[[ ${#lines[@]} -gt 0 ]] || fail "empty commit message"

subject="${lines[0]}"
subject_core="$subject"
if [[ "$subject_core" =~ ^(.+)\ \(#[0-9]+\)$ ]]; then
  subject_core="${BASH_REMATCH[1]}"
fi

merge_re='^Merge (branch|remote-tracking branch|pull request|tag) '
revert_re='^Revert ".*"$'
release_bump_re='^Bump version to v[0-9]+(\.[0-9]+)*([.-][A-Za-z0-9]+)*$'
release_re='^Release v[0-9]+(\.[0-9]+)*([.-][A-Za-z0-9]+)*(: .+)?$'

if git rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1 || [[ "$subject" =~ $merge_re ]] || [[ "$subject_core" =~ $revert_re ]] || [[ "$subject_core" =~ $release_bump_re ]] || [[ "$subject_core" =~ $release_re ]]; then
  exit 0
fi

subject_re="^(${TYPES})(\\([A-Za-z0-9#][A-Za-z0-9._/#-]*\\))?(!)?: .+$"
[[ "$subject" =~ $subject_re ]] || fail "expected Conventional Commits subject"

[[ "$subject_core" != *. ]] || fail "subject must not end with a period"

if [[ ${#lines[@]} -gt 1 ]] && [[ -n "${lines[1]}" ]]; then
  fail "leave line 2 blank before body or trailers"
fi
