#!/bin/sh

set -eu

printf '%s\n' "$@" > "$MOCK_GH_ARGUMENTS"
if [ "${MOCK_GH_STATUS:-0}" -ne 0 ]; then
	echo "mock GitHub API failure" >&2
	exit "$MOCK_GH_STATUS"
fi
cat "$MOCK_GH_RESPONSE"
