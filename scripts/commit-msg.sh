#!/usr/bin/env bash
# commit-msg hook for nightshift
# Install: make install-hooks
set -euo pipefail

if [[ $# -ne 1 ]]; then
	printf 'commit-msg: expected the commit message file path\n' >&2
	exit 2
fi

repo_root=$(git rev-parse --show-toplevel)

if [[ -x "$repo_root/nightshift" ]]; then
	exec "$repo_root/nightshift" commit-msg "$1"
fi

if command -v nightshift >/dev/null 2>&1 && nightshift commit-msg --help >/dev/null 2>&1; then
	exec nightshift commit-msg "$1"
fi

cd "$repo_root"
exec go run ./cmd/nightshift commit-msg "$1"
