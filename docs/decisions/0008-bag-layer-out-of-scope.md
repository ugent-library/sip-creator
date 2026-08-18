# 0008 — The bag layer is out of scope: BagIt is the transfer envelope

Status: **Accepted** (2026-07-17, with the [meemoo-1.2 plan](../archive/meemoo-12.md))

## Context

meemoo SIP 1.2 — the stable specification, and the one meemoo's production
ingest accepts — wraps the information package in a BagIt bag for transfer:
`bagit.txt`, payload manifests, and a `data/` directory containing the package.
Retargeting the basic profile at 1.2 therefore raised the question whether this
tool should produce bags.

Three facts shaped the answer:

1. **The bag is an envelope, not part of the package.** Everything this tool is
   about — METS, PREMIS, descriptive metadata, fixity, the CSIP shape — lives
   *inside* the bag's `data/`. The bag adds transfer-level integrity
   (manifests) and nothing archival. The meemoo 2.x drafts drop the bag layer
   entirely, confirming it as packaging, not content.
2. **Reference implementations exist and are the norm.** The Library of
   Congress maintains BagIt implementations (bagit-python, bagit-java);
   producing bags with the reference tool is how conformance is normally
   guaranteed. A homegrown Go bagger would be a wheel reinvented, then
   maintained for conformance forever — against the small-and-boring
   dependency rule either way (write it or import it).
3. **commons-ip2 draws the same line.** The E-ARK reference implementation has
   no "bag the SIP" capability; BagIt appears only in its legacy v1 namespace
   as a separate SIP *format*, never as an envelope for E-ARK packages.

This extends [ADR-0001](0001-package-builder-not-archive.md): the tool builds
one well-formed information package and stops. Transfer packaging is the step
after "stops".

## Decision

SIP Creator's output contract ends at the **package boundary**: the package
directory and, as a convenience, an uncompressed zip of it. It never produces
BagIt bags.

Bagging is an operator/pipeline step performed with a reference BagIt
implementation, e.g.:

```sh
sip-creator create --profile basic ./input ./dest
bagit.py --md5 dest/uuid-<uuid>/     # LOC reference implementation
# zip the bag, deliver per meemoo's transfer instructions
```

Deliverable semantics are per family: for eark/RODA the zip *is* the
deliverable; for meemoo-1.2 the package **directory** is the input to bagging
and our zip is at best unused (a `--no-zip` convenience flag is a plan item).
Validation splits along the same seam: this repo's gates validate the package;
`bagit --validate` (the reference tool) validates the envelope.

## Consequences

- No BagIt code or dependency enters the Go module.
- The operator workflow for meemoo delivery must be documented (README/runbook):
  bag the *directory*, not our zip — the mistake a novice would make.
- If meemoo tooling ever requires envelope properties beyond plain BagIt
  (bag-info fields, tag manifests), that is configuration of the external
  bagging step, still not code here.
- A future "why doesn't the tool just bag it?" feature request gets this ADR
  as the answer, including the commons-ip2 precedent.
