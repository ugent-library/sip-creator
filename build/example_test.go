package build_test

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

// The content files in these examples are the ones in the repository's
// examples/ folder.

// Build a plain E-ARK package with Simple Dublin Core metadata.
func ExampleBuilder_Build() {
	destination, err := os.MkdirTemp("", "sip")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(destination)

	def, ok := profiles.Get("ugent/basic")
	if !ok {
		log.Fatal("no ugent/basic profile")
	}
	// The second argument is the Meemoo OR-id, used by the basic profile only.
	def, err = def.WithSubmitter("Example Organization", "")
	if err != nil {
		log.Fatal(err)
	}

	builder, err := build.New(&build.Config{
		Profile:     def,
		Destination: destination,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := builder.Build(&build.SourcePackage{
		Description: simpledc.Terms{
			{Key: "identifier", Value: "example-0001"},
			{Key: "title", Value: "Example photograph"},
			{Key: "description", Value: "An example package with one image."},
			{Key: "date", Value: "2026-01-15"},
		},
		Representations: []build.SourceRepresentation{{
			Name: "archival",
			Files: []build.SourceFile{{
				Source: "../examples/ugent/basic/representations/archival/image-001.tif",
				Path:   "image-001.tif",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Root.DescriptionFile.Path)
	// Output: metadata/descriptive/dc.xml
}

// Build a package with MODS metadata: a bibliographic record with two
// physical copies.
func ExampleBuilder_Build_mods() {
	destination, err := os.MkdirTemp("", "sip")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(destination)

	def, ok := profiles.Get("ugent/bibliographic")
	if !ok {
		log.Fatal("no ugent/bibliographic profile")
	}
	def, err = def.WithSubmitter("Example Organization", "")
	if err != nil {
		log.Fatal(err)
	}
	builder, err := build.New(&build.Config{
		Profile:     def,
		Destination: destination,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := builder.Build(&build.SourcePackage{
		Description: mods.Record{
			Identifier: "example-0001",
			Titles: []mods.Title{
				{Value: "Example book", Lang: "en"},
			},
			Items: []mods.Item{
				{CallNumber: "EX.0001", Barcode: "0000000001"},
				{CallNumber: "EX.0002", Enumeration: "vol. 2"},
			},
		},
		Representations: []build.SourceRepresentation{{
			Name: "archival",
			Files: []build.SourceFile{{
				Source: "../examples/ugent/bibliographic/representations/archival/image-001.tif",
				Path:   "image-001.tif",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Root.DescriptionFile.Path)
	// Output: metadata/descriptive/mods.xml
}

// Supply a finished descriptive document instead of terms or a record. The
// build checks its root element and copies it into the package as it is.
func ExampleEncodedDescription() {
	destination, err := os.MkdirTemp("", "sip")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(destination)

	def, ok := profiles.Get("ugent/bibliographic")
	if !ok {
		log.Fatal("no ugent/bibliographic profile")
	}
	def, err = def.WithSubmitter("Example Organization", "")
	if err != nil {
		log.Fatal(err)
	}
	builder, err := build.New(&build.Config{
		Profile:     def,
		Destination: destination,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := builder.Build(&build.SourcePackage{
		Description: build.EncodedDescription{Source: "../examples/ugent/bibliographic/mods.xml"},
		Representations: []build.SourceRepresentation{{
			Name: "archival",
			Files: []build.SourceFile{{
				Source: "../examples/ugent/bibliographic/representations/archival/image-001.tif",
				Path:   "image-001.tif",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Root.DescriptionFile.Path)
	// Output: metadata/descriptive/mods.xml
}

// catalogueRecord is the description type of a profile written outside
// this module, for a record format of the institution's own.
type catalogueRecord struct {
	Title string
}

func (r catalogueRecord) Validate() error {
	return build.ValidateXMLText(r.Title)
}

func (r catalogueRecord) ValidateRequired() error {
	if r.Title == "" {
		return errors.New("title is required but missing")
	}
	return nil
}

// catalogueSchema is the record format's XSD. A real profile embeds its
// XSD files in its own package with go:embed.
const catalogueSchema = `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="record">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="title" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>
`

// catalogueModel is the profile's metadata model: it accepts a
// catalogueRecord, writes it as a document, and supplies the XSD the
// document points at.
type catalogueModel struct{}

func (catalogueModel) ValidateType(d sip.Description) error {
	if _, ok := d.(catalogueRecord); !ok {
		return fmt.Errorf("descriptive metadata is %T, not a catalogue record", d)
	}
	return nil
}

func (catalogueModel) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var title bytes.Buffer
	if err := xml.EscapeText(&title, []byte(d.(catalogueRecord).Title)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<record xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="%s/catalogue.xsd">
  <title>%s</title>
</record>
`, schemasDir, title.String())
	return err
}

func (catalogueModel) Schemas() []build.Schema {
	return []build.Schema{{Name: "catalogue.xsd", Content: []byte(catalogueSchema)}}
}

// ModelType names the institution's own format. The METS MDTYPE
// vocabulary does not list it, so the METS dmdSec records it as MDTYPE
// OTHER with the name in OTHERMDTYPE.
func (catalogueModel) ModelType() string        { return "catalogue-record" }
func (catalogueModel) ModelTypeVersion() string { return "" }

// Build a package with a profile of your own: a description type, a
// metadata model that writes it and supplies its own XSD, and a definition
// handed to build.New. Nothing in this module's profiles/ is involved, and
// the package ships the XSD next to the ones its METS documents need.
func Example_ownProfile() {
	destination, err := os.MkdirTemp("", "sip")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(destination)

	def := build.Definition{
		Name:         "catalogue",
		Model:        catalogueModel{},
		DocumentName: "record.xml",
		Declaration: sip.MetsDeclaration{
			ProfileURL:             "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
			Type:                   "Mixed",
			ContentInformationType: "MIXED",
		},
	}
	def, err = def.WithSubmitter("Example Organization", "")
	if err != nil {
		log.Fatal(err)
	}
	builder, err := build.New(&build.Config{
		Profile:     def,
		Destination: destination,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := builder.Build(&build.SourcePackage{
		Description: catalogueRecord{Title: "Example photograph"},
		Representations: []build.SourceRepresentation{{
			Name: "archival",
			Files: []build.SourceFile{{
				Source: "../examples/ugent/basic/representations/archival/image-001.tif",
				Path:   "image-001.tif",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	description := pkg.Root.DescriptionFile
	fmt.Println(description.Path, description.MDType, description.OtherMDType)
	for _, sf := range pkg.SchemaFiles {
		fmt.Println(sf.Path)
	}
	// Output:
	// metadata/descriptive/record.xml OTHER catalogue-record
	// schemas/DILCISExtensionMETS.xsd
	// schemas/DILCISExtensionSIPMETS.xsd
	// schemas/catalogue.xsd
	// schemas/mets1_12.xsd
	// schemas/xlink.xsd
}

// Build a package that replaces an earlier one. The earlier package's
// identifier becomes this package's identifier (mets/@OBJID); RecordStatus
// is metsHdr/@RECORDSTATUS and ContentCategory is mets/@TYPE. Left empty,
// all three take the profile's values, and a package without a status is
// read as new.
func ExampleBuilder_Build_update() {
	destination, err := os.MkdirTemp("", "sip")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(destination)

	def, ok := profiles.Get("ugent/basic")
	if !ok {
		log.Fatal("no ugent/basic profile")
	}
	def, err = def.WithSubmitter("Example Organization", "")
	if err != nil {
		log.Fatal(err)
	}
	builder, err := build.New(&build.Config{
		Profile:     def,
		Destination: destination,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := builder.Build(&build.SourcePackage{
		PackageIdentifier: "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e",
		RecordStatus:      sip.RecordStatusReplacement,
		ContentCategory:   "Photographs – Digital",
		Description: simpledc.Terms{
			{Key: "identifier", Value: "example-0001"},
			{Key: "title", Value: "Example photograph"},
		},
		Representations: []build.SourceRepresentation{{
			Name: "archival",
			Files: []build.SourceFile{{
				Source: "../examples/ugent/basic/representations/archival/image-001.tif",
				Path:   "image-001.tif",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Identifier)
	// Output: uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e
}
