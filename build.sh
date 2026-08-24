#!/usr/bin/env bash
# Local CI loop: rebuild, regenerate the sample SIP for a profile, validate
# with commons-ip, publish the HTML report (serve it: docker compose up -d reports).
#
# usage: build.sh [profile]    (default: basic)
#
# Exits non-zero iff the generated package is not VALID. Each profile
# validates against the E-ARK spec version of its era: basic (meemoo 1.2)
# against 2.0.4, eark against 2.2.0 (docs/archive/meemoo-12.md).
#
# Input fixture: ./tmp/<profile> (untracked). The eark fixture is the basic
# one plus an optional documentation/ directory (recommended: CSIPSTR16 is a
# SHOULD, and package-level documentation satisfies it).
# Requires: go, docker, jq. Siegfried (sf) on PATH is recommended: the
# fixture's siegfried.json sidecar is regenerated each run: the assembler
# verifies its MD5s against the source bytes, so a stale sidecar is a hard
# build failure by design (ADR-0009).
set -euo pipefail
cd "$(dirname "$0")"

PROFILE="${1:-basic}"
SRC="tmp/$PROFILE"
OUT="$PROFILE-uuid"

case "$PROFILE" in
    basic) SPEC_VERSION=2.0.4 ;;
    *)     SPEC_VERSION=2.2.0 ;;
esac

if [ ! -d "$SRC" ]; then
    echo "missing input fixture $SRC; create it, e.g.:" >&2
    echo "  cp -R tmp/basic $SRC" >&2
    echo "  mkdir $SRC/documentation && echo 'sample documentation' > $SRC/documentation/README.txt" >&2
    exit 2
fi

# Refresh the fixture's characterization sidecar (ADR-0009). Capture first,
# write after: sf must never scan its own half-written output.
if command -v sf >/dev/null; then
    report="$(cd "$SRC" && sf -hash md5 -json .)"
    printf '%s\n' "$report" > "$SRC/siegfried.json"
elif [ -f "$SRC/siegfried.json" ]; then
    echo "warning: sf not on PATH; $SRC/siegfried.json may be stale, and a stale sidecar aborts the build" >&2
else
    echo "warning: sf not on PATH and no $SRC/siegfried.json; building without format info" >&2
fi

go build -o bin/sip-creator .

rm -rf "$OUT"
./bin/sip-creator create --profile "$PROFILE" "$SRC" "$OUT"

run_dir="reports/runs/$(date -u +%Y%m%dT%H%M%SZ)-$PROFILE"
mkdir -p "$run_dir"

# The acceptance check: validate the zip only; the zip is the deliverable that gets ingested.
status=0
./scripts/validate.sh -o "$run_dir" -s "$SPEC_VERSION" "$OUT"/uuid-*.zip || status=$?

./scripts/publish-report.sh "$run_dir"
echo "report: http://localhost:8080/"

exit "$status"
