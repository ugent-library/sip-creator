package input

import (
	"errors"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/cli/input/mapping"
)

// twoRepTree returns the file map of a valid folder with two
// representations. Tests add their representations.csv to it.
func twoRepTree() map[string]string {
	return map[string]string{
		"description.csv":                  minimalCSV,
		"representations/master/scan.tiff": "a",
		"representations/access/book.pdf":  "b",
	}
}

func TestRepresentationsCSV(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv"] = "folder,label\nmaster,\"Master scan, R&D \"\"A\"\"\"\naccess,\n"
	root := writeTree(t, tree)

	pkg, err := Read(root, mapping.Meemoo{}, meemooDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(pkg.Representations) != 2 {
		t.Fatalf("want 2 representations, got %d", len(pkg.Representations))
	}

	// Row order becomes packaging order: master listed first wins over the
	// lexical folder order that put access first.
	master, access := pkg.Representations[0], pkg.Representations[1]
	if master.Name != "master" || access.Name != "access" {
		t.Fatalf("order = %q, %q; want the CSV row order master, access", master.Name, access.Name)
	}
	// The label is taken as written: the METS encoder escapes it.
	if master.Label != `Master scan, R&D "A"` {
		t.Errorf("master label = %q, want the CSV value", master.Label)
	}
	// An empty cell stays empty: build.SourceRepresentation fills in the
	// default, not Read.
	if access.Label != "" {
		t.Errorf("access label = %q, want empty (the default resolves in the library)", access.Label)
	}
}

// Columns are found by name: reordered, capitalized (spreadsheet tools
// capitalize) and left out are all fine. The BOM spreadsheet tools write
// must not hide the header, and spaces around a folder name are dropped.
func TestParseRepresentationRows(t *testing.T) {
	data := "\ufeffLabel,Folder\nMaster scan, master \nAccess copy,access\n"
	rows, errs, err := parseRepresentationRows([]byte(data))
	if err != nil || len(errs) > 0 {
		t.Fatalf("parseRepresentationRows: err %v, findings %v", err, errs)
	}
	want := []repRow{
		{line: 2, folder: "master", label: "Master scan"},
		{line: 3, folder: "access", label: "Access copy"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], w)
		}
	}
}

// A file that cannot be used is refused as a whole, with every header
// problem named.
func TestParseRepresentationRowsUnusableFile(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string // substrings of the joined error
	}{
		{"empty file", "", []string{"the file is empty"}},
		{"not utf-8", "folder\n\xff\n", []string{"not valid UTF-8"}},
		{"unknown column", "folder,colour\nmaster,red\n", []string{`unknown column "colour"`}},
		{"no folder column", "label\na\n", []string{"no folder column"}},
		// A file with a type column is refused, not read without it, so
		// a producer learns the column has no effect.
		{"type column", "folder,type\nmaster,archival\n", []string{`unknown column "type"`}},
		{"duplicate column", "folder,folder\nmaster,master\n", []string{`column "folder" twice`}},
		{"every header problem", "colour,label,label\na,b,c\n", []string{`unknown column "colour"`, `column "label" twice`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseRepresentationRows([]byte(tt.data))
			if err == nil {
				t.Fatal("want the file refused")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
		})
	}
}

func TestParseRepresentationRowsFindings(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantLine int    // line of the *rowError, or 0 for a finding about the file
		want     string // substring of the finding
		wantRows int    // rows still returned
	}{
		{"row width", "folder,label\nmaster\n", 2, "expected 2 columns", 0},
		{"empty folder cell", "folder,label\n ,Master scan\n", 2, "folder cell is empty", 0},
		{"broken quote", "folder,label\nmaster,\"open\n", 0, "extraneous or missing", 0},
		// A row with a value XML cannot carry is kept, so matching it to a
		// folder still reports on it.
		{"label XML cannot carry", "folder,label\nmaster,Master\x01\n", 2, "label:", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, errs, err := parseRepresentationRows([]byte(tt.data))
			if err != nil {
				t.Fatalf("parseRepresentationRows: %v", err)
			}
			assertFinding(t, errs, tt.want)
			if len(rows) != tt.wantRows {
				t.Errorf("got %d rows, want %d", len(rows), tt.wantRows)
			}
			if tt.wantLine == 0 {
				return
			}
			re, ok := errors.AsType[*rowError](errs[0])
			if !ok || re.line != tt.wantLine {
				t.Errorf("finding %v, want a *rowError at line %d", errs[0], tt.wantLine)
			}
		})
	}
}

func TestRepresentationsCSVViolations(t *testing.T) {
	cases := []struct {
		name string
		csv  string
		want string
	}{
		{"unknown header column", "folder,colour,folder\nmaster,red,x\naccess,blue,y\n", `representations.csv: unknown column "colour"`},
		{"duplicate header column", "folder,colour,folder\nmaster,red,x\naccess,blue,y\n", `representations.csv: the header names column "folder" twice`},
		{"row finding at its line", "folder,label\n,Master scan\naccess,\n", "representations.csv line 2: the folder cell is empty"},
		{"no data rows", "folder,label\n", "no rows"},
		{"unmatched row", "folder\nmaster\naccess\npreservation\n", "no folder representations/preservation"},
		{"duplicate folder row", "folder\nmaster\naccess\nmaster\n", "already has a row"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tree := twoRepTree()
			tree["representations.csv"] = c.csv
			_, err := Read(writeTree(t, tree), mapping.Meemoo{}, meemooDocumentSpec)
			assertViolation(t, err, c.want)
		})
	}
}

func TestRepresentationsCSVUncoveredFolder(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv"] = "folder\nmaster\n"
	_, err := Read(writeTree(t, tree), mapping.Meemoo{}, meemooDocumentSpec)
	assertViolation(t, err, "representations/access is not listed")
}

func TestRepresentationsCSVRequiresRepresentationsFolder(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":     minimalCSV,
		"scan.tiff":           "a",
		"representations.csv": "folder\nx\n",
	})
	_, err := Read(root, mapping.Meemoo{}, meemooDocumentSpec)
	assertViolation(t, err, "requires a representations/ folder")
}

func TestRepresentationsCSVMustBeAFile(t *testing.T) {
	tree := twoRepTree()
	tree["representations.csv/"] = ""
	_, err := Read(writeTree(t, tree), mapping.Meemoo{}, meemooDocumentSpec)
	assertViolation(t, err, "representations.csv is a folder")
}

// A CSV with only the folder column, listing every folder in lexical
// order, changes nothing: Read returns the same representations as
// without the file.
func TestRepresentationsCSVFolderOnlyIsANoop(t *testing.T) {
	plain, err := Read(writeTree(t, twoRepTree()), mapping.Meemoo{}, meemooDocumentSpec)
	if err != nil {
		t.Fatalf("Read without CSV: %v", err)
	}

	tree := twoRepTree()
	tree["representations.csv"] = "folder\naccess\nmaster\n"
	withCSV, err := Read(writeTree(t, tree), mapping.Meemoo{}, meemooDocumentSpec)
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
