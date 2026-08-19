package profiles

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

func fileMD5(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // test files are tiny
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func testDescriptive() metadata.Terms {
	return metadata.Terms{
		{Element: "dcterms:identifier", Value: "local-id-001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Catus Testus"},
	}
}

// writeEssence puts one content file on disk and returns its SourceFile.
func writeEssence(t *testing.T, dir, name, content string) SourceFile {
	t.Helper()
	src := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(src), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return SourceFile{Source: src, Key: name, Path: name}
}

// testFormat returns the canned format assertion the report-based tests use.
func testFormat() *sip.Format {
	fr := sip.NewFormatRegistry()
	fr.Name = "pronom"
	fr.Key = "fmt/999"
	return &sip.Format{FormatRegistry: fr}
}

// report builds a characterization report with a checksum-bound entry (and
// the canned format) for every given source file.
func report(t *testing.T, files ...SourceFile) characterization.Report {
	t.Helper()
	rep := make(characterization.Report, len(files))
	for _, f := range files {
		rep[f.Key] = characterization.Record{
			Format: testFormat(),
			Mime:   "image/test",
			MD5:    fileMD5(t, f.Source),
		}
	}
	return rep
}

// newTestBuilder returns a builder over the minimal valid input data: one
// representation with one essence file, descriptive terms, no report.
func newTestBuilder(t *testing.T) (b *Builder, in *Input, outDir string) {
	t.Helper()
	inDir, outDir := t.TempDir(), t.TempDir()
	cat := writeEssence(t, inDir, "cat.jpg", "not really a jpeg")

	in = &Input{
		Descriptive: testDescriptive(),
		Representations: []SourceRepresentation{
			{Label: "master", Files: []SourceFile{cat}},
		},
	}
	b = New(&Config{
		Destination: outDir,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return b, in, outDir
}

// basicDef returns the registered "basic" definition the tests build with.
func basicDef(t *testing.T) Definition {
	t.Helper()
	def, ok := Get("basic")
	if !ok {
		t.Fatal(`no "basic" definition registered`)
	}
	return def
}

func requireEmpty(t *testing.T, outDir string) {
	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the destination is not empty: %v", entries)
	}
}

func TestAssemble(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	in.Characterization = report(t, in.Representations[0].Files...)

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	// The package is rooted under the destination but nothing exists yet.
	if want := filepath.Join(outDir, pkg.Identifier); pkg.Location != want {
		t.Errorf("Location = %q, want %q", pkg.Location, want)
	}

	// Root entity wired, with the local identifier lifted onto it and the
	// entity identifier swapped into the description.
	e := pkg.Root
	if e == nil {
		t.Fatal("no root entity")
	}
	if got := e.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; got != "local-id-001" {
		t.Errorf("MEEMOO-LOCAL-ID = %q, want %q", got, "local-id-001")
	}
	d, ok := e.Description.(metadata.Terms)
	if !ok {
		t.Fatalf("Description is %T, want metadata.Terms", e.Description)
	}
	if got := d.GetLocalIdentifier("dcterms"); got != e.Identifier {
		t.Errorf("description identifier = %q, want entity identifier %q", got, e.Identifier)
	}

	// Description file node: the profile's declared name, package-relative path.
	df := e.DescriptionFile
	if df == nil {
		t.Fatal("no description file node")
	}
	if df.Name != "dc+schema.xml" {
		t.Errorf("description file Name = %q, want %q", df.Name, "dc+schema.xml")
	}
	if df.Path != "metadata/descriptive/dc+schema.xml" {
		t.Errorf("description file Path = %q, want %q", df.Path, "metadata/descriptive/dc+schema.xml")
	}
	if df.Mime != "text/xml" {
		t.Errorf("description file Mime = %q, want %q", df.Mime, "text/xml")
	}

	// One schema node per bundled XSD, in sorted (deterministic) order.
	if len(pkg.SchemaFiles) != len(schemas.Get()) {
		t.Errorf("schema nodes = %d, want %d", len(pkg.SchemaFiles), len(schemas.Get()))
	}
	names := make([]string, 0, len(pkg.SchemaFiles))
	for _, sf := range pkg.SchemaFiles {
		names = append(names, sf.Name)
		if sf.Path != "schemas/"+sf.Name {
			t.Errorf("schema Path = %q, want %q", sf.Path, "schemas/"+sf.Name)
		}
	}
	if !slices.IsSorted(names) {
		t.Errorf("schema nodes not sorted: %v", names)
	}

	// One representation: package-side name is the meemoo convention, not
	// the producer's label.
	if len(e.Representations) != 1 {
		t.Fatalf("representations = %d, want 1", len(e.Representations))
	}
	r := e.Representations[0]
	if r.Label != "representation_1" {
		t.Errorf("Label = %q, want %q", r.Label, "representation_1")
	}
	if r.Entity != e {
		t.Error("representation not wired back to the entity")
	}

	// The essence node records its source, a rep-relative path, and the
	// report's enrichment.
	if len(r.Files) != 1 {
		t.Fatalf("essence files = %d, want 1", len(r.Files))
	}
	f := r.Files[0]
	if f.Source != in.Representations[0].Files[0].Source {
		t.Errorf("Source = %q, want the supplied source path", f.Source)
	}
	if f.Path != "data/cat.jpg" {
		t.Errorf("Path = %q, want %q", f.Path, "data/cat.jpg")
	}
	if f.Representation != r {
		t.Error("essence file not wired back to the representation")
	}
	if f.Format == nil || f.Format.FormatRegistry.Key != "fmt/999" {
		t.Errorf("essence Format = %+v, want fmt/999 from the report", f.Format)
	}
	if f.Mime != "image/test" {
		t.Errorf("essence Mime = %q, want %q from the report", f.Mime, "image/test")
	}

	// PREMIS and METS nodes are the writer's to create, not the assembler's.
	if pkg.PremisFile != nil || pkg.MetsFile != nil || r.PremisFile != nil || r.MetsFile != nil {
		t.Error("assemble created premis/mets nodes; those belong to write")
	}

	// The load-bearing guarantee: assembly writes nothing.
	requireEmpty(t, outDir)
}

func TestAssembleRepresentations(t *testing.T) {
	b, in, _ := newTestBuilder(t)
	inDir := t.TempDir()

	// A second representation with a nested file: supplied order decides
	// the package-side numbering, nesting is preserved under data/.
	a := writeEssence(t, inDir, "a.jpg", "essence bytes")
	deep := writeEssence(t, inDir, "sub/deep.tif", "essence bytes")
	in.Representations = append(in.Representations,
		SourceRepresentation{Label: "access", Files: []SourceFile{a, deep}})

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	var labels []string
	for _, r := range pkg.Root.Representations {
		labels = append(labels, r.Label)
	}
	want := []string{"representation_1", "representation_2"}
	if !slices.Equal(labels, want) {
		t.Fatalf("representation labels = %v, want %v", labels, want)
	}

	rep2 := pkg.Root.Representations[1]
	if len(rep2.Files) != 2 {
		t.Fatalf("representation_2 files = %d, want 2", len(rep2.Files))
	}
	if got := rep2.Files[1].Path; got != "data/sub/deep.tif" {
		t.Errorf("nested file Path = %q, want %q (nesting preserved)", got, "data/sub/deep.tif")
	}
}

// Characterization is optional in contract (ADR-0009): no report means the
// build proceeds without format info.
func TestAssembleWithoutReport(t *testing.T) {
	b, in, _ := newTestBuilder(t)

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble without report: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format != nil {
		t.Errorf("essence Format = %+v, want nil without a report", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream without a report", f.Mime)
	}
}

// An entry with no match is an honest no-match: Format stays nil for that
// file only, and assembly succeeds.
func TestAssembleReportNoMatch(t *testing.T) {
	b, in, _ := newTestBuilder(t)
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {MD5: fileMD5(t, src.Source)},
	}

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble with no-match report: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format != nil {
		t.Errorf("essence Format = %+v, want nil on no match", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream on no match", f.Mime)
	}
}

// A match that asserts no mime still yields the Format, and the mime falls
// back to the admitted unknown — the two facts are independent.
func TestAssembleReportMatchWithoutMime(t *testing.T) {
	b, in, _ := newTestBuilder(t)
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {Format: testFormat(), MD5: fileMD5(t, src.Source)},
	}

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble with mimeless match: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format == nil || f.Format.FormatRegistry.Key != "fmt/999" {
		t.Errorf("essence Format = %+v, want fmt/999", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream when the match asserts none", f.Mime)
	}
}

// Essence the report doesn't know aborts: the file was added (or the report
// generated from the wrong directory) after the characterization run.
func TestAssembleReportMissingEntry(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	in.Characterization = characterization.Report{
		"somewhere/else.jpg": {MD5: "ab"},
	}

	if _, err := b.assemble(basicDef(t), in); err == nil {
		t.Fatal("assemble succeeded despite essence missing from the report")
	}
	requireEmpty(t, outDir)
}

// Changed bytes fail the MD5 binding: a stale report must never lend its
// format claims to different content.
func TestAssembleReportChecksumMismatch(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	src := in.Representations[0].Files[0]
	in.Characterization = report(t, src)
	if err := os.WriteFile(src.Source, []byte("different bytes now"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t), in); err == nil {
		t.Fatal("assemble succeeded despite essence changed since the report")
	}
	requireEmpty(t, outDir)
}

// A record without a checksum can't bind to bytes, so it aborts rather than
// being trusted.
func TestAssembleReportChecksumless(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {Format: testFormat(), Mime: "image/test"},
	}

	if _, err := b.assemble(basicDef(t), in); err == nil {
		t.Fatal("assemble succeeded despite a checksumless report entry")
	}
	requireEmpty(t, outDir)
}

// A per-file error recorded by the characterizer aborts: the tool is telling
// us it never characterized these bytes.
func TestAssembleReportEntryError(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {MD5: "ab", Errors: "permission denied"},
	}

	if _, err := b.assemble(basicDef(t), in); err == nil {
		t.Fatal("assemble succeeded despite a characterizer-reported file error")
	}
	requireEmpty(t, outDir)
}

// Documentation is lenient where essence is strict (ADR-0009): no entry is
// fine, a present entry enriches the mime but must be checksum-true.
func TestAssembleDocumentation(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	inDir := t.TempDir()
	manual := writeEssence(t, inDir, "manual.txt", "doc")
	notes := writeEssence(t, inDir, "sub/notes.txt", "doc")
	in.Documentation = []SourceFile{manual, notes}
	// The report knows the essence and one documentation file; the other
	// documentation file has no entry — allowed.
	in.Characterization = report(t, in.Representations[0].Files[0], manual)

	pkg, err := b.assemble(basicDef(t), in)
	if err != nil {
		t.Fatalf("assemble with documentation: %v", err)
	}

	if len(pkg.DocumentationFiles) != 2 {
		t.Fatalf("documentation nodes = %d, want 2", len(pkg.DocumentationFiles))
	}
	withEntry, withoutEntry := pkg.DocumentationFiles[0], pkg.DocumentationFiles[1]
	if withEntry.Path != "documentation/manual.txt" {
		t.Errorf("documentation Path = %q, want structure preserved", withEntry.Path)
	}
	if withEntry.Mime != "image/test" {
		t.Errorf("documentation Mime = %q, want the report's mime", withEntry.Mime)
	}
	if withoutEntry.Mime != "application/octet-stream" {
		t.Errorf("documentation Mime = %q, want octet-stream without an entry", withoutEntry.Mime)
	}
	requireEmpty(t, outDir)

	// A stale entry for a documentation file still aborts.
	if err := os.WriteFile(manual.Source, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.assemble(basicDef(t), in); err == nil {
		t.Fatal("assemble succeeded despite a stale documentation entry")
	}
}

// Input.Validate is the embedding-caller guardrail: the graph rules the
// folder convention enforces with Violations, re-checked for every producer.
func TestInputValidate(t *testing.T) {
	valid := func(t *testing.T) *Input {
		_, in, _ := newTestBuilder(t)
		return in
	}

	tests := []struct {
		name   string
		break_ func(*Input)
		want   string
	}{
		{"no descriptive", func(c *Input) { c.Descriptive = nil }, "no descriptive metadata"},
		{"invalid term", func(c *Input) {
			c.Descriptive = append(c.Descriptive, metadata.Term{Element: "dcterms:titel", Value: "x"})
		}, "not a Dublin Core term"},
		{"no identifier", func(c *Input) {
			c.Descriptive = metadata.Terms{{Element: "dcterms:title", Value: "x"}}
		}, "no dcterms:identifier"},
		{"no title", func(c *Input) {
			c.Descriptive = metadata.Terms{{Element: "dcterms:identifier", Value: "x"}}
		}, "no dcterms:title"},
		{"no representations", func(c *Input) { c.Representations = nil }, "at least one version"},
		{"bad label", func(c *Input) { c.Representations[0].Label = "master copy" }, "may only contain"},
		{"duplicate label", func(c *Input) {
			c.Representations = append(c.Representations, c.Representations[0])
		}, "supplied twice"},
		{"empty representation", func(c *Input) { c.Representations[0].Files = nil }, "no content files"},
		{"duplicate logical path", func(c *Input) {
			c.Representations[0].Files = append(c.Representations[0].Files, c.Representations[0].Files[0])
		}, "share the logical path"},
		{"file without source", func(c *Input) {
			c.Representations[0].Files[0].Source = ""
		}, "needs both a Source and a Path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid(t)
			tt.break_(in)
			err := in.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}

	if err := valid(t).Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

// Build refuses invalid input data before any side effect: the negative
// twin of the embedding-caller contract.
func TestBuildInvalidConfigWritesNothing(t *testing.T) {
	b, in, outDir := newTestBuilder(t)
	in.Representations = nil

	if _, err := b.Build(basicDef(t), in); err == nil {
		t.Fatal("Build succeeded on an invalid config")
	}
	requireEmpty(t, outDir)
}
