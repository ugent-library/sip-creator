package build_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

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
	// (the METS list plus the metadata model's), in sorted
	// (deterministic) order.
	names := make([]string, 0, len(pkg.SchemaFiles))
	for _, sf := range pkg.SchemaFiles {
		names = append(names, sf.Name)
		if sf.Path != "schemas/"+sf.Name {
			t.Errorf("schema Path = %q, want %q", sf.Path, "schemas/"+sf.Name)
		}
	}
	referenced := slices.Concat(mets.Schemas, basicDef(t).Model.Schemas())
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

	// The message shows a key the report does hold, so a report generated
	// from the wrong folder explains itself.
	_, err := b.Assemble(in)
	if want := `characterization report has no entry for "cat.jpg" (report keys look like "somewhere/else.jpg")`; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
	}
	requireEmpty(t, outDir)

	in.Characterization = characterization.Report{}
	_, err = b.Assemble(in)
	if want := "(report keys look like (the report is empty))"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error with an empty report = %v, want %q", err, want)
	}
}

// With a report, every essence file is read for its checksum during
// assembly, so a file that is gone ends the build before anything is
// written, with the file system's error.
func TestAssembleReportSourceMissing(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = report(t, src)
	if err := os.Remove(src.Source); err != nil {
		t.Fatal(err)
	}

	_, err := b.Assemble(in)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("assemble error = %v, want the missing file reported", err)
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

	_, err := b.Assemble(in)
	if want := "changed since the characterization report was generated"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
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

	// An empty checksum would also fail the comparison with the file; the
	// error must come from the rule on a missing checksum.
	_, err := b.Assemble(in)
	if want := `characterization report carries no checksum for "cat.jpg"`; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
	}
	requireEmpty(t, outDir)
}

// A per-file error recorded by the characterizer aborts: the tool is telling
// us it never characterized these bytes.
func TestAssembleReportEntryError(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	src := in.Representations[0].Files[0]
	in.Characterization = characterization.Report{
		// The checksum matches, so only the recorded error can refuse it.
		src.Key: {MD5: fileMD5(t, src.Source), Errors: "permission denied"},
	}

	_, err := b.Assemble(in)
	if want := `characterization report records an error for "cat.jpg": permission denied`; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
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
	_, err = b.Assemble(in)
	if want := "manual.txt changed since the characterization report was generated"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
	}
}

// A representation may carry its own descriptive terms: they
// land on the representation with a rep-relative file node, an identifier
// term is swapped for the representation identifier, and identity is not
// required.
func TestAssembleRepresentationDescriptive(t *testing.T) {
	b, in, _ := newTestBuilder(t, basicDef(t))
	in.Representations[0].Description = meemoo.Terms{
		{Key: "dcterms:license", Value: "publiek domein"},
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
		{Key: "dcterms:identifier", Value: "rep-local-1"},
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

// identifierTerm returns the identifier any world's description states
// ("" when absent): what the Meemoo swap wrote, or what the eark profiles
// left alone. No profile package exports an accessor for it; the swap is
// the Meemoo package's own business, and the eark profiles never swap.
func identifierTerm(d sip.Description) string {
	var terms []sip.Term
	key := "identifier"
	switch v := d.(type) {
	case meemoo.Terms:
		terms, key = v, "dcterms:identifier"
	case eark.Terms:
		terms = v
	case earkmods.Record:
		return v.Identifier
	}
	for _, term := range terms {
		if term.Key == key {
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

// Both descriptive file nodes, the package's and a representation's, carry
// the dmdSec label of the profile's metadata model.
func TestAssembleLabelsDescriptionFilesWithTheModel(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = eark.Terms{{Key: "rights", Value: "CC BY 4.0"}}

	pkg, err := b.Assemble(in)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	for name, df := range map[string]*sip.File{
		"package":        pkg.Root.DescriptionFile,
		"representation": pkg.Root.Representations[0].DescriptionFile,
	} {
		if df.MDType != "DC" || df.MDTypeVersion != "SimpleDC20021212" {
			t.Errorf("%s descriptive file labeled %q/%q, want the Simple DC model's DC/SimpleDC20021212",
				name, df.MDType, df.MDTypeVersion)
		}
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
		t.Error("MEEMOO-LOCAL-ID lifted onto the entity; it is a Meemoo concept")
	}
	if got := identifierTerm(e.Representations[0].Description); got != "rep-local-1" {
		t.Errorf("rep descriptive identifier = %q, want the producer's %q", got, "rep-local-1")
	}
}

// The eark profile types each representation METS by its resolved type,
// in both the TYPE and the CONTENTINFORMATIONTYPE pair; the basic profile
// keeps the profile declaration unchanged; the package declaration never
// changes (ADR-0013).
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
// required, and a present entry's checksum must match.
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

	// A stale entry for a representation's documentation file still aborts.
	in.Characterization = report(t, in.Representations[0].Files[0], note)
	if err := os.WriteFile(note.Source, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = b.Assemble(in)
	if want := "scan-notes.txt changed since the characterization report was generated"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("assemble error = %v, want %q", err, want)
	}
}

// A received file that is not a PREMIS document aborts assembly, at both
// levels: packaging it under metadata/preservation/ would be a false
// preservation claim. So does one that cannot be read. The error names the
// level, and the file where assembly got as far as reading it.
func TestAssembleReceivedPremisRejectsNonPremis(t *testing.T) {
	tests := []struct {
		name  string
		place func(in *build.SourcePackage, f build.SourceFile)
		file  func(t *testing.T, dir string) build.SourceFile
		want  string
	}{
		{"not PREMIS, package level",
			func(in *build.SourcePackage, f build.SourceFile) { in.Premis = []build.SourceFile{f} },
			func(t *testing.T, dir string) build.SourceFile {
				return writeEssence(t, dir, "vendor.xml", "not xml at all")
			},
			"package premis vendor.xml: not an XML document"},
		{"not PREMIS, representation level",
			func(in *build.SourcePackage, f build.SourceFile) {
				in.Representations[0].Premis = []build.SourceFile{f}
			},
			func(t *testing.T, dir string) build.SourceFile { return writeEssence(t, dir, "capture.xml", "<mets/>") },
			`representation "master" premis capture.xml: root element is {}mets`},
		{"missing file",
			func(in *build.SourcePackage, f build.SourceFile) { in.Premis = []build.SourceFile{f} },
			func(t *testing.T, dir string) build.SourceFile {
				return build.SourceFile{Source: filepath.Join(dir, "gone.xml"), Path: "gone.xml"}
			},
			"package premis: open "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, basicDef(t))
			tt.place(in, tt.file(t, t.TempDir()))

			_, err := b.Assemble(in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("assemble error = %v, want %q", err, tt.want)
			}
			requireEmpty(t, outDir)
		})
	}
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

// unbundledSchemas wraps a real metadata model and claims an XSD the bundle does
// not hold.
type unbundledSchemas struct{ build.MetadataModel }

func (unbundledSchemas) Schemas() []string { return []string{"nope.xsd"} }

// A metadata model listing a schema the bundle does not hold is refused at
// assembly, before any write: the alternative is an empty XSD in the
// package.
func TestBuildRefusesUnbundledSchema(t *testing.T) {
	def := basicDef(t)
	def.Model = unbundledSchemas{def.Model}
	b, in, outDir := newTestBuilder(t, def)
	_, err := b.Build(in)
	if err == nil || !strings.Contains(err.Error(), `"nope.xsd"`) {
		t.Fatalf("Build error = %v, want the unbundled schema named", err)
	}
	requireEmpty(t, outDir)
}
