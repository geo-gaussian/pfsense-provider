#!/bin/sh

set -eu

case "$RELEASE_TAG" in
	v*) ;;
	*)
		echo "Release tag must begin with v." >&2
		exit 1
		;;
esac

ref="refs/tags/$RELEASE_TAG"
git check-ref-format "$ref"
commit=$(git rev-parse --verify "$ref^{commit}")
if [ -n "${EXPECTED_COMMIT:-}" ] && [ "$commit" != "$EXPECTED_COMMIT" ]; then
	echo "Release tag no longer points to the validated commit." >&2
	exit 1
fi
printf 'commit=%s\n' "$commit" >> "$GITHUB_OUTPUT"
