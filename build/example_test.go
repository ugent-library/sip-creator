package build_test

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
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

	def, ok := profiles.Get("eark")
	if !ok {
		log.Fatal("no eark profile")
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
		Description: eark.Terms{
			{Key: "identifier", Value: "example-0001"},
			{Key: "title", Value: "Example photograph"},
			{Key: "description", Value: "An example package with one image."},
			{Key: "date", Value: "2026-01-15"},
		},
		Representations: []build.SourceRepresentation{{
			Name: "master",
			Files: []build.SourceFile{{
				Source: "../examples/eark/representations/master/image-001.jpg",
				Path:   "image-001.jpg",
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

	def, ok := profiles.Get("eark-mods")
	if !ok {
		log.Fatal("no eark-mods profile")
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
		Description: earkmods.Record{
			Identifier: "example-0001",
			Titles: []earkmods.Title{
				{Value: "Example book", Lang: "en"},
			},
			Items: []earkmods.Item{
				{CallNumber: "EX.0001", Barcode: "0000000001"},
				{CallNumber: "EX.0002", Enumeration: "vol. 2"},
			},
		},
		Representations: []build.SourceRepresentation{{
			Name: "master",
			Files: []build.SourceFile{{
				Source: "../examples/eark-mods/representations/master/image-001.jpg",
				Path:   "image-001.jpg",
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

	def, ok := profiles.Get("eark-mods")
	if !ok {
		log.Fatal("no eark-mods profile")
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
		Description: build.EncodedDescription{Source: "../examples/eark-mods/mods.xml"},
		Representations: []build.SourceRepresentation{{
			Name: "master",
			Files: []build.SourceFile{{
				Source: "../examples/eark-mods/representations/master/image-001.jpg",
				Path:   "image-001.jpg",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Root.DescriptionFile.Path)
	// Output: metadata/descriptive/mods.xml
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

	def, ok := profiles.Get("eark")
	if !ok {
		log.Fatal("no eark profile")
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
		Description: eark.Terms{
			{Key: "identifier", Value: "example-0001"},
			{Key: "title", Value: "Example photograph"},
		},
		Representations: []build.SourceRepresentation{{
			Name: "master",
			Files: []build.SourceFile{{
				Source: "../examples/eark/representations/master/image-001.jpg",
				Path:   "image-001.jpg",
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Identifier)
	// Output: uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e
}
