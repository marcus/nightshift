#!/bin/sh

set -u

usage() {
  echo "usage: $0 <commit-message-file>" >&2
}

reject() {
  echo "commit-msg: $1" >&2
  echo "expected: type(scope)!: summary" >&2
  echo "types: build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

message_file=$1

if [ ! -f "$message_file" ]; then
  echo "commit-msg: message file does not exist or is not a regular file: $message_file" >&2
  exit 2
fi

if [ ! -r "$message_file" ] || [ ! -w "$message_file" ]; then
  echo "commit-msg: message file must be readable and writable: $message_file" >&2
  exit 2
fi

subject=
IFS= read -r subject < "$message_file"
read_status=$?
if [ "$read_status" -ne 0 ] && [ -z "$subject" ]; then
  reject "commit message is empty"
fi

case "$subject" in
  Merge\ * | Revert\ \"*\" | fixup!\ * | squash!\ * | amend!\ * | \
    WIP\ on\ *:* | On\ *:\ * | index\ on\ *:* | untracked\ files\ on\ *:*)
    exit 0
    ;;
esac

is_combined_message=0
commented_probe=$(printf 'probe\n' | git stripspace --comment-lines 2>/dev/null) || commented_probe='# probe'
case "$commented_probe" in
  *" probe")
    comment_prefix=${commented_probe%" probe"}
    case "$subject" in
      "$comment_prefix This is a combination of "*)
        combined_count=${subject#"$comment_prefix This is a combination of "}
        if printf '%s\n' "$combined_count" | LC_ALL=C grep -Eq '^([2-9]|[1-9][0-9]+) commits\.$'; then
          is_combined_message=1
        fi
        ;;
    esac
    ;;
esac

comment_config_line=$(git config --get-regexp '^core\.comment(char|string)$' 2>/dev/null | tail -n 1)
comment_setting=${comment_config_line#* }
if [ "$is_combined_message" -eq 0 ] && [ "$comment_setting" = "auto" ]; then
  if printf '%s\n' "$subject" |
    LC_ALL=C grep -Eq '^[#;@!$%^&|:] This is a combination of ([2-9]|[1-9][0-9]+) commits\.$'; then
    is_combined_message=1
  fi
fi

if [ "$is_combined_message" -eq 1 ]; then
  exit 0
fi

trimmed_subject=$(
  printf '%s\n' "$subject" |
    LC_ALL=C sed \
      -e 's/[[:blank:]][[:blank:]]*/ /g' \
      -e 's/^ //' \
      -e 's/ $//'
)

case "$trimmed_subject" in
  *:*)
    raw_header=${trimmed_subject%%:*}
    summary=${trimmed_subject#*:}
    ;;
  *)
    reject "subject is not a Conventional Commit"
    ;;
esac

header=$(
  printf '%s\n' "$raw_header" |
    LC_ALL=C sed \
      -e 's/^ //' \
      -e 's/ $//' \
      -e 's/[[:blank:]]*(/(/' \
      -e 's/)[[:blank:]]*!$/)!/' \
      -e 's/[[:blank:]]*!$/!/' |
    LC_ALL=C tr '[:upper:]' '[:lower:]'
)

summary=$(
  printf '%s\n' "$summary" |
    LC_ALL=C sed \
      -e 's/[[:blank:]][[:blank:]]*/ /g' \
      -e 's/^ //' \
      -e 's/ $//'
)

if ! printf '%s\n' "$header" |
  LC_ALL=C grep -Eq '^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9][a-z0-9._/-]*\))?!?$'; then
  reject "subject has an unsupported type or malformed scope"
fi

if [ -z "$summary" ]; then
  reject "subject summary is empty"
fi

normalized_subject="$header: $summary"
if [ "$normalized_subject" = "$subject" ]; then
  exit 0
fi

subject_bytes=$(printf '%s' "$subject" | LC_ALL=C wc -c | tr -d '[:space:]')
temporary_file=$(mktemp "${message_file}.normalize.XXXXXX") || {
  echo "commit-msg: could not create a temporary message file" >&2
  exit 2
}

cleanup() {
  rm -f "$temporary_file"
}
trap cleanup EXIT HUP INT TERM

if ! printf '%s' "$normalized_subject" > "$temporary_file"; then
  echo "commit-msg: could not write normalized message" >&2
  exit 2
fi

if ! dd if="$message_file" bs=1 skip="$subject_bytes" >> "$temporary_file" 2>/dev/null; then
  echo "commit-msg: could not preserve the commit message body" >&2
  exit 2
fi

if ! mv "$temporary_file" "$message_file"; then
  echo "commit-msg: could not replace the commit message" >&2
  exit 2
fi

trap - EXIT HUP INT TERM
