#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
response=$(gh api "repos/$GITHUB_REPOSITORY/commits/main")
current=$(printf '%s\n' "$response" | jq -er -f "$script_dir/current-commit-sha.jq")
if [ "$current" = "$GITHUB_SHA" ]; then
	printf '%s\n' 'deploy=true' >> "$GITHUB_OUTPUT"
else
	echo "Skipping an outdated main-branch deployment."
fi
