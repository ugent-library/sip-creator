#!/usr/bin/env bash
# Structural diff of two generated SIP package trees (refactoring-plan Phase 0).
# Normalizes run-varying values, then diffs; exits non-zero on any difference.
#
# usage: baseline-diff.sh <ref-pkg-dir> <new-pkg-dir>
#   e.g. baseline-diff.sh tmp/baseline/pkg basic-uuid/uuid-*/
#
# Normalized (legitimately different on every run):
#   - UUIDs, mapped to uuid-1, uuid-2, ... in first-seen order per document,
#     so cross-references (structMap -> fileSec, PREMIS relationships) must
#     still line up structurally — unlike mapping every UUID to one token,
#     which would hide a dangling or swapped reference.
#   - ISO datetimes (CREATED/CREATEDATE attrs and PREMIS element text).
#   - SIZE/CHECKSUM of *generated XML* entries (mdRef and file/FLocat whose
#     href is an .xml outside data/): their raw bytes contain UUIDs and
#     timestamps, so their fixity churns by construction.
#   - <file> order within a fileGrp (sorted by href): the current code walks
#     a Go map for schema files, so METS emits them in random order per run.
#     fileSec order carries no meaning (structure lives in structMap); the
#     refactor's canonical emitter should sort at the source, making this
#     normalization inert.
#   - the submitting organization's name and IDENTIFICATIONCODE note: these
#     are operator config (SIP_SUBMITTER_* env vars), so their values vary
#     per environment; the agent's structure still must match.
# Deliberately NOT normalized (a diff here is a real regression):
#   - essence and schema fixity (SIZE/CHECKSUM of data/* and schemas/*),
#   - PREMIS messageDigest/size element text (essence fixity),
#   - everything else: file inventory, element structure, attribute values.
set -euo pipefail

if [ $# -ne 2 ] || [ ! -d "$1" ] || [ ! -d "$2" ]; then
    echo "usage: $0 <ref-pkg-dir> <new-pkg-dir>  (unzipped package dirs)" >&2
    exit 2
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cp -R "$1" "$work/ref"
cp -R "$2" "$work/new"

normalize() {
    find "$1" -name '*.xml' -exec perl -0777 -pi -e '
        sub href_of { my ($s) = @_; return $s =~ /xlink:href="([^"]*)"/ ? $1 : "" }

        # Order <file> siblings by href before renumbering UUIDs, so the
        # map-iteration randomness of the current schema emission cancels out.
        s{(<fileGrp\b[^>]*>)(.*?)(</fileGrp>)}{
            my ($open, $body, $close) = ($1, $2, $3);
            my @sorted = sort { href_of($a) cmp href_of($b) } $body =~ m{<file\b.*?</file>}gs;
            my $i = 0;
            $body =~ s{<file\b.*?</file>}{$sorted[$i++]}gs;
            $open . $body . $close;
        }gse;

        # Generated-XML fixity: mdRef carries its href inline; file elements
        # carry it on the FLocat child. Blank SIZE/CHECKSUM only when the href
        # is an .xml outside data/ — essence and schema fixity must survive.
        sub is_generated { my ($href) = @_; return $href =~ /\.xml$/ && $href !~ m{(^|/|%2F)data(/|%2F)}i }
        sub blank { my ($tag) = @_; $tag =~ s/SIZE="[0-9]+"/SIZE="N"/; $tag =~ s/CHECKSUM="[0-9a-fA-F]+"/CHECKSUM="X"/; return $tag }
        s{<mdRef\b[^>]*>}{
            my $tag = $&;
            $tag =~ /xlink:href="([^"]*)"/ && is_generated($1) ? blank($tag) : $tag;
        }ge;
        s{(<file\b[^>]*>)(\s*<FLocat\b[^>]*>)}{
            my ($tag, $flocat) = ($1, $2);
            ($flocat =~ /xlink:href="([^"]*)"/ && is_generated($1) ? blank($tag) : $tag) . $flocat;
        }ge;

        # The submitting organization is operator config (SIP_SUBMITTER_*),
        # not code output: blank its name and OR-id note so the baseline
        # never encodes one operator env. The agent structure still compares.
        s{(<agent ROLE="CREATOR" TYPE="ORGANIZATION">)(.*?)(</agent>)}{
            my ($open, $body, $close) = ($1, $2, $3);
            $body =~ s{<name>[^<]*</name>}{<name>SUBMITTER</name>};
            $body =~ s{(<note csip:NOTETYPE="IDENTIFICATIONCODE">)[^<]*(</note>)}{$1ORID$2};
            $open . $body . $close;
        }gse;

        # XML indentation and blank lines are not structural. (Added at
        # plan Step 7: templated metsHdr agents indent uniformly where the
        # old hardcoded block mixed tabs and spaces.)
        s/^[ \t]+//mg;
        s/^[ \t]*\n//mg;

        # ISO datetimes, attribute or element text. Requires the T-time part,
        # so descriptive dates like <dcterms:created>2022-01-01 survive.
        s/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(Z|[+-][0-9]{2}:[0-9]{2})?/TS/g;

        # UUIDs -> first-seen sequence numbers (map is per document: BEGIN-less
        # perl -pi runs this once per file with %map reset by local).
        our %map; local %map = (); our $n; local $n = 0;
        s/uuid-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/$map{$&} \/\/= "uuid-" . ++$n/ge;
    ' {} +
}

normalize "$work/ref"
normalize "$work/new"

if diff -ru "$work/ref" "$work/new"; then
    echo "OK: trees are structurally identical"
else
    echo "FAIL: trees differ (see diff above)" >&2
    exit 1
fi
