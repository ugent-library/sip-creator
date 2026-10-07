package cli

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
)

// Every registered profile has a mapping, and each builds what its
// profile's metadata model accepts: Read hands terms to the one, the
// engine runs ValidateType on the result. A profile without a mapping
// could not read a folder.
func TestEveryProfileHasAMappingItsModelAccepts(t *testing.T) {
	for _, name := range profiles.Names() {
		def, _ := profiles.Get(name)
		mapper, ok := mappers[name]
		if !ok {
			t.Errorf("profile %q has no mapping", name)
			continue
		}
		description, errs := mapper.Map(nil)
		if len(errs) != 0 {
			t.Errorf("profile %q: no terms, yet errors %v", name, errs)
		}
		if err := def.Model.ValidateType(description); err != nil {
			t.Errorf("profile %q: ValidateType refuses what its mapping built: %v", name, err)
		}
	}
	for name := range mappers {
		if _, ok := profiles.Get(name); !ok {
			t.Errorf("mapping for %q, which is not a registered profile", name)
		}
	}
}

// Every registered profile has an example input folder under examples/,
// and it builds. The examples are what producers copy and what build.sh
// validates, so a rule change that breaks one must fail here first. The
// examples carry no siegfried.json, so the build needs no sf.
func TestExamplesBuild(t *testing.T) {
	for _, name := range profiles.Names() {
		t.Run(name, func(t *testing.T) {
			mapper, ok := mappers[name]
			if !ok {
				t.Fatalf("profile %q has no mapping", name)
			}
			def, _ := profiles.Get(name)
			source, err := input.Read(filepath.Join("..", "examples", filepath.FromSlash(name)), mapper, input.DocumentSpec{Name: def.DocumentName, Model: def.Model})
			if err != nil {
				t.Fatalf("reading the example: %v", err)
			}

			def, err = def.WithSubmitter("Example Organization", "OR-0000000")
			if err != nil {
				t.Fatalf("WithSubmitter() error = %v", err)
			}
			builder, err := build.New(&build.Config{
				Profile:     def,
				Destination: t.TempDir(),
				Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
			})
			if err != nil {
				t.Fatalf("build.New() error = %v", err)
			}
			if _, err := builder.Build(source); err != nil {
				t.Fatalf("Build() error = %v", err)
			}
		})
	}
}
