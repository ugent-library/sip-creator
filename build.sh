#!/usr/bin/env bash
# Local CI loop: rebuild, regenerate the sample SIP from ./tmp/basic, validate
# with commons-ip, publish the HTML report (serve it: docker compose up -d reports).
# Requires: go, docker, jq, and a configured .env (working Siegfried install).
set -euo pipefail
cd "$(dirname "$0")"

OUT=basic-uuid

go build -o bin/sip-creator .

rm -rf "$OUT"
./bin/sip-creator create --profile basic ./tmp/basic "$OUT"

run_dir="reports/runs/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$run_dir"

# The gate: validate the zip only — the zip is the deliverable meemoo ingests.
# Publish the report either way; the exit code stays the validator's verdict.
status=0
./scripts/validate.sh -o "$run_dir" "$OUT"/uuid-*.zip || status=$?

./scripts/publish-report.sh "$run_dir"
echo "report: http://localhost:8080/"

exit "$status"
