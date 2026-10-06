#!/usr/bin/env bash
# Local CI loop: rebuild, regenerate the sample SIP for a profile, validate
# with commons-ip, publish the HTML report (serve it: docker compose up -d reports).
#
# usage: build.sh [profile] [input]    (default: basic, examples/<profile>)
#
# Exits non-zero iff the generated package is not VALID, or a descriptive
# document in it is not valid against its schema. Each profile validates
# against the E-ARK spec version of its era: basic (Meemoo 1.2) against
# 2.0.4, eark and eark-mods against 2.2.0 (docs/archive/meemoo-12.md).
# commons-ip does not validate the descriptive documents the METS points
# at, so every mods.xml, dc.xml and dc+schema.xml in the package is checked
# with xmllint against the schema the package ships, offline through
# scripts/schema-catalog.xml (docs/input-spec.md §3).
#
# The input is copied to tmp/build/<profile> and built from there, because
# the siegfried.json sidecar is written next to the input on every run and
# examples/ is tracked in git.
# Requires: go, docker, jq, xmllint. Siegfried (sf) on PATH is recommended:
# the copy's siegfried.json sidecar is generated each run: the assembler
# verifies its MD5s against the source bytes, so a stale sidecar is a hard
# build failure by design (ADR-0009).
set -euo pipefail
cd "$(dirname "$0")"

PROFILE="${1:-basic}"
INPUT="${2:-examples/$PROFILE}"
SRC="tmp/build/$PROFILE"
OUT="$PROFILE-uuid"

case "$PROFILE" in
    basic)          SPEC_VERSION=2.0.4 ;;
    eark|eark-mods) SPEC_VERSION=2.2.0 ;;
    *)
        echo "unknown profile $PROFILE (one of: basic, eark, eark-mods)" >&2
        exit 2
        ;;
esac

if [ ! -d "$INPUT" ]; then
    echo "missing input folder $INPUT" >&2
    exit 2
fi
if [ "$(cd "$INPUT" && pwd)" = "$PWD/$SRC" ]; then
    echo "input folder $INPUT is the build copy $SRC, which each run deletes; pass another folder" >&2
    exit 2
fi

rm -rf "$SRC"
mkdir -p "$(dirname "$SRC")"
# -p keeps the files' modification times, which the package keeps too.
cp -Rp "$INPUT" "$SRC"

# Generate the copy's characterization sidecar (ADR-0009). Capture first,
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
pkg="$(ls -d "$OUT"/uuid-*/)"
pkg="${pkg%/}"

status=0

# commons-ip checks the METS, not the descriptive documents it points at,
# and a supplied document is copied into the package as it is, so each
# descriptive document is validated here against the schema the package
# ships for it. The MODS schema imports two loc.gov URLs; the catalog maps
# them onto the bundled copies so xmllint stays offline.
validate_documents() {
    local name="$1" schema="$2"
    local files=()
    while IFS= read -r f; do files+=("$f"); done < <(find "$pkg" -name "$name" | sort)
    [ "${#files[@]}" -gt 0 ] || return 0
    if ! command -v xmllint >/dev/null; then
        echo "xmllint not on PATH; it checks every $name in the package against $schema" >&2
        exit 2
    fi
    XML_CATALOG_FILES="$PWD/scripts/schema-catalog.xml" \
        xmllint --noout --nonet --schema "$pkg/schemas/$schema" "${files[@]}" || status=1
}
validate_documents mods.xml mods-3-7.xsd
validate_documents dc.xml simpledc.xsd
validate_documents dc+schema.xml descriptive_basic.xsd

run_dir="reports/runs/$(date -u +%Y%m%dT%H%M%SZ)-$PROFILE"
mkdir -p "$run_dir"

# The acceptance check: validate the zip only; the zip is the deliverable that gets ingested.
./scripts/validate.sh -o "$run_dir" -s "$SPEC_VERSION" "$OUT"/uuid-*.zip || status=$?

./scripts/publish-report.sh "$run_dir"
echo "report: http://localhost:8080/"

exit "$status"
