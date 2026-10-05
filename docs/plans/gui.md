# Plan: a desktop front end on Wails v2

*Status: **drafted** (2026-10-05), not started. A moonshot next to the CLI; no step has a
date. Each step starts on an explicit go, in one or two commits. Update this line as steps
land.*

## Context

Digital-preservation staff at UGent Library, on macOS and Windows, should be able to point
at an input folder prepared to the [input specification](../input-spec.md), check it, and
build an E-ARK SIP without a terminal. The CLI stays. The desktop app is a second front end
on the same library, so the package it writes is what `sip-creator create` writes for the
same folder and the same settings, byte for byte.

**Framework: Wails v2** (v2.14.0, released 2026-08-10, needs Go 1.25 or later; the
repository is on 1.27). A Go backend with the user interface in HTML and CSS, rendered by
the operating system's web view: WebKit on macOS, WebView2 on Windows. Reasons, in order:

1. Native folder dialogs on both systems. Pointing at a folder is the core interaction,
   and staff get the Finder or Explorer picker they know.
2. A plain HTML front end the project can write without npm or a bundler.
3. The Go side is the code `create` and `check` run today.

Rejected:

- **Fyne.** Its folder picker is a drawn widget, not the operating system's dialog, so
  navigating a deep network share is slow and unfamiliar; its dependency tree brings
  OpenGL and GLFW.
- **A web page served from the binary** (`net/http` and `html/template`, no dependency
  at all). A browser cannot hand the program a folder path, so the user would paste one.
- **Wails v3**, in beta since August 2026. v2 is the stable line. Moving to v3 is later,
  separate work.

## Shape

**A nested Go module at `gui/`**, not a package in the main module and not a separate
repository.

- Wails owns the directory it runs in: `wails.json`, `main.go`, `go.mod`, `frontend/` and
  a `build/` directory for icons, `Info.plist` and installer files. At the repository
  root that `build/` would collide with the library's `build` package. So the Wails
  project lives in `gui/` with its own `go.mod` (`github.com/ugent-library/sip-creator/gui`)
  and a `replace github.com/ugent-library/sip-creator => ../`, so it builds against the
  working tree. `go.work` is gitignored and stays local.
- Accepted consequence: `go test ./...` at the root does not enter `gui/`, so
  CONTRIBUTING.md adds `cd gui && go test ./...`. `go install` of the CLI is unaffected,
  and the main module gains no dependency on Wails.
- No node, no npm, no bundler. Wails's frontend commands stay empty
  (`"frontend:install": ""` and `"frontend:build": ""`, which `wails build` skips with
  "No Install command. Skipping."), and `assetdir` points at `gui/frontend/`: static
  `index.html`, `style.css` and `app.js`, embedded with `go:embed`. The bindings Wails
  generates in `frontend/wailsjs/` are committed, as Wails projects do.

**What the app does**, mirroring `create` and `check` in
[cli/create_cmd.go](../../cli/create_cmd.go) and [cli/check_cmd.go](../../cli/check_cmd.go):

1. **Settings pane**: submitter name, Meemoo OR-id, default content category, default
   destination directory. Stored under `os.UserConfigDir()/sip-creator/` under the
   variable names [CONFIG.md](../../CONFIG.md) documents, so one vocabulary describes
   both front ends. These values are configuration, not folder content
   ([ADR-0010](../decisions/0010-config-over-self-describing-input.md)).
2. **Main pane**: the profile (the registry's names, `profiles.Names()`), the input
   folder and the destination (native dialogs), and the optional record status, updated
   package identifier and content category. The rule that pairs the status with the
   identifier, `recordStatusFromFlags` in the CLI, applies here too.
3. **Check**: `input.Read` with the profile's mapper, then `Definition.ValidateSource`;
   every violation listed, as `check` prints them.
4. **Create**: `build.New`, `Builder.Build`, `archive.Zip`, with a progress view that
   advances per file, then the package path with an "open in Finder" or "open in
   Explorer" action.

Siegfried stays as it is ([ADR-0009](../decisions/0009-characterization-as-sidecar-input.md)):
a `siegfried.json` in the folder is used, none means no format info, and the app never
runs `sf`. A button that runs Siegfried is a separate decision for later.

## Steps

**Step 1: the decision in the repository.** An ADR (the next free number), "A desktop
front end on Wails v2, as a nested module": the dependency discussion CLAUDE.md asks for,
the `build/` collision, the no-npm choice. No code.

**Step 2: library pre-work the app needs.** All in the main module, each useful to the
CLI too. `scripts/reference-diff.sh` stays clean and `./build.sh` keeps reporting `VALID`
for every profile.

- **Per-file write progress.** The assembler logs every node
  ([build/assemble.go](../../build/assemble.go)), but the write phase, where the copying
  time goes, logs only "starting..." and "finished."
  ([build/builder.go](../../build/builder.go)). Add one structured log line per written
  file in [build/write.go](../../build/write.go), with the path and size as attributes,
  so a `slog.Handler` in the app drives the progress view without new exported API; the
  app knows the total from the `SourcePackage`. If that proves too thin, the fallback is
  a `Progress` callback on `build.Config`, discussed at that point.
- **The profile-to-mapper pairing.** The `mappers` table in
  [cli/profile.go](../../cli/profile.go) is unexported; the app needs the same table.
  Move the pairing into `cli/input/mapping` as an exported lookup by profile name, with
  the existing test that every registered profile has a mapper.
- **To settle at this step, not before:** whether `cli/input` and `cli/input/mapping`
  move to the top level, since a second front end will import them and the name under
  `cli/` then misleads. [ADR-0023](../decisions/0023-cli-input-one-package.md) and
  [ADR-0027](../decisions/0027-input-reader-walks-then-decodes.md) stay as written; the
  design doc and CLAUDE.md's package table would change.

**Step 3: the Wails project.** `gui/` from `wails init -t vanilla`, stripped of npm:
`go.mod` with the replace, `wails.json` with empty frontend commands, `main.go`, `app.go`
with the bound methods (`Profiles`, `ChooseFolder`, `ChooseDirectory`, `Check`, `Create`,
`Settings`, `SaveSettings`), `settings.go`, and the static `frontend/`. Every bound method
is a thin wrapper over a plain function that takes paths and values, so those functions
are tested with `go test` against `examples/` without the Wails runtime.

**Step 4: packaging.** `wails build -platform darwin/universal` on a Mac, and
`wails build -platform windows/amd64` cross-compiled from the same Mac, because Wails on
Windows needs no cgo. Icons and `Info.plist` in `gui/build/`. Signing and notarization
need an Apple Developer ID and a Windows certificate, an organizational question outside
this plan; until then the README says the binaries are unsigned.

**Step 5: docs**, in the same commits as the code they describe, per CONTRIBUTING.md:

- README.md gains a "Desktop app" section;
- [sip-creator-design.md](../sip-creator-design.md) gains the app under a "Front ends"
  heading next to the CLI/library boundary;
- CLAUDE.md's package table gains `gui/`;
- CONTRIBUTING.md gains the Wails CLI
  (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0`, `wails doctor`) and the
  second test command;
- this plan retires to `archive/` when the app ships, per the
  [docs lifecycle](../README.md#lifecycle-what-happens-when-a-plan-ships).

## Verification

- `go test ./...` at the root and `cd gui && go test ./...` both pass; the Go tests still
  need no Docker, `sf` or `.env`.
- `wails doctor` reports a working toolchain; `cd gui && wails build` produces
  `gui/build/bin/`.
- Launch the app, point it at an example folder, run check (expect OK with the
  representation and file counts), run create into a temporary directory, and validate
  the zip with `./scripts/validate.sh -s 2.2.0`. Repeat for the Meemoo profile with an
  OR-id set, and for the MODS profile.
- The decisive test: a package built by the app and one built by `./bin/sip-creator
  create` from the same folder, with the same submitter and content category, differ in
  nothing `./scripts/reference-diff.sh` does not normalize.
- Progress: build a folder with a few large files and confirm the progress view advances
  per file.
