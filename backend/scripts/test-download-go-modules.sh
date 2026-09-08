#!/bin/sh
set -eu
# Inject transient/permanent download failures; no network or actual modules.
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
cat > "$scratch/go" <<'MOCK'
#!/bin/sh
count=$(cat "$STATE")
count=$((count + 1))
echo "$count" > "$STATE"
test "$count" -gt "$FAILURES"
MOCK
printf '#!/bin/sh\nexit 0\n' > "$scratch/sleep"
chmod +x "$scratch/go" "$scratch/sleep"
export PATH="$scratch:$PATH" STATE="$scratch/count"
for failures in 0 1 2 3; do
  echo 0 > "$STATE"
  status=0
  FAILURES=$failures sh scripts/download-go-modules.sh || status=$?
  if [ "$failures" -lt 3 ]; then
    test "$status" -eq 0
    test "$(cat "$STATE")" -eq "$((failures + 1))"
  else
    test "$status" -eq 1
    test "$(cat "$STATE")" -eq 3
  fi
done
echo 'PASS: module download succeeds immediately, recovers after retries and fails after three attempts.'
