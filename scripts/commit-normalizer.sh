#!/bin/sh
# commit-normalizer.sh — language-agnostic Conventional Commits normalizer.
#
# Reads a commit message file, validates and normalizes its first line to
# Conventional Commits format `<type>(<scope>)!: <subject>` (scope and `!`
# optional), trims trailing whitespace, lowercases the type, and preserves
# the body and trailer block intact.
#
# Exit codes:
#   0 — message normalized (or intentionally skipped)
#   1 — unfixable violation (caller should reject the commit)
#   2 — usage error
#
# Skip rules: merge commits, fixup!/squash! commits, revert commits, and
# messages that already begin with "Revert " are left untouched.

set -eu

PROG=$(basename "$0")

usage() {
    cat >&2 <<EOF
Usage: $PROG <commit-message-file>

Reads the file, normalizes the commit subject to Conventional Commits, and
writes the result back in place. Exits non-zero on unfixable violations.
EOF
    exit 2
}

[ "$#" -eq 1 ] || usage
MSG_FILE=$1
[ -f "$MSG_FILE" ] || { echo "$PROG: not a file: $MSG_FILE" >&2; exit 2; }

# Subject length limit (configurable via env).
MAX_SUBJECT_LEN=${COMMIT_NORMALIZER_MAX_SUBJECT:-72}

# Allowed Conventional Commits types.
ALLOWED_TYPES="feat fix docs style refactor perf test build ci chore revert"

# Read the file, stripping comment lines (git's scissor lines start with '#').
RAW=$(awk '!/^#/' "$MSG_FILE")

# Determine the first non-empty line — that's the subject candidate.
SUBJECT=$(printf '%s\n' "$RAW" | awk 'NF { print; exit }')

# Strip CR (Windows line endings) and trailing whitespace from the subject.
SUBJECT=$(printf '%s' "$SUBJECT" | tr -d '\r' | sed 's/[[:space:]]*$//')

# Skip rules — leave the message untouched.
case "$SUBJECT" in
    "Merge "*|"Merge branch "*|"Merge remote-tracking "*|"Merge pull request "*)
        exit 0 ;;
    "fixup! "*|"squash! "*|"amend! "*)
        exit 0 ;;
    "Revert "*|"revert: "*)
        # Revert commits keep their auto-generated subject.
        exit 0 ;;
    "")
        echo "$PROG: empty commit message" >&2
        exit 1 ;;
esac

# Parse the subject into type, scope (optional), breaking marker, body.
# Pattern:  type(scope)!: subject   |   type!: subject   |   type: subject
PARSE=$(printf '%s' "$SUBJECT" | awk '
    {
        line = $0
        # Capture optional leading type / scope / !.
        if (match(line, /^[A-Za-z]+(\([^)]+\))?!?:[[:space:]]/)) {
            header = substr(line, 1, RLENGTH)
            rest   = substr(line, RLENGTH + 1)
            # Split header into parts.
            hd = header
            sub(/:[[:space:]]*$/, "", hd)
            bang = ""
            if (hd ~ /!$/) { bang = "!"; sub(/!$/, "", hd) }
            scope = ""
            if (match(hd, /\([^)]+\)$/)) {
                scope = substr(hd, RSTART + 1, RLENGTH - 2)
                hd = substr(hd, 1, RSTART - 1)
            }
            type = hd
            printf "%s\t%s\t%s\t%s\n", type, scope, bang, rest
        } else {
            printf "\t\t\t%s\n", line
        }
    }
')

TYPE=$(printf '%s' "$PARSE" | awk -F'\t' '{print $1}')
SCOPE=$(printf '%s' "$PARSE" | awk -F'\t' '{print $2}')
BANG=$(printf '%s' "$PARSE" | awk -F'\t' '{print $3}')
BODY=$(printf '%s' "$PARSE" | awk -F'\t' '{print $4}')

if [ -z "$TYPE" ]; then
    cat >&2 <<EOF
$PROG: subject is not Conventional Commits format
  got:      $SUBJECT
  expected: <type>(<scope>)!: <subject>
  types:    $ALLOWED_TYPES
  example:  feat(parser): allow optional trailing comma
EOF
    exit 1
fi

# Lowercase the type.
TYPE=$(printf '%s' "$TYPE" | tr '[:upper:]' '[:lower:]')

# Validate type.
TYPE_OK=0
for t in $ALLOWED_TYPES; do
    if [ "$t" = "$TYPE" ]; then TYPE_OK=1; break; fi
done
if [ "$TYPE_OK" -ne 1 ]; then
    cat >&2 <<EOF
$PROG: unknown commit type "$TYPE"
  allowed: $ALLOWED_TYPES
EOF
    exit 1
fi

# Trim leading whitespace from the body portion of the subject.
BODY=$(printf '%s' "$BODY" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')

if [ -z "$BODY" ]; then
    echo "$PROG: subject has no description after '$TYPE:'" >&2
    exit 1
fi

# Rebuild the normalized subject.
if [ -n "$SCOPE" ]; then
    NORMALIZED="$TYPE($SCOPE)$BANG: $BODY"
else
    NORMALIZED="$TYPE$BANG: $BODY"
fi

# Enforce subject length.
SUBJECT_LEN=$(printf '%s' "$NORMALIZED" | awk '{print length}')
if [ "$SUBJECT_LEN" -gt "$MAX_SUBJECT_LEN" ]; then
    cat >&2 <<EOF
$PROG: subject too long ($SUBJECT_LEN > $MAX_SUBJECT_LEN chars)
  subject: $NORMALIZED
  tip:     move detail into the body (blank line, then explanation)
EOF
    exit 1
fi

# Reassemble the message: normalized subject + remaining lines after the
# original subject. We preserve the body and trailer block byte-for-byte
# (minus trailing whitespace per line). Comment lines from git templates
# are stripped — git would do this anyway.
TMP=$(mktemp "${TMPDIR:-/tmp}/commit-normalizer.XXXXXX")
trap 'rm -f "$TMP"' EXIT INT TERM

awk -v normalized="$NORMALIZED" '
    BEGIN { subject_done = 0 }
    /^#/  { next }
    {
        if (!subject_done && $0 ~ /[^[:space:]]/) {
            print normalized
            subject_done = 1
            next
        }
        # Strip trailing whitespace from every line.
        sub(/[[:space:]]+$/, "", $0)
        print
    }
' "$MSG_FILE" > "$TMP"

# Collapse trailing blank lines down to one terminating newline.
awk '
    { lines[NR] = $0 }
    END {
        n = NR
        while (n > 0 && lines[n] == "") n--
        for (i = 1; i <= n; i++) print lines[i]
    }
' "$TMP" > "$TMP.2"
mv "$TMP.2" "$MSG_FILE"
rm -f "$TMP"
trap - EXIT INT TERM

exit 0
