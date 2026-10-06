# 0032 — A characterization report's checksum is taken as given

Status: **Accepted** (2026-10-06). Supersedes in part [ADR-0009](0009-characterization-as-sidecar-input.md): the report's MD5 no longer binds the report to the bytes on disk; it is the checksum the package declares.

## Context

Under ADR-0009 a build with a Siegfried report read every essence file twice. Assembly read it to compare its MD5 with the report's, so a stale report stopped the build before anything was written. The writer read it again to copy it into the package, computing the MD5 a second time. Siegfried had already read and hashed every file once to make the report.

Measured on 2026-10-06 (docs/TODO.md, under the library builder API item), MD5 and not the disk sets the build time. MD5 runs at about 0.6 GB/s on one core, and a plain copy of 32 GB takes 53 s, the time of one MD5 pass. The report check added 46 s, and the zip's two extra reads, with only CRC to compute, added 25 s. With a report, each byte of essence was hashed three times.

commons-ip 2, the E-ARK reference implementation, takes a checksum the caller supplies. When its algorithm matches the package's, commons-ip copies the file without computing a checksum and compares nothing else, such as the size. Only the METS files it generates are always hashed.

## Decision

A characterization report's MD5 is the checksum the package declares for every essence and documentation file the report has an entry for. The build does not read the file to check it, and the copy computes no checksum for it. Whether a report still describes the files is the operator's judgement about the report and the tool that made it, not the library's.

The build still refuses a report that cannot be used: an essence file without an entry, an entry with Siegfried's error, or an entry without an MD5 (a report made without `-hash md5`). Every file the report does not cover, and every document the build writes, gets a checksum computed from the bytes written, as before.

## Alternatives rejected

- **Keep ADR-0009's check before writing.** It costs a full read and a full MD5 pass of the essence, close to half the build time, to repeat what the operator's tool already did.
- **Check during the copy instead.** The copy would compare the MD5 it computes with the report's and fail the build on a difference. That removes the separate read but keeps an MD5 pass, which is the cost that matters. It also keeps the library judging the report, which this decision leaves to the operator.
- **Compare the report's file size with the bytes copied.** It would cost nothing and catch a replaced or truncated file. commons-ip compares nothing beyond the algorithm, and the operator owns the report's trustworthiness here as well.

## Consequences

- A build with a report and a zip is expected to take about a third of its earlier time; re-measure with the same input (docs/TODO.md).
- A file that changed after the report was made gets a checksum in METS and PREMIS that does not match its bytes. Nothing in the build notices. An ingest system that checks fixity rejects the package; one that does not stores a false fixity claim. The operator prevents this by making the report right before the build, from the input as it will be packaged.
- The format claims follow the same trust: a report made for other bytes gives them its formats.
- A missing source file is now found when the writer opens it, not at assembly. The build still fails and leaves nothing under the final name ([ADR-0031](0031-final-names-hold-complete-output.md)).
- This is the first case of the TODO's "pre-computed fixity" for the library: a program that holds checksums can supply them through the characterization report. A field on `SourceFile` for checksums from other sources would follow the same rule.
