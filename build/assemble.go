package build

import (
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/sip"
)

// assemble builds the package graph from a validated source package
// without writing anything to disk. It creates every File node with its
// Path. The writer fills in each node's fixity later, as it writes the
// file. It returns an error if a schema, a received PREMIS document or a
// characterization record cannot be used.
func (b *Builder) assemble(source *SourcePackage) (*sip.Package, error) {
	pkg := sip.NewPackage(b.destination, source.PackageIdentifier)
	b.logger.Info("created a new package", slog.String("id", pkg.Identifier))

	// The package's declaration is a copy of the profile's, with the record
	// status and content category the source package supplies. A copy, so
	// the graph never points into the builder's profile.
	decl := b.profile.Declaration
	if source.RecordStatus != "" {
		decl.RecordStatus = source.RecordStatus
	}
	if source.ContentCategory != "" {
		decl.Type = source.ContentCategory
	}
	// The software agent comes first, ahead of the profile's agents and the
	// submitter. A new slice, so the profile's backing array is never
	// written into.
	decl.Agents = append([]sip.Agent{softwareAgent()}, decl.Agents...)
	pkg.Declaration = &decl

	e := sip.NewEntity()
	b.logger.Info("created an intellectual entity", slog.String("id", e.Identifier))

	b.assembleDescriptive(e, source)
	// The package ships the XSDs the METS documents point at, which this
	// module bundles, and those the metadata model supplies. The
	// descriptive list ships for a supplied document too, whatever that
	// document's own schema-location hint names.
	schemaFiles, err := schemaFileNodes(slices.Concat(BundledSchemas(mets.Schemas...), b.profile.Model.Schemas()))
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", b.profile.Name, err)
	}
	pkg.SchemaFiles = schemaFiles

	pkg.DocumentationFiles = b.assembleDocumentationNodes(source.Documentation, source.Characterization)
	received, err := b.assembleReceivedPremis("package", source.Premis)
	if err != nil {
		return nil, err
	}
	pkg.ReceivedPremisFiles = received
	if err := b.assembleRepresentations(e, *pkg.Declaration, source); err != nil {
		return nil, err
	}

	if b.profile.EmitPackagePremis {
		pf := sip.NewFile()
		pf.Name = "premis.xml"
		pf.Path = "metadata/preservation/premis.xml"
		pf.Mime = "text/xml" // generated XML
		pkg.PremisFile = pf
		b.logger.Info("created a package PREMIS file", slog.String("id", pf.Identifier))
	}

	mf := sip.NewFile()
	mf.Name = "METS.xml"
	mf.Path = "METS.xml"
	// Every node has a media type, so the package METS gets one too,
	// although nothing inside the package references it.
	mf.Mime = "text/xml"
	pkg.MetsFile = mf
	b.logger.Info("created a package METS file", slog.String("id", mf.Identifier))

	pkg.Root = e
	return pkg, nil
}

func (b *Builder) assembleDescriptive(e *sip.Entity, source *SourcePackage) {
	d := source.Description
	if s, ok := b.profile.Model.(IdentifierSwapper); ok {
		e.AdditionalIdentifiers["MEEMOO-LOCAL-ID"] = s.Swap(d, e.Identifier)
	}
	e.Description = d

	df := b.descriptionFile()
	e.DescriptionFile = df
	b.logger.Info("created a descriptive file", slog.String("id", df.Identifier))
}

// descriptionFile declares the node for a descriptive document, the
// package's or a representation's: the same name and path under the
// metadata/descriptive/ of its level, and the type and version of the
// profile's metadata model, which the METS dmdSec declares.
func (b *Builder) descriptionFile() *sip.File {
	df := sip.NewFile()
	df.Name = b.profile.DocumentName
	df.Path = "metadata/descriptive/" + df.Name // relative to the METS of its level, per File.Path
	df.Mime = "text/xml"                        // rendered, or supplied and read as XML
	df.MDType, df.OtherMDType = mets.MDType(b.profile.Model.ModelType())
	df.MDTypeVersion = b.profile.Model.ModelTypeVersion()
	return df
}

// schemaFileNodes declares one graph node per XSD the package ships,
// sorted by name so the METS lists them in the same order on every build.
// Each name gets one node, because the METS list and the metadata model's
// list overlap where two documents point at the same schema. It returns an
// error if a name is not a plain file name, if a schema has no contents,
// or if two different schemas share a name, where one would replace the
// other.
func schemaFileNodes(list []Schema) ([]*sip.File, error) {
	contents := make(map[string][]byte, len(list))
	for _, s := range list {
		if err := validateSchemaName(s.Name); err != nil {
			return nil, err
		}
		if len(s.Content) == 0 {
			return nil, fmt.Errorf("the schema %q has no contents", s.Name)
		}
		if seen, ok := contents[s.Name]; ok && !bytes.Equal(seen, s.Content) {
			return nil, fmt.Errorf("two different schemas are named %q", s.Name)
		}
		contents[s.Name] = s.Content
	}

	files := make([]*sip.File, 0, len(contents))
	for _, name := range slices.Sorted(maps.Keys(contents)) {
		f := sip.NewFile()
		f.Name = name
		f.Path = "schemas/" + name
		f.Mime = "application/xml"
		f.Content = contents[name]
		files = append(files, f)
	}
	return files, nil
}

// validateSchemaName checks that name is a plain file name under schemas/
// and is text XML can carry in the documents' schema-location hints. It
// returns an error if name is empty, is . or .., holds a slash or a
// backslash, or holds a character XML cannot carry.
func validateSchemaName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("the schema name %q is not a plain file name", name)
	}
	if err := ValidateXMLText(name); err != nil {
		return fmt.Errorf("the schema name: %w", err)
	}
	return nil
}

// assembleDocumentationNodes declares a graph node for each documentation
// file of the package or of a representation, with its Path under
// documentation/ relative to its container. Unlike essence, documentation
// needs no characterization entry (ADR-0009). When a file has one, the
// entry gives it its media type and its checksum (ADR-0032).
func (b *Builder) assembleDocumentationNodes(sources []SourceFile, chars characterization.Report) []*sip.File {
	var files []*sip.File
	for _, src := range sources {
		f := sip.NewFile()
		f.Name = path.Base(src.Path)
		f.Source = src.Source
		f.Path = "documentation/" + src.Path
		f.Mime = "application/octet-stream" // unknown until a report entry says otherwise

		if rec, ok := chars[src.Key]; ok {
			f.Checksum = rec.MD5
			if rec.Mime != "" {
				f.Mime = rec.Mime
			}
		}

		files = append(files, f)
	}
	return files
}

// assembleRepresentations declares a graph node for each source
// representation, with nodes for its files. decl is the package's
// declaration, which each representation's declaration starts from. The
// producer's name is used verbatim as the directory under
// representations/ and as the representation METS OBJID, because no spec
// prescribes a naming scheme. CSIP requires only that the names are
// unique. Meemoo 2.x requires the directory name to equal the OBJID, which
// holds because both come from Name. It returns an error if a
// characterization record or a received PREMIS document cannot be used.
func (b *Builder) assembleRepresentations(e *sip.Entity, decl sip.MetsDeclaration, source *SourcePackage) error {
	for _, sr := range source.Representations {
		r := sip.NewRepresentation(sr.Name)
		r.Label = sr.label()
		r.Declaration = b.profile.representationDeclaration(decl, b.profile.representationType(sr))
		b.logger.Info("created a representation", slog.String("id", r.Identifier), slog.String("name", sr.Name))

		if sr.Description != nil {
			// As at the package level, the written document carries the
			// representation's identifier in place of the producer's. A
			// representation's description need not carry an identifier.
			// The replaced identifier is dropped, because MEEMOO-LOCAL-ID
			// identifies the entity.
			if s, ok := b.profile.Model.(IdentifierSwapper); ok {
				s.Swap(sr.Description, r.Identifier)
			}
			r.Description = sr.Description

			df := b.descriptionFile()
			r.DescriptionFile = df
			b.logger.Info("created a representation descriptive file", slog.String("id", df.Identifier))
		}

		for _, src := range sr.Files {
			f := sip.NewFile()
			f.Name = path.Base(src.Path)
			f.Source = src.Source
			f.Path = "data/" + src.Path // relative to the representation, per File.Path
			// A characterization report is optional (ADR-0009). When
			// present, it gives each file its format, media type and
			// checksum. The writer then copies the file without computing a
			// checksum (ADR-0032).
			f.Mime = "application/octet-stream" // unknown until the report says otherwise
			if source.Characterization != nil {
				rec, err := b.essenceRecord(source.Characterization, src)
				if err != nil {
					return err
				}
				f.Format = rec.Format
				f.Checksum = rec.MD5
				if rec.Mime != "" {
					f.Mime = rec.Mime
				}
			}
			f.Representation = r
			r.Files = append(r.Files, f)
			b.logger.Info("placed an essence file", slog.String("id", f.Identifier))
		}

		received, err := b.assembleReceivedPremis(fmt.Sprintf("representation %q", sr.Name), sr.Premis)
		if err != nil {
			return err
		}
		r.ReceivedPremisFiles = received

		r.DocumentationFiles = b.assembleDocumentationNodes(sr.Documentation, source.Characterization)

		if b.profile.EmitRepresentationPremis {
			pf := sip.NewFile()
			pf.Name = "premis.xml"
			pf.Path = "metadata/preservation/premis.xml" // relative to the representation, per File.Path
			pf.Mime = "text/xml"                         // generated XML
			r.PremisFile = pf
			b.logger.Info("created a representation PREMIS file", slog.String("id", pf.Identifier))
		}

		mf := sip.NewFile()
		mf.Name = "METS.xml"
		mf.Path = "representations/" + r.Name + "/METS.xml" // relative to the package METS, which references it
		mf.Mime = "text/xml"                                // generated XML
		r.MetsFile = mf
		b.logger.Info("created a representation METS file", slog.String("id", mf.Identifier))

		r.Entity = e
		e.Representations = append(e.Representations, r)
	}
	return nil
}

// assembleReceivedPremis declares a graph node for each received
// preservation document of a container. The package carries each document
// as received, never merged with the generated one. It checks that each
// document parses as XML and has a premis:premis root in the PREMIS 3
// namespace, because a file under metadata/preservation/ that is not
// PREMIS would be a false preservation claim. It returns an error if a
// document cannot be opened or fails the check.
func (b *Builder) assembleReceivedPremis(container string, sources []SourceFile) ([]*sip.File, error) {
	var files []*sip.File
	for _, src := range sources {
		f, err := os.Open(src.Source)
		if err != nil {
			return nil, fmt.Errorf("%s premis: %w", container, err)
		}
		err = premis.ValidateReceived(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s premis %s: %w", container, src.Path, err)
		}

		node := sip.NewFile()
		node.Name = path.Base(src.Path)
		node.Source = src.Source
		node.Path = "metadata/preservation/" + src.Path // relative to the METS of its level, per File.Path
		node.Mime = "text/xml"                          // verified XML above
		files = append(files, node)
		b.logger.Info("placed a received preservation file", slog.String("id", node.Identifier))
	}
	return files, nil
}

// essenceRecord looks up the characterization record of an essence file
// and checks that it exists, records no error and carries a checksum
// (ADR-0009). It returns the record, or an error if a check fails. The
// record's checksum is taken as given (ADR-0032).
func (b *Builder) essenceRecord(chars characterization.Report, src SourceFile) (characterization.Record, error) {
	rec, ok := chars[src.Key]
	if !ok {
		return characterization.Record{}, fmt.Errorf(
			"characterization report has no entry for %q (report keys look like %s); generate the report from the input root: sf -hash md5 -json .",
			src.Key, sampleKey(chars))
	}
	if rec.Errors != "" {
		return characterization.Record{}, fmt.Errorf("characterization report records an error for %q: %s", src.Key, rec.Errors)
	}
	if rec.MD5 == "" {
		return characterization.Record{}, fmt.Errorf("characterization report carries no checksum for %q; generate it with sf -hash md5 -json", src.Key)
	}
	return rec, nil
}

// sampleKey returns the report's first key in sorted order, as an example
// for an error message, so an operator can see when a report was
// generated from the wrong folder.
func sampleKey(chars characterization.Report) string {
	keys := slices.Sorted(maps.Keys(chars))
	if len(keys) == 0 {
		return "(the report is empty)"
	}
	return fmt.Sprintf("%q", keys[0])
}
