# 0035 — commons-ip runs on the host's Java

Status: Accepted (2026-10-09)

## Context

[ADR-0005](0005-dockerized-validation-and-html-reporting.md) put the commons-ip validator in a Docker image, pinned by version and sha256, so that every machine ran the same invocation and nobody kept a host jar. On the macOS machine the project is developed on, Docker Desktop's backend died during `docker compose run` of the validator, taking every running container with it. The first diagnosis blamed the launchd open-file limit of 256; raising it to 65536 did not stop the crashes, and the root cause was never found. From 2026-09-29 on, every acceptance run bypassed Docker through the `CSIP_CMD` override, with a jar downloaded by hand from the URL the Dockerfile pinned and checked by hand against its sha256. The results were identical to the Docker runs: the same verdicts and the same pass counts for every profile.

The Docker image also carried the nginx service that served the HTML reports.

## Decision

- **commons-ip runs on the host's Java.** `scripts/validate.sh` runs `java -jar` on a jar it takes from `scripts/fetch-commons-ip.sh`. That script holds the version and sha256 pins that the Dockerfile held, downloads the release jar once into `tmp/commons-ip/`, verifies the checksum on every call and deletes a jar that fails it. Upgrading the validator stays a deliberate two-line change, now in the fetch script.
- **The validator runs from the report directory**, so the log file commons-ip writes to its working directory lands with the reports and not in the repository.
- **`CSIP_CMD` stays** as the way to run another command, for example the commons-ip version RODA bundles, to see what the deployed validator says about a package.
- **`reports/` is served by any static file server.** CONTRIBUTING names `python3 -m http.server`. The viewer needs a server only because browsers refuse `fetch` from a page opened from disk; the VALID verdict never needs one.
- **The Docker image, the compose file and the nginx service are removed.** Java 17 or later and `curl` replace Docker in the requirements.

## Alternatives rejected

- **Keep both paths, Docker by default and the host jar on request.** Two code paths to maintain, one of which cannot be run on the machine the project is developed on. The host path has been the only one in use since 2026-09-29.
- **Keep raising the open-file limit.** The limit is per login session and resets on reboot, and the crash recurred with the limit raised.
- **Fix the viewer to open from disk.** Possible by loading the reports through script tags instead of `fetch`, but it changes the viewer and the publish script to save one command that any machine with Python has.

## Consequences

- The requirements for `build.sh` become Go, Java 17 or later, `curl`, `jq`, `sf` and `xmllint`. The Go tests still need none of them.
- The validator's Java runtime is whatever `java` is on `PATH`, not the pinned JRE of the image. commons-ip 2.11.2 has produced the same results under Java 21 in the image and Java 27 on the host.
- A future GitHub Actions job runs the same scripts: the runners ship Java and `curl`.
- The first `build.sh` on a machine downloads the jar, about 11 MB, from GitHub.
