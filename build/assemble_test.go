package build_test

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

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
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

// testDescription satisfies the strictest registered profile: meemoo's
// four required elements, Dutch entries on the lang-tagged ones. It is the
// basic profile's input; eark tests swap in identityTerms.
func testDescription() meemoo.Terms {
	return meemoo.Terms{
		{Key: "identifier", Value: "local-id-001"},
		{Key: "title", Lang: "nl", Value: "Catus Testus"},
		{Key: "description", Lang: "nl", Value: "Een testkat"},
		{Key: "created", Value: "2026"},
	}
}

// writeEssence puts one content file on disk and returns its build.SourceFile.
func writeEssence(t *testing.T, dir, name, content string) build.SourceFile {
	t.Helper()
	src := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(src), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return build.SourceFile{Source: src, Key: name, Path: name}
}

// testFormat returns the canned format assertion the report-based tests use.
func testFormat() *sip.Format {
	fr := sip.NewFormatRegistry()
	fr.Name = "pronom"
	fr.Key = "fmt/999"
	return &sip.Format{FormatRegistry: fr}
}

// report builds a characterization report with an entry (matching checksum,
// canned format) for every given source file.
func report(t *testing.T, files ...build.SourceFile) characterization.Report {
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

// newTestBuilder returns a builder for def over the minimal valid input
// data: one representation with one essence file, descriptive terms, no
// report.
func newTestBuilder(t *testing.T, def build.Definition) (b *build.Builder, in *build.SourcePackage, outDir string) {
	t.Helper()
	inDir, outDir := t.TempDir(), t.TempDir()
	cat := writeEssence(t, inDir, "cat.jpg", "not really a jpeg")

	in = &build.SourcePackage{
		Description: testDescription(),
		Representations: []build.SourceRepresentation{
			{Name: "master", Files: []build.SourceFile{cat}},
		},
	}
	b, err := build.New(&build.Config{
		Profile:     def,
		Destination: outDir,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return b, in, outDir
}

// basicDef returns the registered "basic" definition the tests build with.
func basicDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("basic")
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
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.Characterization = report(t, in.Representations[0].Files...)

	pkg, err := b.Assemble(in)
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
	if got := identifierTerm(e.Description); got != e.Identifier {
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

	// One schema node per distinct XSD the package's documents point at
	// (the METS list plus the descriptive encoder's), in sorted
	// (deterministic) order.
	names := make([]string, 0, len(pkg.SchemaFiles))
	for _, sf := range pkg.SchemaFiles {
		names = append(names, sf.Name)
		if sf.Path != "schemas/"+sf.Name {
			t.Errorf("schema Path = %q, want %q", sf.Path, "schemas/"+sf.Name)
		}
	}
	referenced := slices.Concat(mets.Schemas, basicDef(t).Encoder.Schemas())
	if want := slices.Compact(slices.Sorted(slices.Values(referenced))); !slices.Equal(names, want) {
		t.Errorf("schema nodes = %v, want the referenced XSDs sorted and deduplicated: %v", names, want)
	}

	// One representation: the package-side name is the producer's label,
	// used verbatim.
	if len(e.Representations) != 1 {
		t.Fatalf("representations = %d, want 1", len(e.Representations))
	}
	r := e.Representations[0]
	if r.Name != "master" {
		t.Errorf("Name = %q, want the producer label %q", r.Name, "master")
	}
	if r.Label != "master" {
		t.Errorf("Label = %q, want the producer label %q", r.Label, "master")
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

	// Assembly leaves the graph complete: the generated PREMIS and METS
	// nodes are created here too (basic emits both PREMIS files), each with
	// its Path declared; the writer only emits and back-fills.
	if pkg.PremisFile == nil || pkg.MetsFile == nil || r.PremisFile == nil || r.MetsFile == nil {
		t.Fatal("assemble left generated premis/mets nodes missing; the graph must be complete before write")
	}
	if got := pkg.MetsFile.Path; got != "METS.xml" {
		t.Errorf("package METS Path = %q, want %q", got, "METS.xml")
	}
	if got := pkg.PremisFile.Path; got != "metadata/preservation/premis.xml" {
		t.Errorf("package PREMIS Path = %q, want %q", got, "metadata/preservation/premis.xml")
	}
	if got := r.MetsFile.Path; got != "representations/master/METS.xml" {
		t.Errorf("representation METS Path = %q, want %q", got, "representations/master/METS.xml")
	}
	if got := r.PremisFile.Path; got != "metadata/preservation/premis.xml" {
		t.Errorf("representation PREMIS Path = %q, want %q", got, "metadata/preservation/premis.xml")
	}

	// The core guarantee: assembly writes nothing.
	requireEmpty(t, outDir)
}

func TestAssemblePremislessProfile(t *testing.T) {
	def := basicDef(t)
	def.EmitPackagePremis = false
	def.EmitRepresentationPremis = false
	b, in, _ := newTestBuilder(t, def)

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	r := pkg.Root.Representations[0]
	if pkg.PremisFile != nil || r.PremisFile != nil {
		t.Error("premis nodes created although the profile emits no PREMIS")
	}
	if pkg.MetsFile == nil || r.MetsFile == nil {
		t.Error("METS nodes must be created regardless of the PREMIS emission flags")
	}
}

func TestAssembleRepresentations(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))
	inDir := t.TempDir()

	// A second representation with a nested file: each package-side name is
	// the producer's label, nesting is preserved under data/.
	a := writeEssence(t, inDir, "a.jpg", "essence bytes")
	deep := writeEssence(t, inDir, "sub/deep.tif", "essence bytes")
	in.Representations = append(in.Representations,
		build.SourceRepresentation{Name: "access", Files: []build.SourceFile{a, deep}})

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	var names []string
	for _, r := range pkg.Root.Representations {
		names = append(names, r.Name)
	}
	want := []string{"master", "access"}
	if !slices.Equal(names, want) {
		t.Fatalf("representation names = %v, want %v", names, want)
	}

	rep2 := pkg.Root.Representations[1]
	if len(rep2.Files) != 2 {
		t.Fatalf("representation %q files = %d, want 2", rep2.Name, len(rep2.Files))
	}
	if got := rep2.Files[1].Path; got != "data/sub/deep.tif" {
		t.Errorf("nested file Path = %q, want %q (nesting preserved)", got, "data/sub/deep.tif")
	}
}

// Characterization is optional in contract (ADR-0009): no report means the
// build proceeds without format info.
func TestAssembleWithoutReport(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))

	pkg, err := b.Assemble(in)
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

// An entry with no match is a genuine no-match: Format stays nil for that
// file only, and assembly succeeds.
func TestAssembleReportNoMatch(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {MD5: fileMD5(t, src.Source)},
	}

	pkg, err := b.Assemble(in)
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
// back to the admitted unknown; the two facts are independent.
func TestAssembleReportMatchWithoutMime(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {Format: testFormat(), MD5: fileMD5(t, src.Source)},
	}

	pkg, err := b.Assemble(in)
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
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.Characterization = characterization.Report{
		"somewhere/else.jpg": {MD5: "ab"},
	}

	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble succeeded despite essence missing from the report")
	}
	requireEmpty(t, outDir)
}

// Changed bytes fail the MD5 check: a stale report must never lend its
// format claims to different content.
func TestAssembleReportChecksumMismatch(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = report(t, src)
	if err := os.WriteFile(src.Source, []byte("different bytes now"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble succeeded despite essence changed since the report")
	}
	requireEmpty(t, outDir)
}

// A record without a checksum can't be verified against the bytes, so it
// aborts rather than being trusted.
func TestAssembleReportChecksumless(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {Format: testFormat(), Mime: "image/test"},
	}

	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble succeeded despite a checksumless report entry")
	}
	requireEmpty(t, outDir)
}

// A per-file error recorded by the characterizer aborts: the tool is telling
// us it never characterized these bytes.
func TestAssembleReportEntryError(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		src.Key: {MD5: "ab", Errors: "permission denied"},
	}

	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble succeeded despite a characterizer-reported file error")
	}
	requireEmpty(t, outDir)
}

// Documentation needs no characterization entry (ADR-0009): no entry is
// fine, a present entry enriches the mime but its checksum must match.
func TestAssembleDocumentation(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	inDir := t.TempDir()
	manual := writeEssence(t, inDir, "manual.txt", "doc")
	notes := writeEssence(t, inDir, "sub/notes.txt", "doc")
	in.Documentation = []build.SourceFile{manual, notes}
	// The report knows the essence and one documentation file; the other
	// documentation file has no entry, which is allowed.
	in.Characterization = report(t, in.Representations[0].Files[0], manual)

	pkg, err := b.Assemble(in)
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
	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble succeeded despite a stale documentation entry")
	}
}

// build.SourcePackage.Validate is the embedding-caller guardrail: the graph rules the
// folder convention enforces with Violations, re-checked for every producer.
func TestSourcePackageValidate(t *testing.T) {
	valid := func(t *testing.T) *build.SourcePackage {
		_, in, _ := newTestBuilder(t, basicDef(t))
		return in
	}

	tests := []struct {
		name   string
		break_ func(*build.SourcePackage)
		want   string
	}{
		{"no descriptive", func(c *build.SourcePackage) { c.Description = nil }, "no descriptive metadata"},
		{"invalid term", func(c *build.SourcePackage) {
			c.Description = append(c.Description.(meemoo.Terms), sip.Term{Key: "titel", Value: "x"})
		}, "not in the descriptive vocabulary"},
		{"no identifier", func(c *build.SourcePackage) {
			c.Description = meemoo.Terms{{Key: "title", Value: "x"}}
		}, "identifier is required"},
		{"no title", func(c *build.SourcePackage) {
			c.Description = meemoo.Terms{{Key: "identifier", Value: "x"}}
		}, "title is required"},
		{"no representations", func(c *build.SourcePackage) { c.Representations = nil }, "at least one version"},
		{"bad name", func(c *build.SourcePackage) { c.Representations[0].Name = "master copy" }, "may only contain"},
		{"xml-unsafe label", func(c *build.SourcePackage) { c.Representations[0].Label = `Master "scan"` }, "cannot be emitted"},
		{"xml-unsafe type", func(c *build.SourcePackage) { c.Representations[0].Type = "a<b" }, "cannot be emitted"},
		{"duplicate label", func(c *build.SourcePackage) {
			c.Representations = append(c.Representations, c.Representations[0])
		}, "supplied twice"},
		{"empty representation", func(c *build.SourcePackage) { c.Representations[0].Files = nil }, "no content files"},
		{"duplicate logical path", func(c *build.SourcePackage) {
			c.Representations[0].Files = append(c.Representations[0].Files, c.Representations[0].Files[0])
		}, "share the logical path"},
		{"file without source", func(c *build.SourcePackage) {
			c.Representations[0].Files[0].Source = ""
		}, "needs both a Source and a Path"},
		{"malformed package identifier", func(c *build.SourcePackage) {
			c.PackageIdentifier = "not-a-uuid"
		}, "uuid-<uuid> form"},
		{"record status outside the vocabulary", func(c *build.SourcePackage) {
			c.RecordStatus = "supplement"
		}, "SIP3 vocabulary"},
		{"update status without the updated package's identifier", func(c *build.SourcePackage) {
			c.RecordStatus = "REPLACEMENT"
		}, "PackageIdentifier must carry"},
		{"xml-unsafe content category", func(c *build.SourcePackage) {
			c.ContentCategory = "a<b"
		}, "cannot be emitted"},
		{"invalid representation descriptive", func(c *build.SourcePackage) {
			c.Representations[0].Description = meemoo.Terms{{Key: "titel", Value: "x"}}
		}, "not in the descriptive vocabulary"},
		{"received premis claims the generated name", func(c *build.SourcePackage) {
			c.Premis = []build.SourceFile{{Source: "/x/premis.xml", Path: "premis.xml"}}
		}, "reserved for the generated"},
		{"rep received premis claims the generated name", func(c *build.SourcePackage) {
			c.Representations[0].Premis = []build.SourceFile{{Source: "/x/premis.xml", Path: "sub/premis.xml"}}
		}, "reserved for the generated"},
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

// A representation may carry its own descriptive terms: they
// land on the representation with a rep-relative file node, an identifier
// term is swapped for the representation identifier, and identity is not
// required.
func TestAssembleRepresentationDescriptive(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))
	in.Representations[0].Description = meemoo.Terms{
		{Key: "license", Value: "publiek domein"},
	}

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	r := pkg.Root.Representations[0]
	if r.Description == nil {
		t.Fatal("representation descriptive not assembled")
	}
	df := r.DescriptionFile
	if df == nil {
		t.Fatal("no representation description file node")
	}
	if df.Path != "metadata/descriptive/dc+schema.xml" {
		t.Errorf("Path = %q, want rep-relative %q", df.Path, "metadata/descriptive/dc+schema.xml")
	}

	// With an identifier term present, the representation identifier is
	// swapped in, mirroring the package-level behavior.
	b2, in2, _ := newTestBuilder(t, basicDef(t))
	in2.Representations[0].Description = meemoo.Terms{
		{Key: "identifier", Value: "rep-local-1"},
	}
	pkg2, err := b2.Assemble(in2)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	r2 := pkg2.Root.Representations[0]
	if got := identifierTerm(r2.Description); got != r2.Identifier {
		t.Errorf("rep descriptive identifier = %q, want the representation identifier %q", got, r2.Identifier)
	}

	// Without rep terms, no node exists.
	b3, in3, _ := newTestBuilder(t, basicDef(t))
	pkg3, err := b3.Assemble(in3)
	if err != nil {
		t.Fatal(err)
	}
	if pkg3.Root.Representations[0].DescriptionFile != nil {
		t.Error("description file node created for a representation without terms")
	}
}

// identifierTerm returns the value of the identifier term of either
// world's terms ("" when absent): what the meemoo swap wrote, or what the
// eark profile left alone. Neither profile package exports an accessor for
// it; the swap is the meemoo package's own business, and the eark profile
// never swaps.
func identifierTerm(d sip.Description) string {
	var terms []sip.Term
	switch v := d.(type) {
	case meemoo.Terms:
		terms = v
	case eark.Terms:
		terms = v
	}
	for _, term := range terms {
		if term.Key == "identifier" {
			return term.Value
		}
	}
	return ""
}

// A package's record status and content category are its own, supplied on
// the source package: they land on the package declaration and on each
// representation's, and the profile's declaration stays as it was.
func TestAssembleDeclaresPackageValues(t *testing.T) {
	before := basicDef(t).Declaration
	b, in, _ := newTestBuilder(t, basicDef(t))
	in.PackageIdentifier = "uuid-3f2c1d0e-1111-4222-8333-444455556666"
	in.RecordStatus = "REPLACEMENT"
	in.ContentCategory = "Textual works – Print"

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if pkg.Declaration.RecordStatus != "REPLACEMENT" || pkg.Declaration.Type != "Textual works – Print" {
		t.Errorf("package declaration = %q/%q, want the source package's status and category", pkg.Declaration.RecordStatus, pkg.Declaration.Type)
	}
	rd := pkg.Root.Representations[0].Declaration
	if rd.RecordStatus != "REPLACEMENT" || rd.Type != "Textual works – Print" {
		t.Errorf("representation declaration = %q/%q, want the package's", rd.RecordStatus, rd.Type)
	}
	if after := basicDef(t).Declaration; after.RecordStatus != before.RecordStatus || after.Type != before.Type {
		t.Errorf("profile declaration changed: %q/%q", after.RecordStatus, after.Type)
	}
}

// The eark profile keeps the producer's identifier in the descriptive
// terms, at both levels, and lifts no MEEMOO-LOCAL-ID onto the entity: its
// standard has no swap (ADR-0012).
func TestAssembleEarkKeepsProducerIdentifier(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = eark.Terms{
		{Key: "identifier", Value: "rep-local-1"},
	}

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	e := pkg.Root
	if got := identifierTerm(e.Description); got != "local-id-001" {
		t.Errorf("description identifier = %q, want the producer's %q", got, "local-id-001")
	}
	if _, ok := e.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
		t.Error("MEEMOO-LOCAL-ID lifted onto the entity; it is a meemoo concept")
	}
	if got := identifierTerm(e.Representations[0].Description); got != "rep-local-1" {
		t.Errorf("rep descriptive identifier = %q, want the producer's %q", got, "rep-local-1")
	}
}

// The eark profile types each representation METS by its label, in both the
// TYPE and the CONTENTINFORMATIONTYPE pair; the basic profile keeps the
// profile declaration unchanged; the package declaration never changes
// (ADR-0013).
func TestAssembleRepresentationDeclaration(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	decl := pkg.Root.Representations[0].Declaration
	if decl.Type != "Other" || decl.OtherType != "master" {
		t.Errorf("rep TYPE = %q/%q, want Other/master", decl.Type, decl.OtherType)
	}
	if decl.ContentInformationType != "OTHER" || decl.OtherContentInformationType != "master" {
		t.Errorf("rep CONTENTINFORMATIONTYPE = %q/%q, want OTHER/master",
			decl.ContentInformationType, decl.OtherContentInformationType)
	}
	if pkg.Declaration.Type != "Mixed" || pkg.Declaration.OtherType != "" {
		t.Errorf("package TYPE = %q/%q, want Mixed with no OTHERTYPE",
			pkg.Declaration.Type, pkg.Declaration.OtherType)
	}

	b2, in2, _ := newTestBuilder(t, basicDef(t))
	pkg2, err := b2.Assemble(in2)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	decl2, pkgDecl := pkg2.Root.Representations[0].Declaration, pkg2.Declaration
	if decl2.Type != pkgDecl.Type || decl2.OtherType != pkgDecl.OtherType ||
		decl2.ContentInformationType != pkgDecl.ContentInformationType ||
		decl2.OtherContentInformationType != pkgDecl.OtherContentInformationType {
		t.Errorf("basic rep content typing differs from the package declaration:\n%+v\n%+v", decl2, pkgDecl)
	}
}

// Label and type resolve along the name → label → type cascade, and an
// explicit type reaches the eark representation declaration.
func TestAssembleRepresentationCascade(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	inDir := t.TempDir()
	in.Representations = []build.SourceRepresentation{
		{Name: "master", Label: "Master scan", Type: "archival",
			Files: []build.SourceFile{writeEssence(t, inDir, "a.tiff", "a")}},
		{Name: "access", Label: "Access copy",
			Files: []build.SourceFile{writeEssence(t, inDir, "b.pdf", "b")}},
		{Name: "preservation",
			Files: []build.SourceFile{writeEssence(t, inDir, "c.tiff", "c")}},
	}

	in.Description = identityTerms()
	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	reps := pkg.Root.Representations
	want := []struct{ name, label, typ string }{
		{"master", "Master scan", "archival"},            // everything explicit
		{"access", "Access copy", "Access copy"},         // type defaults to the label
		{"preservation", "preservation", "preservation"}, // both default to the name
	}
	for i, w := range want {
		r := reps[i]
		if r.Name != w.name || r.Label != w.label {
			t.Errorf("rep %d = %q/%q, want name %q label %q", i, r.Name, r.Label, w.name, w.label)
		}
		if got := r.Declaration.OtherType; got != w.typ {
			t.Errorf("rep %q OTHERTYPE = %q, want %q", w.name, got, w.typ)
		}
		if got := r.Declaration.OtherContentInformationType; got != w.typ {
			t.Errorf("rep %q OTHERCONTENTINFORMATIONTYPE = %q, want %q", w.name, got, w.typ)
		}
	}
}

const validPremis = `<?xml version="1.0"?><premis:premis xmlns:premis="http://www.loc.gov/premis/v3" version="3.0"><premis:event/></premis:premis>`

// Received preservation files become graph nodes at both levels (copied,
// never parsed) and must actually be premis:premis documents.
func TestAssembleReceivedPremis(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	inDir := t.TempDir()
	pkgPremis := writeEssence(t, inDir, "vendor.xml", validPremis)
	repPremis := writeEssence(t, inDir, "scanner/ocr.xml", validPremis)
	in.Premis = []build.SourceFile{pkgPremis}
	in.Representations[0].Premis = []build.SourceFile{repPremis}

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if len(pkg.ReceivedPremisFiles) != 1 {
		t.Fatalf("package received premis nodes = %d, want 1", len(pkg.ReceivedPremisFiles))
	}
	if got := pkg.ReceivedPremisFiles[0].Path; got != "metadata/preservation/vendor.xml" {
		t.Errorf("package received Path = %q", got)
	}
	r := pkg.Root.Representations[0]
	if len(r.ReceivedPremisFiles) != 1 {
		t.Fatalf("rep received premis nodes = %d, want 1", len(r.ReceivedPremisFiles))
	}
	if got := r.ReceivedPremisFiles[0].Path; got != "metadata/preservation/scanner/ocr.xml" {
		t.Errorf("rep received Path = %q (nesting preserved)", got)
	}
	if got := r.ReceivedPremisFiles[0].Mime; got != "text/xml" {
		t.Errorf("received Mime = %q", got)
	}

	// PremisFiles feeds the METS amdSec: the generated node first (created
	// at assembly, basic emits it), the received one after.
	if files := r.PremisFiles(); len(files) != 2 || files[0] != r.PremisFile || files[1] != r.ReceivedPremisFiles[0] {
		t.Errorf("PremisFiles = %v, want [generated, received]", files)
	}
	requireEmpty(t, outDir)
}

// Representation documentation gets the same treatment as package
// documentation: nodes under documentation/, no characterization entry
// required.
func TestAssembleRepresentationDocumentation(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	inDir := t.TempDir()
	note := writeEssence(t, inDir, "sub/scan-notes.txt", "doc")
	in.Representations[0].Documentation = []build.SourceFile{note}

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	r := pkg.Root.Representations[0]
	if len(r.DocumentationFiles) != 1 {
		t.Fatalf("rep documentation nodes = %d, want 1", len(r.DocumentationFiles))
	}
	f := r.DocumentationFiles[0]
	if f.Path != "documentation/sub/scan-notes.txt" {
		t.Errorf("Path = %q (nesting preserved under documentation/)", f.Path)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("Mime = %q, want octet-stream without a report entry", f.Mime)
	}
	requireEmpty(t, outDir)
}

// A received file that is not a PREMIS document aborts assembly: packaging
// it under metadata/preservation/ would be a false preservation claim.
func TestAssembleReceivedPremisRejectsNonPremis(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	inDir := t.TempDir()
	bad := writeEssence(t, inDir, "vendor.xml", "not xml at all")
	in.Premis = []build.SourceFile{bad}

	if _, err := b.Assemble(in); err == nil {
		t.Fatal("assemble accepted a non-PREMIS received file")
	}
	requireEmpty(t, outDir)
}

// A supplied package identifier is reused verbatim (how an update keeps
// the original package's mets/@OBJID); empty means mint.
func TestAssemblePackageIdentifier(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.PackageIdentifier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if pkg.Identifier != in.PackageIdentifier {
		t.Errorf("Identifier = %q, want the supplied %q", pkg.Identifier, in.PackageIdentifier)
	}
	if want := filepath.Join(outDir, in.PackageIdentifier); pkg.Location != want {
		t.Errorf("Location = %q, want %q", pkg.Location, want)
	}
}

// Build refuses invalid input data before any side effect: the negative
// twin of the embedding-caller contract.
func TestBuildInvalidConfigWritesNothing(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.Representations = nil

	if _, err := b.Build(in); err == nil {
		t.Fatal("Build succeeded on an invalid config")
	}
	requireEmpty(t, outDir)
}

// otherDescription stands in for a description of a standard no registered
// profile writes.
type otherDescription struct{}

func (otherDescription) Validate() error         { return nil }
func (otherDescription) ValidateRequired() error { return nil }

// A description of another standard is refused by the profile's
// descriptive-standard check before validation and before any side effect,
// at package and representation level alike: a type no profile writes,
// meemoo terms handed to eark, Simple DC terms handed to basic.
func TestBuildRejectsDescriptionOfAnotherStandard(t *testing.T) {
	cases := []struct {
		name string
		def  build.Definition
		desc sip.Description
		want string
	}{
		{"unknown type to eark", earkDef(t), otherDescription{}, "eark.Terms"},
		{"meemoo terms to eark", earkDef(t), testDescription(), "meemoo.Terms, not Simple Dublin Core"},
		{"simple dc terms to basic", basicDef(t), identityTerms(), "eark.Terms, not meemoo dc+schema"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, c.def)
			in.Description = c.desc
			_, err := b.Build(in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Build error = %v, want the mismatch mentioning %q", err, c.want)
			}
			requireEmpty(t, outDir)
		})
	}

	b, in, outDir := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = otherDescription{}
	_, err := b.Build(in)
	if err == nil || !strings.Contains(err.Error(), `representation "master"`) {
		t.Fatalf("Build error = %v, want the mismatch naming the representation", err)
	}
	requireEmpty(t, outDir)
}

// unbundledSchemas wraps a real encoder and claims an XSD the bundle does
// not hold.
type unbundledSchemas struct{ build.DescriptionEncoder }

func (unbundledSchemas) Schemas() []string { return []string{"nope.xsd"} }

// An encoder listing a schema the bundle does not hold is refused at
// assembly, before any write: the alternative is an empty XSD in the
// package.
func TestBuildRefusesUnbundledSchema(t *testing.T) {
	def := basicDef(t)
	def.Encoder = unbundledSchemas{def.Encoder}
	b, in, outDir := newTestBuilder(t, def)
	_, err := b.Build(in)
	if err == nil || !strings.Contains(err.Error(), `"nope.xsd"`) {
		t.Fatalf("Build error = %v, want the unbundled schema named", err)
	}
	requireEmpty(t, outDir)
}

// A definition built outside the registry names no descriptive encoder
// and is refused when the builder is constructed, as an error rather than
// a panic, so no build can start from it.
func TestNewRefusesDefinitionWithoutEncoder(t *testing.T) {
	def := basicDef(t)
	def.Encoder = nil
	_, err := build.New(&build.Config{
		Profile:     def,
		Destination: t.TempDir(),
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("New accepted a definition without a descriptive standard")
	}
}

// Build enforces what each standard requires of a package-level
// description. Identity-only terms build a complete eark package; under
// basic they are refused, and a missing identity is refused under either
// profile, all before any side effect.
func TestBuildRequiredPerStandard(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	if _, err := b.Build(in); err != nil {
		t.Fatalf("eark Build refused identity-only terms: %v", err)
	}

	cases := []struct {
		name string
		def  build.Definition
		desc sip.Description
		want string
	}{
		{"basic without description and created", basicDef(t), meemooIdentityTerms(), "description is required"},
		{"basic without a title", basicDef(t), meemoo.Terms{{Key: "identifier", Value: "x"}}, "title is required"},
		{"eark without an identifier", earkDef(t), eark.Terms{{Key: "title", Value: "x"}}, "identifier is required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, c.def)
			in.Description = c.desc
			_, err := b.Build(in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Build error = %v, want %q", err, c.want)
			}
			requireEmpty(t, outDir)
		})
	}
}
