package xmldoc

import (
	"strings"
	"testing"
)

func TestRoot(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		space string // expected root namespace when accepted
		local string // expected root local name when accepted
		want  string // "" means accepted; else substring of the error
	}{
		{"prefixed root", `<?xml version="1.0"?><mods:mods xmlns:mods="http://www.loc.gov/mods/v3" version="3.7"><mods:titleInfo/></mods:mods>`, "http://www.loc.gov/mods/v3", "mods", ""},
		{"default namespace", `<premis xmlns="http://www.loc.gov/premis/v3" version="3.0"/>`, "http://www.loc.gov/premis/v3", "premis", ""},
		{"no namespace", `<simpledc><title>x</title></simpledc>`, "", "simpledc", ""},
		{"comment before the root", `<!-- a note --><simpledc/>`, "", "simpledc", ""},
		{"not xml", `not xml at all`, "", "", "not an XML document"},
		{"empty", ``, "", "", "not an XML document"},
		{"truncated", `<simpledc><title>`, "", "", "not well-formed"},
		{"broken after the root", `<simpledc></simpledc><`, "", "", "not well-formed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := Root(strings.NewReader(tt.doc))
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("want error mentioning %q, got %v", tt.want, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want accepted, got %v", err)
			}
			if root.Name.Space != tt.space || root.Name.Local != tt.local {
				t.Errorf("root = {%s}%s, want {%s}%s", root.Name.Space, root.Name.Local, tt.space, tt.local)
			}
		})
	}
}

func TestAttr(t *testing.T) {
	root, err := Root(strings.NewReader(`<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" version="3.7"/>`))
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := Attr(root, "version"); !ok || v != "3.7" {
		t.Errorf("version = %q, %v; want 3.7, true", v, ok)
	}
	if _, ok := Attr(root, "missing"); ok {
		t.Error("an absent attribute was found")
	}
}
