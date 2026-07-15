#!/bin/sh

# Normalize and validate a Conventional Commit message in place.
set -eu

usage() {
  echo "usage: $0 <commit-message-file>" >&2
}

if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

message_file=$1
if [ ! -f "$message_file" ] || [ ! -r "$message_file" ] || [ ! -w "$message_file" ]; then
  echo "commit message is not a readable, writable file: $message_file" >&2
  exit 2
fi

# Git creates these subjects itself. Rewriting them can break merges, reverts,
# autosquash, stashes, and interactive rebases, so preserve the entire file.
first_line=$(awk '
  {
    line = $0
    sub(/\r$/, "", line)
    if (line !~ /^[[:space:]]*$/) {
      sub(/^[[:space:]]+/, "", line)
      print line
      exit
    }
  }
' "$message_file")

comment_prefix=$(
  git config --get-regexp '^core\.comment(char|string)$' 2>/dev/null |
    awk '
      {
        sub(/^[^[:space:]]+[[:space:]]+/, "")
        prefix = $0
      }
      END { print prefix }
    '
)
case "$comment_prefix" in
  ""|auto)
    comment_prefix="#"
    ;;
esac

case "$first_line" in
  "$comment_prefix"*)
    exit 0
    ;;
esac

case "$first_line" in
  Merge\ *|Revert\ *|fixup\!\ *|squash\!\ *|amend\!\ *|WIP\ on\ *|index\ on\ *)
    exit 0
    ;;
esac

tmp_file=$(mktemp "${TMPDIR:-/tmp}/nightshift-commit-msg.XXXXXX") || exit 2
cleanup() {
  rm -f "$tmp_file"
}
trap cleanup 0
trap 'exit 1' 1 2 15

if ! awk '
  function trim(value) {
    sub(/^[[:space:]]+/, "", value)
    sub(/[[:space:]]+$/, "", value)
    return value
  }

  function collapse(value) {
    value = trim(value)
    gsub(/[[:space:]]+/, " ", value)
    return value
  }

  function supported(value) {
    return value == "build" || value == "chore" || value == "ci" ||
      value == "docs" || value == "feat" || value == "fix" ||
      value == "perf" || value == "refactor" || value == "style" ||
      value == "test"
  }

  {
    line = $0
    sub(/[[:space:]]+$/, "", line)
    lines[++count] = line
  }

  END {
    first = 1
    while (first <= count && lines[first] == "") {
      first++
    }

    last = count
    while (last >= first && lines[last] == "") {
      last--
    }

    if (first > last) {
      exit 1
    }

    subject = collapse(lines[first])
    colon = index(subject, ":")
    if (colon == 0) {
      exit 1
    }

    header = trim(substr(subject, 1, colon - 1))
    summary = collapse(substr(subject, colon + 1))
    if (summary == "") {
      exit 1
    }

    breaking = ""
    if (header ~ /![[:space:]]*$/) {
      sub(/![[:space:]]*$/, "", header)
      header = trim(header)
      breaking = "!"
    }

    open = index(header, "(")
    scope = ""
    if (open > 0) {
      if (substr(header, length(header), 1) != ")") {
        exit 1
      }
      type = tolower(trim(substr(header, 1, open - 1)))
      scope = collapse(substr(header, open + 1, length(header) - open - 1))
      if (scope == "" || scope !~ /^[[:alnum:]_.\/#-]+$/) {
        exit 1
      }
    } else {
      type = tolower(trim(header))
      if (type ~ /[()]/) {
        exit 1
      }
    }

    if (!supported(type)) {
      exit 1
    }

    normalized = type
    if (scope != "") {
      normalized = normalized "(" scope ")"
    }
    print normalized breaking ": " summary

    for (i = first + 1; i <= last; i++) {
      print lines[i]
    }
  }
' "$message_file" > "$tmp_file"; then
  cat >&2 <<'EOF'
invalid commit subject; expected type(scope)!: summary
supported types: build, chore, ci, docs, feat, fix, perf, refactor, style, test
EOF
  exit 1
fi

cat "$tmp_file" > "$message_file"
