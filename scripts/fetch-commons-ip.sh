#!/usr/bin/env bash
# Fetch the commons-ip CLI jar that validate.sh runs, and print its path.
#
# usage: fetch-commons-ip.sh
#
# The jar is pinned by version AND sha256 so a re-tagged release can't
# silently change what "valid" means; bump both together, deliberately
# (docs/decisions/0035-commons-ip-runs-on-host-java.md). The download goes
# to tmp/commons-ip/, which git ignores, and happens once; the checksum is
# verified on every call, and a jar that fails it is deleted.
# Requires: curl, and shasum or sha256sum
set -euo pipefail

COMMONS_IP_VERSION=2.11.2
COMMONS_IP_SHA256=e43bd94bb37a6f04a192b160010fc7e1e4cc1423f9d41fd04f61d8303ec3648e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
jar_name="commons-ip2-cli-$COMMONS_IP_VERSION.jar"
jar="$ROOT/tmp/commons-ip/$jar_name"
url="https://github.com/keeps/commons-ip/releases/download/$COMMONS_IP_VERSION/$jar_name"

if [ ! -f "$jar" ]; then
    echo "fetching commons-ip $COMMONS_IP_VERSION from $url" >&2
    mkdir -p "$(dirname "$jar")"
    curl --fail --silent --show-error --location --output "$jar" "$url"
fi

# macOS ships shasum, coreutils ships sha256sum; either prints "<hex>  <file>".
if command -v sha256sum >/dev/null; then
    actual="$(sha256sum "$jar" | cut -d' ' -f1)"
elif command -v shasum >/dev/null; then
    actual="$(shasum -a 256 "$jar" | cut -d' ' -f1)"
else
    echo "neither sha256sum nor shasum on PATH; one of them verifies $jar" >&2
    exit 2
fi

if [ "$actual" != "$COMMONS_IP_SHA256" ]; then
    rm -f "$jar"
    echo "$jar has sha256 $actual, expected $COMMONS_IP_SHA256; deleted it, run again to fetch a fresh copy" >&2
    exit 1
fi

echo "$jar"
