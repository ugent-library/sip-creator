package input

import (
	"testing"
)

// twoRepTree returns the file map of a valid two-representation folder;
// tests add their representations.csv on top.
func twoRepTree() map[string]string {
	return map[string]string{
		"metadata.csv":                     minimalCSV,
		"representations/master/scan.tiff": "a",
		"representations/access/book.pdf":  "b",
	}
}

func TestRepresentationsCSV(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv"] = "directory,label,type\nmaster,Master scan,archival\naccess,,\n"
	root := writeTree(t, tree)

	pkg, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(pkg.Representations) != 2 {
		t.Fatalf("want 2 representations, got %d", len(pkg.Representations))
	}

	// Row order becomes packaging order: master listed first wins over the
	// lexical directory order that put access first.
	master, access := pkg.Representations[0], pkg.Representations[1]
	if master.Name != "master" || access.Name != "access" {
		t.Fatalf("order = %q, %q; want the CSV row order master, access", master.Name, access.Name)
	}
	if master.Label != "Master scan" || master.Type != "archival" {
		t.Errorf("master label/type = %q/%q, want the CSV values", master.Label, master.Type)
	}
	// Empty cells stay empty: the library applies the defaulting cascade,
	// not the decoder.
	if access.Label != "" || access.Type != "" {
		t.Errorf("access label/type = %q/%q, want empty (defaults resolve in the library)", access.Label, access.Type)
	}
}

func TestRepresentationsCSVColumnsByHeaderName(t *testing.T) {
	tree := twoRepTree()
	// Reordered columns, capitalized headers (spreadsheet tools capitalize;
	// matching is case-insensitive), and a directory-only file are all fine.
	tree["representations.csv"] = "Type,Directory\narchival,master\naccess-copy,access\n"
	root := writeTree(t, tree)

	pkg, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := pkg.Representations[0].Type; got != "archival" {
		t.Errorf("type = %q, want %q via the reordered header", got, "archival")
	}
	if got := pkg.Representations[0].Label; got != "" {
		t.Errorf("label = %q, want empty when the file has no Label column", got)
	}
}

func TestRepresentationsCSVViolations(t *testing.T) {
	cases := []struct {
		name string
		csv  string
		want string
	}{
		{"unknown header column", "directory,colour\nmaster,red\naccess,blue\n", "unknown column"},
		{"no directory column", "label,type\na,b\n", "no directory column"},
		{"duplicate header column", "directory,directory\nmaster,master\naccess,access\n", "twice"},
		{"no data rows", "directory,label,type\n", "no rows"},
		{"empty file", "", "the file is empty"},
		{"unmatched row", "directory\nmaster\naccess\npreservation\n", "no directory representations/preservation"},
		{"duplicate directory row", "directory\nmaster\naccess\nmaster\n", "already has a row"},
		{"empty directory cell", "directory,label\n,Master scan\naccess,\n", "directory cell is empty"},
		{"row width mismatch", "directory,label\nmaster\naccess,Access copy\n", "expected 2 columns"},
		{"xml-unsafe label", "directory,label\nmaster,\"Master \"\"scan\"\"\"\naccess,\n", "label"},
		{"xml-unsafe type", "directory,type\nmaster,a&b\naccess,\n", "type"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tree := twoRepTree()
			tree["representations.csv"] = c.csv
			_, err := Read(writeTree(t, tree))
			assertViolation(t, err, c.want)
		})
	}
}

func TestRepresentationsCSVUncoveredDirectory(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv"] = "directory\nmaster\n"
	_, err := Read(writeTree(t, tree))
	assertViolation(t, err, "representations/access is not listed")
}

func TestRepresentationsCSVRequiresRepresentationsFolder(t *testing.T) {
	root := writeTree(t, map[string]string{
		"metadata.csv":        minimalCSV,
		"scan.tiff":           "a",
		"representations.csv": "directory\nx\n",
	})
	_, err := Read(root)
	assertViolation(t, err, "requires a representations/ folder")
}

func TestRepresentationsCSVMustBeAFile(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv/"] = ""
	_, err := Read(writeTree(t, tree))
	assertViolation(t, err, "representations.csv is a folder")
}

// A Directory-only CSV listing every directory in lexical order is a no-op:
// the read result equals the no-CSV read.
func TestRepresentationsCSVDirectoryOnlyIsANoop(t *testing.T) {
	plain, err := Read(writeTree(t, twoRepTree()))
	if err != nil {
		t.Fatalf("Read without CSV: %v", err)
	}

	tree := twoRepTree()
	tree["representations.csv"] = "directory\naccess\nmaster\n"
	withCSV, err := Read(writeTree(t, tree))
	if err != nil {
		t.Fatalf("Read with CSV: %v", err)
	}

	if len(plain.Representations) != len(withCSV.Representations) {
		t.Fatalf("representation counts differ: %d vs %d", len(plain.Representations), len(withCSV.Representations))
	}
	for i := range plain.Representations {
		p, w := plain.Representations[i], withCSV.Representations[i]
		if p.Name != w.Name || w.Label != "" || w.Type != "" {
			t.Errorf("representation %d differs: %q/%q/%q vs %q/%q/%q",
				i, p.Name, p.Label, p.Type, w.Name, w.Label, w.Type)
		}
	}
}

// The BOM spreadsheet tools prepend must not hide the header.
func TestRepresentationsCSVWithBOM(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv"] = "\ufeffdirectory,label\nmaster,Master scan\naccess,\n"
	pkg, err := Read(writeTree(t, tree))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := pkg.Representations[0].Label; got != "Master scan" {
		t.Errorf("label = %q, want %q", got, "Master scan")
	}
}
