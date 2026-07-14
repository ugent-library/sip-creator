#!/usr/bin/env bash
# Validate SIP packages (zip or directory) against E-ARK CSIP with commons-ip.
# Prints FAILED checks with their messages; exits non-zero if any package is INVALID.
#
# usage: validate.sh [-o report-dir] <sip.zip|sip-dir>...
#
# Reports land in -o (kept, for the HTML publisher) or a temp dir (cleaned up).
# Validation runs in the dockerized validator by default; set CSIP_CMD to a host
# command (e.g. CSIP_CMD="java -jar commons-ip.jar") to bypass docker.
# Requires: docker (or CSIP_CMD), jq
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# The spec version is pinned so a commons-ip upgrade changing its default
# can't silently move the goalposts (docs/decisions/0003-validation-stays-external.md).
SPEC_VERSION=2.2.0

report_dir=""
while getopts "o:" opt; do
    case $opt in
        o) report_dir="$OPTARG" ;;
        *) exit 2 ;;
    esac
done
shift $((OPTIND - 1))

if [ $# -eq 0 ]; then
    echo "usage: $0 [-o report-dir] <sip.zip|sip-dir>..." >&2
    exit 2
fi

if [ -z "$report_dir" ]; then
    report_dir="$(mktemp -d)"
    trap 'rm -rf "$report_dir"' EXIT
else
    mkdir -p "$report_dir"
fi
report_dir="$(cd "$report_dir" && pwd)"

run_csip() {
    local input="$1"
    if [ -n "${CSIP_CMD:-}" ]; then
        # shellcheck disable=SC2086 # CSIP_CMD is a command line, word splitting intended
        $CSIP_CMD validate --specification-version=$SPEC_VERSION \
            -i "$input" -o "$report_dir"
    else
        local dir base
        dir="$(cd "$(dirname "$input")" && pwd)"
        base="$(basename "$input")"
        # -w /reports: commons-ip drops a log file in its working directory,
        # which must land in the writable mount, not the container image.
        docker compose -f "$ROOT/docker-compose.yml" run --rm \
            --user "$(id -u):$(id -g)" \
            -v "$dir:/data:ro" -v "$report_dir:/reports" -w /reports \
            validator validate --specification-version=$SPEC_VERSION \
            -i "/data/$base" -o /reports
    fi
}

status=0
for input in "$@"; do
    if [ ! -e "$input" ]; then
        echo "== $input: ERROR — no such file or directory" >&2
        status=1
        continue
    fi

    # csip is chatty and exits 0 even for INVALID packages; keep its output for
    # crash diagnosis only and read the verdict from the JSON report.
    if ! csip_output="$(run_csip "$input" 2>&1)"; then
        echo "== $input: ERROR — csip failed" >&2
        echo "$csip_output" >&2
        status=1
        continue
    fi

    report="$(find "$report_dir" -name "$(basename "$input")_validation-report_*.json" | sort | tail -n 1)"
    if [ -z "$report" ]; then
        echo "== $input: ERROR — csip produced no validation report" >&2
        echo "$csip_output" >&2
        status=1
        continue
    fi

    jq -r --arg input "$input" '
        "== \($input): \(.summary.result) " +
        "(errors=\(.summary.errors) warnings=\(.summary.warnings) " +
        "passed=\(.summary.success) skipped=\(.summary.skipped))"
    ' "$report"

    jq -r '
        .validation[]
        | select(.testing.outcome == "FAILED")
        | "   FAILED [\(.level)] \(.id): \(.name)" +
          ((.testing.issues + .testing.warnings + .testing.notes)
           | map("\n      - " + .) | join(""))
    ' "$report"

    if [ "$(jq -r '.summary.result' "$report")" != "VALID" ]; then
        status=1
    fi
done

exit $status
