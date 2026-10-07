# 0031 — A package directory or zip appears under its final name only when complete

Status: **Accepted** (2026-10-06). **Amended 2026-10-07:** for an update, `create` now checks for `dest/<identifier>.zip` before building (`archive.ValidateDestination`), so the last consequence below no longer holds: an old zip is refused before anything is written. A suffix on the new names (`uuid-X_1.zip`) was rejected: the directory would no longer be named after its `OBJID`, the number would say nothing about the record, and a workflow that picks up every `*.zip` would send each copy.

## Context

`Build` wrote straight into `dest/<identifier>/`, creating it if needed and writing into it if it existed. Two things went wrong with that. An update reuses the earlier package's identifier, and so its directory name: building a replacement into the destination that still held the original left the original's files beside the new ones. In a test with two builds of one identifier, `data/` held both images, the zip carried both, the new METS listed one, and commons-ip reported the package INVALID (CSIP66, files the METS does not reference); `create` reported success. And a build that failed while writing (a source file gone, a full disk, a permission) left a half-written package directory behind, under the name a consumer looks for. `archive.Zip` had the same two faults: it replaced an existing `<identifier>.zip` without a word, and a failed zip left a truncated file under the final name, where a workflow that picks up every `*.zip` would find it.

## Decision

The package directory and the zip appear under their final names only when complete, and an existing one is never written into or replaced.

- `Build` refuses when `dest/<identifier>` exists. It writes the package into `dest/.<identifier>.tmp`, renames that to `dest/<identifier>` once every file is written, and removes it when the write fails. A temporary directory left by a killed run is removed before writing, so none of its files reach the package.
- `archive.Zip` refuses when `dest/<identifier>.zip` exists. It writes `dest/.<identifier>.zip.tmp`, renames it once the zip is complete, and removes it when zipping fails. A temporary file left by a killed run is written over.

Both temporary names sit in the destination itself, so the rename stays on one file system, where it is atomic.

## Alternatives rejected

- **Check that every source file exists before writing**: it covers one cause of a failed write, not a full disk or a permission, and the copy's error already names the file.
- **Remove the package directory on failure, without a temporary name**: safe only if `Build` created the directory, and a reader could still see a half-written package while the build runs.
- **Replace an existing package directory or zip**: it may be one a transfer has picked up, or another build's; the tool does not delete what it did not write in this run.
- **`os.Link` instead of a check before the rename**, for the zip: it closes the gap between the check and the rename, but some file systems (FAT, some network shares) do not support it. For a CLI one operator runs, the gap is acceptable.

## Consequences

- A failed `Build` or zip leaves nothing under a final name; a process killed partway leaves only an entry whose name starts with a dot and ends in `.tmp`, which the next run with the same identifier clears.
- Building an update into the destination that still holds the original now fails with an error naming the directory; move the original away first.
- In the CLI, `create` builds the directory and then zips it. If only an old zip with the same identifier exists, `create` fails at the zip, after the directory is built.
