#!/usr/bin/env bash
# Publish a validation run to the reports/ site served by the nginx container.
#
# usage: publish-report.sh <run-dir>
#
# Summarizes the commons-ip reports in <run-dir> into <run-dir>/run.json,
# rebuilds the reports/runs.json index over all runs (newest first), and copies
# the static viewer so http://localhost:8080 always shows the latest state.
# Requires: jq
set -euo pipefail
shopt -s nullglob

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

run_dir="${1:?usage: $0 <run-dir>}"
run_dir="$(cd "$run_dir" && pwd)"
run_id="$(basename "$run_dir")"
reports_root="$ROOT/reports"

reports=("$run_dir"/*validation-report*.json)
if [ ${#reports[@]} -eq 0 ]; then
    echo "$0: no validation reports in $run_dir" >&2
    exit 1
fi

packages="$(
    for report in "${reports[@]}"; do
        jq --arg report_file "$(basename "$report")" '{
            report_file: $report_file,
            package: (.header.path | split("/") | last),
            result: .summary.result,
            errors: .summary.errors,
            warnings: .summary.warnings,
            passed: .summary.success,
            skipped: .summary.skipped,
            notes: .summary.notes,
            date: .header.date,
            commons_ip: .header.version_commons_ip,
            specifications: [.header.specifications[].id]
        }' "$report"
    done | jq -s .
)"
jq -n --arg run "$run_id" --argjson packages "$packages" \
    '{run: $run, packages: $packages}' > "$run_dir/run.json"

jq -s 'sort_by(.run) | reverse' "$reports_root"/runs/*/run.json > "$reports_root/runs.json"

cp "$ROOT/scripts/report/index.html" "$reports_root/index.html"

echo "published $run_id ($(jq -r '.packages | length' "$run_dir/run.json") package(s))"
