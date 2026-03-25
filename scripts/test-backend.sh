#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../backend"

test_packages=()
while IFS= read -r pkg; do
  test_packages+=("$pkg")
done <<EOF
$(go list -f '{{if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0)}}{{.ImportPath}}{{end}}' ./... | sed '/^$/d')
EOF

if [ "${#test_packages[@]}" -eq 0 ]; then
  echo "No Go test packages found." >&2
  exit 1
fi

# Go 1.25 coverage currently fails on packages without tests in this module.
go test "$@" "${test_packages[@]}"
