package input

import (
	"strings"
	"testing"
)

// The walk runs without a mapper or a document format: it reads names,
// kinds and places only.
func TestWalkReportsFolderRules(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        "not read by the walk",
		"stray.tiff":                             "x",
		"representations/master/a.tiff":          "a",
		"representations/master/dc.xml":          "content under a profile that takes rows only",
		"representations/access/b.jpg":           "b",
		"representations/access/description.csv": "not read by the walk",
	})
	r := &folderReader{root: root}
	source, inv := r.walk()

	if len(r.violations) != 1 || !strings.Contains(r.violations[0], "stray.tiff: content must live inside representations/") {
		t.Errorf("violations = %q, want only the content beside representations/", r.violations)
	}
	if inv.pkg.rows == "" || inv.reps["access"].rows == "" || inv.reps["master"] != (descriptionFiles{}) {
		t.Errorf("inventory = %+v, want the package and access rows files only", inv)
	}
	if source.Description != nil {
		t.Error("the walk must not decode the description")
	}
	if got := paths(source.Representations[1].Files); len(got) != 2 {
		t.Errorf("master content = %v, want a.tiff and dc.xml", got)
	}
}

func TestWalkIgnoresRepresentationsCSVInAFlatFolder(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":     minimalCSV,
		"representations.csv": "folder,label\nmaster,Master\n",
		"a.tiff":              "a",
	})
	r := &folderReader{root: root}
	_, inv := r.walk()

	if len(r.violations) != 1 || !strings.Contains(r.violations[0], "representations.csv requires a representations/ folder") {
		t.Errorf("violations = %q, want only the missing representations/ folder", r.violations)
	}
	if inv.representationsCSV != "" {
		t.Errorf("representationsCSV = %q, want it left out of the inventory", inv.representationsCSV)
	}
}

func TestWalkReportsBothDescriptionsOnce(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalDC,
		"dc.xml":          "<simpledc/>",
		"a.tiff":          "a",
	})
	r := &folderReader{root: root, documentName: "dc.xml"}
	_, inv := r.walk()

	if len(r.violations) != 1 || !strings.Contains(r.violations[0], "are both present") {
		t.Errorf("violations = %q, want only the two descriptions, not a missing one", r.violations)
	}
	if inv.pkg != (descriptionFiles{}) {
		t.Errorf("inventory.pkg = %+v, want neither file recorded", inv.pkg)
	}
}

func TestIsOSArtifact(t *testing.T) {
	for _, name := range []string{".DS_Store", "Thumbs.db", "thumbs.db", "desktop.ini", "._resource"} {
		if !isOSArtifact(name) {
			t.Errorf("%q must be ignored as an OS artifact", name)
		}
	}
	for _, name := range []string{"scan.tiff", ".gitignore", "_underscore.txt"} {
		if isOSArtifact(name) {
			t.Errorf("%q must not be treated as an OS artifact", name)
		}
	}
}
