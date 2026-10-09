#!/usr/bin/env bash
# Validate SIP packages (zip or directory) against E-ARK CSIP with commons-ip.
# Prints FAILED checks with their messages; exits non-zero if any package is INVALID.
#
# usage: validate.sh [-o report-dir] [-s spec-version] <sip.zip|sip-dir>...
#
# Reports land in -o (kept, for the HTML publisher) or a temp dir (cleaned up).
# commons-ip runs on the host's java, from the jar fetch-commons-ip.sh pins
# (docs/decisions/0035-commons-ip-runs-on-host-java.md). Set CSIP_CMD to run
# another command instead, for example the commons-ip version RODA bundles:
# CSIP_CMD="java -jar commons-ip2-cli-2.8.0.jar".
# Requires: java 17 or later (or CSIP_CMD), jq, curl
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# The spec version is pinned so a commons-ip upgrade changing its default
# can't silently move the goalposts (docs/decisions/0003-validation-stays-external.md).
# -s overrides it per profile: Meemoo 1.2 builds on the E-ARK 2.0.4
# era (its mandated PROFILE URL is the one 2.0.4 checks for), the UGent
# profiles (ugent/basic, ugent/bibliographic) target 2.2.0 (docs/archive/meemoo-12.md).
SPEC_VERSION=2.2.0

report_dir=""
while getopts "o:s:" opt; do
    case $opt in
        o) report_dir="$OPTARG" ;;
        s) SPEC_VERSION="$OPTARG" ;;
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

if [ -n "${CSIP_CMD:-}" ]; then
    # shellcheck disable=SC2206 # CSIP_CMD is a command line, word splitting intended
    csip=($CSIP_CMD)
else
    if ! command -v java >/dev/null; then
        echo "java not on PATH; commons-ip needs Java 17 or later (or set CSIP_CMD)" >&2
        exit 2
    fi
    csip=(java -jar "$("$ROOT/scripts/fetch-commons-ip.sh")")
fi

run_csip() {
    local input
    # Absolute, because the command runs from the report directory: commons-ip
    # drops a log file in its working directory, which must not be the caller's.
    input="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
    (cd "$report_dir" && "${csip[@]}" validate --specification-version="$SPEC_VERSION" \
        -i "$input" -o "$report_dir")
}

status=0
for input in "$@"; do
    if [ ! -e "$input" ]; then
        echo "== $input: ERROR: no such file or directory" >&2
        status=1
        continue
    fi

    # csip is chatty and exits 0 even for INVALID packages; keep its output for
    # crash diagnosis only and read the verdict from the JSON report.
    if ! csip_output="$(run_csip "$input" 2>&1)"; then
        echo "== $input: ERROR: csip failed" >&2
        echo "$csip_output" >&2
        status=1
        continue
    fi

    report="$(find "$report_dir" -name "$(basename "$input")_validation-report_*.json" | sort | tail -n 1)"
    if [ -z "$report" ]; then
        echo "== $input: ERROR: csip produced no validation report" >&2
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
