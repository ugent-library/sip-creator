package input

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ugent-library/sip-creator/encoders/metadata"
)

// plainKeys maps the spec's plain metadata.csv keys (§3) onto the Dublin
// Core elements they generate (§7).
var plainKeys = map[string]string{
	"identifier":   "dcterms:identifier",
	"title":        "dcterms:title",
	"description":  "dcterms:description",
	"created":      "dcterms:created",
	"creator":      "dcterms:creator",
	"contributor":  "dcterms:contributor",
	"publisher":    "dcterms:publisher",
	"subject":      "dcterms:subject",
	"spatial":      "dcterms:spatial",
	"extent":       "dcterms:extent",
	"language":     "dcterms:language",
	"type":         "dcterms:type",
	"ispartof":     "dcterms:isPartOf",
	"license":      "dcterms:license",
	"rights":       "dcterms:rights",
	"rightsholder": "dcterms:rightsHolder",
}

// decodeMetadataCSV decodes one metadata.csv into ordered descriptive
// terms, collecting a violation per broken rule (input spec §3). The
// package-level file requires identifier and title; a representation-level
// one does not.
func (r *reader) decodeMetadataCSV(src string, packageLevel bool) metadata.Terms {
	rel := r.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil
	}
	// Spreadsheet tools produce a UTF-8 BOM; accept and drop it (§3).
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	if !utf8.Valid(data) {
		r.violate("%s: not valid UTF-8 — re-export the file as UTF-8", rel)
		return nil
	}

	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1 // row width is checked per row, for a better message

	var terms metadata.Terms
	identifiers, titles := 0, 0
	headerSeen := false
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// The reader may not recover its position after a syntax
			// error; report it (the csv error names the line) and stop.
			r.violate("%s: %v", rel, err)
			break
		}
		line, _ := cr.FieldPos(0)

		if !headerSeen {
			headerSeen = true
			if isHeaderRow(row) {
				continue
			}
			// A missing header is a violation, but the row itself may be
			// data — keep decoding so its findings surface too.
			r.violate(`%s: the first row must be the header "key,value"`, rel)
		}

		if len(row) != 2 {
			r.violate("%s line %d: expected exactly two columns (key,value), got %d", rel, line, len(row))
			continue
		}
		key, value := row[0], row[1]

		element, lang, ok := r.parseKey(rel, line, key)
		if !ok {
			continue
		}
		// What a term may say (vocabulary, language tag, non-empty value)
		// is the library's rule — the same one an embedding caller hits;
		// the decoder only adds the file/line context.
		term := metadata.Term{Element: element, Lang: lang, Value: value}
		if err := term.Validate(); err != nil {
			r.violate("%s line %d: %v", rel, line, err)
			continue
		}

		switch element {
		case "dcterms:identifier":
			identifiers++
		case "dcterms:title":
			titles++
		}
		terms = append(terms, term)
	}

	if packageLevel {
		if identifiers == 0 {
			r.violate("%s: identifier is missing — the local catalog or inventory number is required", rel)
		}
		if titles == 0 {
			r.violate("%s: title is missing — a title is required", rel)
		}
	}
	// One local identifier is the package's identity; two is an ambiguity
	// the tool cannot resolve for the operator.
	if identifiers > 1 {
		r.violate("%s: identifier appears %d times — give exactly one", rel, identifiers)
	}
	return terms
}

func isHeaderRow(row []string) bool {
	return len(row) == 2 &&
		strings.EqualFold(strings.TrimSpace(row[0]), "key") &&
		strings.EqualFold(strings.TrimSpace(row[1]), "value")
}

// parseKey handles the key *syntax* of the CSV convention — the optional
// [lang] bracket and the plain-key spelling table (§3) — and returns the
// element name the key maps onto. Whether that element (or the language
// tag inside the brackets) is *valid* is metadata.Term.Validate's rule:
// prefixed keys pass through untouched for it to judge.
func (r *reader) parseKey(file string, line int, raw string) (element, lang string, ok bool) {
	key := raw
	if i := strings.IndexByte(key, '['); i >= 0 {
		if !strings.HasSuffix(key, "]") || i+2 > len(key)-1 {
			r.violate("%s line %d: malformed language tag in %q — write it like title[nl]", file, line, raw)
			return "", "", false
		}
		lang = key[i+1 : len(key)-1]
		key = key[:i]
	}

	if strings.Contains(key, ":") {
		return key, lang, true
	}
	element, known := plainKeys[strings.ToLower(key)]
	if !known {
		r.violate("%s line %d: unknown key %q — a typo would silently drop metadata; see the supported keys in the input specification", file, line, raw)
		return "", "", false
	}
	return element, lang, true
}
