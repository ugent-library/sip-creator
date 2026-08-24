// Package characterization decodes pre-computed file-characterization
// reports (the siegfried.json sidecar an operator generates with
// `sf -hash md5 -json`) into per-file records the assembler consumes.
// Decoding carries the report's facts (format, mime, checksum, per-file
// tool errors) without judging them: a whole-tree report legitimately
// contains entries no consumer ever looks up, so strictness policy
// belongs to the consumer, which knows which entries it needs.
package characterization

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"

	"github.com/ugent-library/sip-creator/sip"
)

// Record is one file's characterization facts as the report asserts them.
type Record struct {
	// Format is the asserted format; nil when the tool ran and found no
	// match.
	Format *sip.Format
	// Mime is the IANA media type the report asserts; empty when it
	// asserts none.
	Mime string
	// MD5 is the hex digest that ties the record to the bytes it describes.
	MD5 string
	// Errors is the tool's per-file error, verbatim; empty means none.
	Errors string
}

// Report maps input-relative slash paths to their characterization records.
type Report map[string]Record

// sfOutput mirrors the report `sf -hash md5 -json` emits.
type sfOutput struct {
	Siegfried string    `json:"siegfried"`
	Files     []*sfFile `json:"files"`
}

type sfFile struct {
	Filename string     `json:"filename"`
	Filesize int64      `json:"filesize"`
	Errors   string     `json:"errors"`
	MD5      string     `json:"md5"`
	Matches  []*sfMatch `json:"matches"`
}

type sfMatch struct {
	NS      string `json:"ns"`
	ID      string `json:"id"`
	Format  string `json:"format"`
	Version string `json:"version"`
	Mime    string `json:"mime"`
	Class   string `json:"class"`
	Basis   string `json:"basis"`
	Warning string `json:"warning"`
}

// DecodeSiegfried decodes a siegfried JSON report into a Report.
func DecodeSiegfried(r io.Reader) (Report, error) {
	var out sfOutput
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, fmt.Errorf("parse siegfried report: %w", err)
	}
	// Any JSON object decodes into sfOutput without error; the version
	// string is the discriminator proving this is a siegfried report.
	if out.Siegfried == "" {
		return nil, errors.New(`not a siegfried report: missing top-level "siegfried" version`)
	}

	report := make(Report, len(out.Files))
	for _, f := range out.Files {
		rec := Record{MD5: f.MD5, Errors: f.Errors}
		// First match wins; the tool takes no view on ambiguous reports.
		// The registry is always set inside a non-nil Format: the premis
		// template dereferences FormatRegistry unguarded.
		if len(f.Matches) > 0 {
			m := f.Matches[0]
			fr := sip.NewFormatRegistry()
			fr.Name = m.NS
			fr.Key = m.ID
			rec.Format = &sip.Format{FormatRegistry: fr}
			rec.Mime = m.Mime
		}
		// sf records paths as invoked (possibly ./-prefixed, backslashed
		// on Windows); consumers look up input-relative slash paths.
		report[path.Clean(filepath.ToSlash(f.Filename))] = rec
	}
	return report, nil
}
