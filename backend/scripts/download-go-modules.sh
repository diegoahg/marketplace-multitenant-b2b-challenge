#!/bin/sh
set -eu
# Retry network failures without disabling TLS or Go checksum verification.
attempt=1
while ! go mod download; do
  if [ "$attempt" -ge 3 ]; then
    echo "Go module download failed after 3 attempts." >&2
    exit 1
  fi
  delay=$((attempt * 2))
  echo "Go module download failed; retrying in $delay seconds (attempt $((attempt + 1))/3)." >&2
  sleep "$delay"
  attempt=$((attempt + 1))
done
