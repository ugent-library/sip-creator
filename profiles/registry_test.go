package profiles

import (
	"bytes"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// shipped is the sorted, deduplicated set of XSDs a profile's packages ship:
// what the METS documents point at plus what its descriptive document
// points at, the same sum the assembler makes.
func shipped(t *testing.T, name string) []string {
	t.Helper()
	def, ok := Get(name)
	if !ok {
		t.Fatalf("no %q definition registered", name)
	}
	return slices.Compact(slices.Sorted(slices.Values(slices.Concat(mets.Schemas, schemaNames(def.Model.Schemas())))))
}

// schemaNames returns the file names of the schemas in list.
func schemaNames(list []build.Schema) []string {
	names := make([]string, 0, len(list))
	for _, s := range list {
		names = append(names, s.Name)
	}
	return names
}

// withMETS returns the sorted set a profile ships when its descriptive
// document points at names: the METS set plus those.
func withMETS(names ...string) []string {
	return slices.Sorted(slices.Values(append(slices.Clone(mets.Schemas), names...)))
}

// Every XSD a profile ships is bundled, so a typo in a metadata model's list fails
// here rather than at the first build, and each profile ships exactly what
// its documents point at: meemoo/basic the METS set plus Meemoo's
// descriptive schema and what it imports (the Dublin Core family, EDTF,
// schema.org, xml.xsd), eark/dc the METS set plus simpledc.xsd and
// xml.xsd, eark/mods the METS set plus mods-3-7.xsd. The bundle is the
// union of what the profiles ship, so no profile ships all of it.
func TestRegistrySchemas(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Names() {
		for _, xsd := range shipped(t, name) {
			if _, ok := bundle[xsd]; !ok {
				t.Errorf("profile %q ships %q, which is not bundled", name, xsd)
			}
		}
	}

	basic := withMETS("descriptive_basic.xsd", "dc.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd")
	if got := shipped(t, "meemoo/basic"); !slices.Equal(got, basic) {
		t.Errorf("basic ships %v, want the METS set plus Meemoo's descriptive schemas %v", got, basic)
	}
	eark := withMETS("simpledc.xsd", "xml.xsd")
	if got := shipped(t, "eark/dc"); !slices.Equal(got, eark) {
		t.Errorf("eark ships %v, want the METS set plus the simpledc schema and xml.xsd %v", got, eark)
	}
	earkmods := withMETS("mods-3-7.xsd")
	if got := shipped(t, "eark/mods"); !slices.Equal(got, earkmods) {
		t.Errorf("eark/mods ships %v, want the METS set plus mods-3-7.xsd %v", got, earkmods)
	}
}

// sampleDescriptions holds one description per profile that its metadata
// model renders; Encode does not run Validate, and an empty description has no
// key for the template to refuse, so it is enough to render the
// document's root.
var sampleDescriptions = map[string]sip.Description{
	"meemoo/basic": meemoo.Terms{},
	"eark/dc":      simpledc.Terms{},
	"eark/mods":    mods.Record{},
}

// The schema-location hint of each profile's descriptive document points
// into the package's schemas/ directory, at files the profile ships: the
// template names the file and Schemas() lists it, and if the two drift the
// document points at a file the package does not carry. TestRegistrySchemas
// pins the other half, that every listed file is bundled.
func TestRegistryDescriptiveDocumentsPointAtShippedSchemas(t *testing.T) {
	const schemasDir = "../../schemas"
	for _, name := range Names() {
		description, ok := sampleDescriptions[name]
		if !ok {
			t.Errorf("profile %q has no sample description for this test", name)
			continue
		}
		def, _ := Get(name)
		var buf bytes.Buffer
		if err := def.Model.Encode(&buf, description, schemasDir); err != nil {
			t.Errorf("profile %q: Encode: %v", name, err)
			continue
		}
		locations := schemaLocations(t, buf.Bytes())
		if len(locations) == 0 {
			t.Errorf("profile %q: the document hints at no schema", name)
		}
		for _, loc := range locations {
			dir, file := path.Split(loc)
			listed := schemaNames(def.Model.Schemas())
			if path.Clean(dir) != schemasDir || !slices.Contains(listed, file) {
				t.Errorf("profile %q: the document points at %q, want a file under %s/ that Schemas() lists (%v)", name, loc, schemasDir, listed)
			}
		}
	}
}

const xsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"

// schemaLocations returns the locations the document's root hints at: the
// second of each namespace–location pair in xsi:schemaLocation, and every
// entry of xsi:noNamespaceSchemaLocation.
func schemaLocations(t *testing.T, doc []byte) []string {
	t.Helper()
	root, err := xmldoc.Root(bytes.NewReader(doc))
	if err != nil {
		t.Fatalf("rendered document: %v", err)
	}
	var locations []string
	for _, a := range root.Attr {
		if a.Name.Space != xsiNamespace {
			continue
		}
		fields := strings.Fields(a.Value)
		switch a.Name.Local {
		case "schemaLocation":
			for i := 1; i < len(fields); i += 2 {
				locations = append(locations, fields[i])
			}
		case "noNamespaceSchemaLocation":
			locations = append(locations, fields...)
		}
	}
	return locations
}

// Every registry entry names a metadata model; the engine refuses a
// definition without one before any write.
func TestRegistryEntriesNameAModel(t *testing.T) {
	for _, name := range Names() {
		def, _ := Get(name)
		if def.Model == nil {
			t.Errorf("profile %q has no metadata model", name)
		}
		if def.Name != name {
			t.Errorf("profile %q is registered under Name %q", name, def.Name)
		}
	}
}
