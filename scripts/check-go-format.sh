#!/bin/sh

set -eu

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
	printf '%s\n' "$unformatted"
	exit 1
fi
