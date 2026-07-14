# Plan: local CI pipeline with dockerized validation and HTML reports

*Status: shipped 2026-07-14, retired to archive. Implemented and extended [validation-workflow-plan.md](validation-workflow-plan.md) (retired alongside). The durable design lives in [sip-creator-design.md](../sip-creator-design.md) § Validation; the decision in [ADR-0005](../decisions/0005-dockerized-validation-and-html-reporting.md).*

## Context — why this change

[ADR-0003](../decisions/0003-validation-stays-external.md) fixed the strategy: CSIP rules live in commons-ip, and acceptance means generating a package and passing `csip validate`. The [validation-workflow plan](validation-workflow-plan.md) designed the honest-exit-code gate (`scripts/validate.sh` + a rewritten `build.sh`) but never shipped. Two new requirements extend it:

1. **A local CI pipeline** — one command that builds, generates the sample SIP, validates it against E-ARK CSIP, and exits honestly, with reproducible tooling (no Go code for the pipeline itself; shell, jq, Docker).
2. **A readable HTML report** of the validation results, published on a local HTTP endpoint, keeping a history of runs so the red→green trajectory of the gate is visible.

Verified facts the design builds on (from the commons-ip 2.x source and a real report in `basic-uuid/`):

- The commons-ip CLI **always exits 0**, even for an INVALID package — nonzero exits only signal operational errors (bad paths, unwritable report dir). The gate MUST parse the JSON report; `.summary.result == "VALID"` is the pass criterion (MUST-level failures only; SHOULD-level failures count as warnings).
- The CLI emits **JSON only** — no HTML. The HTML view is ours to generate.
- Report shape: top-level `header` (`date`, `path`, `version_commons_ip`, `specifications[]`), `summary` (`success`, `warnings`, `errors`, `skipped`, `notes`, `result`), `validation[]` (`specification`, `id`, `name`, `location`, `description`, `cardinality`, `level`, `testing: {outcome, issues[], warnings[], notes[]}`). Failure messages live in `testing.issues` / `testing.warnings`.
- Report filename: `<input>_validation-report_<yyyy-MM-dd>.json`, written into the `-o` dir.
- The local `~/bin/csip` wrapper pins jar **2.8.0**; latest stable release is **2.11.2**. The Docker image pins 2.11.2, replacing the wrapper. The jar upgrade may shift individual check outcomes — acceptable while the gate is red; the first published run records the new baseline.
- The gate **starts red by design**: the sample package is known-INVALID ([TODO.md](../TODO.md), "Known-INVALID") — the report showing exactly that is the point.

## Design

```
build.sh (local CI entrypoint, honest exit code)
  ├─ go build -o bin/sip-creator .
  ├─ regenerate sample SIP from ./tmp/basic → basic-uuid/
  ├─ scripts/validate.sh -o reports/runs/<ts> basic-uuid/uuid-*.zip
  │     └─ docker compose run --rm validator  (commons-ip 2.11.2, spec 2.2.0)
  │        writes <pkg>_validation-report_<date>.json, prints FAILED detail,
  │        exits 1 iff any package != VALID
  └─ scripts/publish-report.sh reports/runs/<ts>
        └─ rebuilds reports/runs.json (jq scan of all runs), copies the viewer

docker compose up -d reports   → nginx serves ./reports at http://localhost:8080
```

Division of labor: shell + jq only move and aggregate JSON; all presentation lives in one tracked static HTML file (`scripts/report/index.html`, vanilla JS, no build step, no CDN) rendered client-side from `runs.json` and the raw report JSONs. No HTML generation in shell.

Deliberate choices, on top of those already argued in the validation-workflow plan (zip-only gate, `VALID` criterion, `--specification-version=2.2.0` pinned, `bin/` for the binary, no Makefile):

- **Validator in Docker, jar pinned by version + sha256.** Reproducible on any machine with Docker; retires the hand-rolled `~/bin/csip` wrapper and its stale 2.8.0 jar. `validate.sh` accepts a `CSIP_CMD` override for running a host jar without Docker.
- **Run history under `reports/runs/<UTC timestamp>/`** with a generated `runs.json` index. History is cheap (a few JSON files per run) and makes the gate's trajectory visible; `reports/` is gitignored.
- **nginx serves static files only.** No server-side logic anywhere; the endpoint is a dev convenience on localhost:8080.

## Implementation steps

1. **`docker/validator/Dockerfile`** — `eclipse-temurin:17-jre`, download the pinned release jar `commons-ip2-cli-2.11.2.jar` at build time, verify its recorded sha256, `ENTRYPOINT ["java","-jar","/opt/commons-ip.jar"]`.
2. **`docker-compose.yml`** (repo root) — `validator` (build-only, invoked via `docker compose run --rm` with per-run volume mounts from `validate.sh`) and `reports` (`nginx:alpine`, mounts `./reports` read-only, port 8080).
3. **`scripts/validate.sh <sip.zip|sip-dir>...`** — the validation-workflow plan's Step 1 script with two adaptations: an `-o <dir>` option to keep reports for the HTML layer (default stays a cleaned-up temp dir), and the csip invocation goes through `docker compose run --rm validator` with the input's parent dir mounted read-only.
4. **`scripts/publish-report.sh <run-dir>`** — writes `<run-dir>/run.json` (per-package `{report_file, path, result, errors, warnings, ...}` via jq), regenerates `reports/runs.json` from all `reports/runs/*/run.json` (newest first), copies `scripts/report/index.html` → `reports/index.html`.
5. **`scripts/report/index.html`** — the viewer: run index with VALID/INVALID badges; per-run summary tiles; checks table (id, level, outcome, name, expandable description + issue messages), FAILED rows first, filters (All / FAILED / MUST); check ids link to the spec anchors (`https://earkcsip.dilcis.eu/#<id>`, `https://earksip.dilcis.eu/#<id>`).
6. **`build.sh`** — the validation-workflow plan's Step 2, extended: after validating into a fresh `reports/runs/<ts>/`, publish the report and exit with the validator's status.
7. **Housekeeping + docs in the same change** — `.gitignore` gains `bin/` and `reports/`; `README.md` and `CLAUDE.md` document the new loop (deps: Go, Docker, jq, `sf` — catmandu and the csip wrapper gone); new [ADR-0005](../decisions/0005-dockerized-validation-and-html-reporting.md) records the dockerized-validator + static-HTML-reporting decision; `TODO.md`'s "Validation workflow" line comes off the backlog (the Known-INVALID evidence stays — it is what the gate now reports).

## Commits

1. `Added: docker validator image (commons-ip 2.11.2) and compose file with nginx report server`
2. `Added: scripts/validate.sh — csip validation via docker with jq parsing and honest exit codes`
3. `Added: scripts/publish-report.sh and static HTML validation report viewer`
4. `Changed: build.sh — local CI loop: build, generate, validate, publish report` (docs, ADR-0005 and `.gitignore` ride along)

## Verification

1. `docker compose build validator` succeeds; `docker compose run --rm validator validate -h` prints usage.
2. **Red-gate path (expected today):** `./build.sh` prints the `INVALID` verdict and each FAILED check with its `issues`/`warnings` text, exits 1; a report JSON lands under `reports/runs/<ts>/`.
3. `docker compose up -d reports` → http://localhost:8080 lists the run with an INVALID badge; the detail view's FAILED rows match `jq '.validation[] | select(.testing.outcome=="FAILED")'` on the raw report.
4. **History:** two `./build.sh` runs produce two index entries; no stale-report bleed.
5. **Tamper test:** corrupt a `CHECKSUM` in a copied package's `METS.xml`; `scripts/validate.sh` on it goes INVALID with a fixity FAILED check. Also validate an unzipped package dir to prove the standalone debugging path.
6. **Hygiene:** `shellcheck build.sh scripts/*.sh` clean; `git status` shows no untracked binaries or reports; no `*validation-report*.json` outside `reports/` or temp dirs.
