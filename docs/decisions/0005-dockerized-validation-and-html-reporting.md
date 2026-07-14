# 0005 — Dockerized commons-ip validation with static HTML reporting

Status: Accepted (2026-07-14)

## Context

[ADR-0003](0003-validation-stays-external.md) fixed *what* validates a package: commons-ip, externally. The workflow around the tool was still ad hoc: a hand-rolled `~/bin/csip` wrapper pinning a stale jar (2.8.0) that exists only on one machine, a `build.sh` with no exit-code discipline, and validation results readable only by scrolling raw JSON. Two facts about commons-ip force workflow decisions: its CLI **always exits 0**, even for an INVALID package (only operational errors are nonzero), and it emits **JSON only** — no human-oriented report format.

## Decision

- **The validator runs in Docker**, built from `docker/validator/Dockerfile`: a JRE base plus the commons-ip release jar pinned by version **and sha256** (currently 2.11.2). No host jar, no per-machine wrapper. `scripts/validate.sh` is the only entry point; it accepts a `CSIP_CMD` override for running a host jar without Docker.
- **Pass/fail is read from the JSON report**, never the exit code: a package passes iff `.summary.result == "VALID"`. The spec version stays pinned (`--specification-version=2.2.0`), per ADR-0003.
- **Reporting is a static site generated from the commons-ip JSON**: each run's reports land in `reports/runs/<UTC timestamp>/`, `scripts/publish-report.sh` maintains a `runs.json` index, and one tracked HTML file (`scripts/report/index.html`, vanilla JS, no build step, no external assets) renders it client-side. nginx (`docker compose up -d reports`) serves `reports/` on localhost:8080. Shell and jq only move and aggregate JSON; no HTML is assembled in shell, and no Go code is involved anywhere in the pipeline.
- **Run history is kept**, not just the latest run, so the gate's red→green trajectory is visible. `reports/` is generated output and gitignored.

## Alternatives rejected

- **Keep the host `csip` wrapper.** Works on exactly one machine, pins an old jar implicitly, and leaves nothing for future CI to reuse. The Docker image is the same invocation everywhere, including a future GitHub Actions runner.
- **Generate HTML in shell/jq.** String-escaping XML fragments and check descriptions into HTML via shell is a bug farm; keeping presentation in one static JS file keeps the shell layer trivially auditable.
- **A report web app or site generator.** Against the small-and-boring dependency policy; a single self-contained HTML file needs no toolchain.
- **Latest-report-only.** Cheap to keep history (a few JSON files per run), and losing it hides regressions between runs.

## Consequences

- Host prerequisites for the dev loop shrink to Go, Docker, jq, and Siegfried; catmandu and the `~/bin/csip` wrapper are gone.
- Upgrading the validator is a deliberate two-line change (version + sha256 in the Dockerfile). The 2.8.0 → 2.11.2 jump may shift individual check outcomes; the first published run is the new baseline — acceptable while the gate is red (ADR-0003 anticipates a red gate).
- The report endpoint is a localhost dev convenience with no auth; `reports/` must never be exposed beyond the local machine as-is.
- JVM startup per validation (~seconds) rides on every `build.sh` run — same cost class as before, still unsuitable for a pre-commit hook (per the automation sequencing in the retired [validation-workflow plan](../archive/validation-workflow-plan.md): next steps are a pre-push hook once green, then CI).
