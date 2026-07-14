# Plan: streamline the validation workflow (build.sh)

*Status: shipped 2026-07-14 (as implemented and extended by [local-ci-pipeline-plan.md](local-ci-pipeline-plan.md)), retired to archive. The durable design lives in [sip-creator-design.md](../sip-creator-design.md) § Validation; the decisions in [ADR-0003](../decisions/0003-validation-stays-external.md) and [ADR-0005](../decisions/0005-dockerized-validation-and-html-reporting.md).*

## Context — why this change

`build.sh` is the project's acceptance check: generated packages must pass commons-ip CSIP validation. The agreed position is that the CSIP rules themselves stay in the `csip` tool — **no reimplementation of the validation spec as Go tests** (far too cumbersome; commons-ip is the reference validator). What needs fixing is the workflow around the tool. The current script:

```sh
rm -rf basic-uuid
go run main.go create --profile basic ./tmp/basic basic-uuid
cd basic-uuid
ls | grep uuid | xargs -n 1 -I FOO csip validate --inputs=FOO
catmandu convert JSON to CSV --fix "..." < <(jq .validation *.json | less) --fields outcome,id,level,notes | grep "FAILED"
```

Problems, in order of impact:

1. **No exit-code discipline.** The script always "succeeds"; worse, the final `grep "FAILED"` exits 0 when failures are *found* and 1 when the package is *clean* — inverted semantics. It cannot serve as an automated gate (pre-commit, CI, or the refactoring plan's phase gates) without a human reading the output.
2. **catmandu is a heavyweight dependency doing jq's job.** It's a full Perl/CPAN ETL toolkit pulled in solely to flatten a JSON array that `jq` (already required) flattens natively. The `< <(jq ... | less)` construction is also fragile (`less` in a non-interactive pipeline is a pass-through at best).
3. **Duplicate validation.** `ls | grep uuid` matches both the package *directory* and the *zip*, so every package is validated twice (~140 checks each), producing two near-identical reports.
4. **Report litter.** commons-ip writes `<input>_validation-report_<date>.json` next to the package; datestamped reports accumulate across runs and the glob `*.json` picks up stale ones.
5. **Failure output loses detail.** The catmandu pipeline surfaces `outcome,id,level,notes` but drops `testing.issues` — which is where commons-ip puts the actual error message (the report's `summary` distinguishes `errors` from `warnings`; a `SHOULD`-level FAILED is only a warning).

Verified report facts this plan builds on (from an existing report in `basic-uuid/`):

- Top-level keys: `header`, `summary`, `validation`.
- `.summary` = `{success, warnings, errors, skipped, notes, result: "VALID"|"INVALID"}` — the tool's own verdict; `errors` counts MUST-level failures only.
- `.validation[]` = `{specification, id, name, location, description, cardinality, level: MUST|SHOULD|MAY, testing: {outcome: PASSED|FAILED|SKIPPED, issues[], warnings[], notes[]}}`.
- `csip validate` supports `-o/--output-report-dir` (redirect the report), `--specification-version` (default 2.2.0), and multiple `-i` inputs.

## Design

Two small shell scripts, no new dependencies (net: **one dependency removed**). Required tools drop from `csip, jq, catmandu, sf` to `csip, jq, sf`.

- **`scripts/validate.sh <sip.zip|sip-dir>...`** — reusable validator: runs `csip validate` per input with the report redirected to a fresh temp dir (kills the stale-report problem), parses it with jq, prints a one-line verdict + full detail for every FAILED check (id, level, name, and the `issues`/`warnings`/`notes` messages), and **exits non-zero iff any input is not `VALID`**. Usable standalone against any package or zip — including debugging a package directory before zipping, which today's script only did by accident.
- **`build.sh`** — the dev loop, reduced to: compile, regenerate the sample SIP, validate **the zip only** (the zip is the deliverable meemoo ingests; validating the directory too is redundant — use `scripts/validate.sh <dir>` manually when debugging structure). Fails fast (`set -euo pipefail`) and exits with the validator's status.

Deliberate choices:

- **Pass/fail criterion is `.summary.result == "VALID"`** — the tool's own verdict, meaning no MUST-level failures. SHOULD/MAY failures are printed (they're in the FAILED listing with their level) but don't fail the build, matching commons-ip's own error/warning distinction. If a stricter gate is ever wanted, it's a one-line jq change (`.summary.warnings == 0`).
- **`--specification-version=2.2.0` pinned explicitly** so a csip upgrade changing its default can't silently move the goalposts.
- **`go build -o bin/sip-creator` instead of `go run`** — same compile check, and the binary lands in a gitignored `bin/` directory. `bin/` is the Go community convention for project-local builds (deliberately *not* `build/`, which in the common Go project layout holds CI/packaging config, not compiled output). This also closes a real `.gitignore` gap: the root `sip-creator` binary is currently not ignored at all (only `*.exe`/`*.dll`/`*.so`/`*.dylib` patterns are), so it can be committed by accident today.
- **No Makefile, no Go test wrapper** — per the project's small-and-boring dependency policy and the explicit decision to keep CSIP validation external.

## Implementation steps

### Step 1: add `scripts/validate.sh`

```sh
#!/usr/bin/env bash
# Validate SIP packages (zip or directory) with commons-ip.
# Prints FAILED checks with their messages; exits non-zero if any package is INVALID.
# Requires: csip, jq
set -euo pipefail

if [ $# -eq 0 ]; then
    echo "usage: $0 <sip.zip|sip-dir>..." >&2
    exit 2
fi

status=0
for input in "$@"; do
    report_dir="$(mktemp -d)"
    trap 'rm -rf "$report_dir"' EXIT

    csip validate --specification-version=2.2.0 -i "$input" -o "$report_dir" >/dev/null

    report="$(find "$report_dir" -name '*validation-report*.json' | head -n 1)"
    if [ -z "$report" ]; then
        echo "== $input: ERROR — csip produced no validation report" >&2
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

    rm -rf "$report_dir"
done

exit $status
```

`chmod +x scripts/validate.sh`. Implementation notes: confirm during the smoke test that `-o` into an empty temp dir yields exactly one report per input (that's why the loop runs csip once per input rather than passing multiple `-i` flags); if csip's stdout turns out to carry useful context on crashes, drop the `>/dev/null` in favor of capturing to a variable shown only on error.

### Step 2: rewrite `build.sh`

```sh
#!/usr/bin/env bash
# Dev loop: rebuild, regenerate the sample SIP from ./tmp/basic, validate with commons-ip.
# Requires: go, csip, jq on the PATH, and a configured .env (working Siegfried install).
set -euo pipefail
cd "$(dirname "$0")"

OUT=basic-uuid

go build -o bin/sip-creator .

rm -rf "$OUT"
./bin/sip-creator create --profile basic ./tmp/basic "$OUT"

exec ./scripts/validate.sh "$OUT"/uuid-*.zip
```

Ride-along in the same commit: add `bin/` to `.gitignore` (under the "Local fixtures and generated output" section), and delete the stray root `./sip-creator` binary from the working tree if present.

### Step 3: documentation, in the same change (per CLAUDE.md non-negotiables)

- **`CLAUDE.md`**: the `./build.sh` bullet under "Development commands" (currently line 71) — new behavior, dependencies now "`csip` and `jq` on the PATH" (catmandu removed, `go run` → `go build`), and add a bullet for `./scripts/validate.sh` as the standalone validator. Update every `./sip-creator` command reference (including the `go build` bullet) to `./bin/sip-creator` / `go build -o bin/sip-creator`.
- **`README.md`**: if/where the dev workflow is described, reflect the new loop and the `bin/` output path; no env vars change, so `CONFIG.md` is untouched.
- **`docs/refactoring-plan.md`** line 368 lists catmandu among build.sh requirements — update to match.

### Step 4 (optional, cheap while here): `scripts/diff-baseline.sh`

The refactoring plan's Phase 0 needs a normalized diff between a reference package tree and a fresh one (UUIDs and timestamps masked). Since this plan is already creating `scripts/`, add the ~10-line helper now so the refactor can use it as-is:

```sh
#!/usr/bin/env bash
# usage: diff-baseline.sh <ref-pkg-dir> <new-pkg-dir>
# Structural diff of two generated packages, UUIDs/timestamps normalized.
set -euo pipefail
norm() {
    local out; out="$(mktemp -d)"
    cp -R "$1/" "$out/pkg"
    find "$out" -name '*.xml' -exec sed -E -i '' \
        -e 's/uuid-[0-9a-f-]{36}/uuid-X/g' \
        -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.+-]+/TS/g' {} +
    echo "$out/pkg"
}
diff -r "$(norm "$1")" "$(norm "$2")"
```

Skippable if this plan should stay strictly scoped to validation.

## Automation — where this gate should (and shouldn't) run

The exit-code discipline above makes the check automatable. Sequencing, decided 2026-07-02:

**Not a pre-commit hook.** Rejected deliberately:
- Too slow for commit granularity: `go build` + Siegfried + a JVM startup for commons-ip is 15–30+ seconds, and the project's small-commits style multiplies the cost (a docs-only commit would pay it too).
- Machine-state dependent: needs `csip`, `jq`, `sf`, and a configured `.env` — a missing tool would masquerade as a failed gate.
- The gate is currently red: the sample package is known-INVALID (the DC `metadata` declaration and EDTF issues in [TODO.md](../TODO.md)). A blocking hook that starts red trains `--no-verify` reflexes.

**Stage 1 (this plan):** the one-command habit. `./build.sh && git commit …` when touching XML-generating code — the exit code is the automation; no infrastructure.

**Stage 2 (prerequisite: fix the two known FAILED checks so the gate is green):** a **pre-push** hook. Push frequency matches the check's meaning ("don't publish a state that produces invalid SIPs") and 30 seconds is acceptable there. Design constraints:
- Lives in a committed `.githooks/pre-push` with a one-time `git config core.hooksPath .githooks` per clone (hooks aren't tracked otherwise) — document in README.
- Path-filtered: only run when `*.go`, encoder templates, or `schemas/` changed relative to the remote ref; doc-only pushes stay free.
- Escape hatch by design: `git push --no-verify`.

**Stage 3 (end state): CI.** A GitHub Actions workflow that installs Go, Siegfried, and the commons-ip CLI jar, runs `./build.sh`, and fails on exit ≠ 0. Machine-independent, covers future collaborators, and the job definition is trivial once the script exits honestly. One-time setup cost: getting `csip`/`sf` into the runner and a fixture-safe `.env`.

Stages 2 and 3 are follow-ups, not part of this plan's implementation — they both depend on the gate being green first.

## Commits

Small and prefixed, matching history:

1. `Added: scripts/validate.sh — csip validation wrapper with jq report parsing and proper exit codes`
2. `Changed: build.sh — validate zips via scripts/validate.sh, drop catmandu, build binary into bin/` (the Step 3 docs edits and the `.gitignore` `bin/` entry ride along in this commit)
3. (optional) `Added: scripts/diff-baseline.sh for normalized package comparison`

## Verification

1. **Failure path (works today):** `./build.sh` against the current code — the sample package is known-INVALID (existing report: 1 error, 1 warning, 2 FAILED checks). Expect: verdict line `INVALID`, both FAILED checks printed with id/level/messages, **exit code 1** (`echo $?`).
2. **Detail fidelity:** the printed FAILED output must include the message text from `testing.issues` — compare against `jq '.validation[] | select(.testing.outcome=="FAILED")'` on the raw report.
3. **Standalone use:** `./scripts/validate.sh basic-uuid/uuid-*` (the directory) — validates a dir, proving the debugging path works.
4. **Tamper test:** copy a generated package, corrupt one `CHECKSUM` attribute in its `METS.xml`, run `scripts/validate.sh` on it — must go INVALID with a fixity-related FAILED check.
5. **No stale-report bleed:** run `build.sh` twice; the second run's output must not mention the first run's reports (temp-dir redirection).
6. **Hygiene:** `shellcheck scripts/validate.sh build.sh` clean; confirm `grep -ri catmandu` only hits historical docs, and that no `*validation-report*.json` files are left outside temp dirs.
7. **Binary placement:** after `./build.sh`, the binary exists at `bin/sip-creator`, no `sip-creator` sits at the repo root, and `git status` shows no untracked binaries (`bin/` ignored).
