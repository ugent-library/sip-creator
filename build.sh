#!/usr/bin/env bash
# Local CI loop: rebuild, regenerate the sample SIP for a profile, validate
# with commons-ip, publish the HTML report (serve it: docker compose up -d reports).
#
# usage: build.sh [profile]    (default: basic)
#
# Exits non-zero iff the generated package is not VALID. The basic sample is
# known-INVALID (docs/TODO.md) — that exit code is a real signal, not a broken
# script. The eark sample is expected VALID.
#
# Input fixture: ./tmp/<profile> (untracked). The eark fixture is the basic
# one plus an optional documentation/ directory (recommended: CSIPSTR16 is a
# SHOULD, and package-level documentation satisfies it).
# Requires: go, docker, jq. Siegfried is optional (.env).
set -euo pipefail
cd "$(dirname "$0")"

PROFILE="${1:-basic}"
SRC="tmp/$PROFILE"
OUT="$PROFILE-uuid"

if [ ! -d "$SRC" ]; then
    echo "missing input fixture $SRC — create it, e.g.:" >&2
    echo "  cp -R tmp/basic $SRC" >&2
    echo "  mkdir $SRC/documentation && echo 'sample documentation' > $SRC/documentation/README.txt" >&2
    exit 2
fi

go build -o bin/sip-creator .

rm -rf "$OUT"
./bin/sip-creator create --profile "$PROFILE" "$SRC" "$OUT"

run_dir="reports/runs/$(date -u +%Y%m%dT%H%M%SZ)-$PROFILE"
mkdir -p "$run_dir"

# The gate: validate the zip only — the zip is the deliverable that gets ingested.
status=0
./scripts/validate.sh -o "$run_dir" "$OUT"/uuid-*.zip || status=$?

./scripts/publish-report.sh "$run_dir"
echo "report: http://localhost:8080/"

exit "$status"
