// Package characterization decodes a pre-computed characterization report,
// the siegfried.json file an operator generates with `sf -hash md5 -json`,
// into one record per file (ADR-0009). It keeps the report's facts as they
// are: format, media type, checksum and the tool's error for each file. It
// does not judge them, because a report on a whole folder can hold entries
// that no package uses. Package build judges a record when it looks up a
// file for the package.
package characterization

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// Record is one file's characterization facts as the report asserts them.
type Record struct {
	// Format is the format the report asserts, or nil when the tool found
	// no match.
	Format *sip.Format
	// Mime is the IANA media type the report asserts, or empty when it
	// asserts none.
	Mime string
	// MD5 is the hex digest that ties the record to the bytes it describes.
	MD5 string
	// Errors is the tool's error for this file, verbatim, or empty when
	// there is none.
	Errors string
}

// Report maps each file's path, relative to the input folder and with
// slash separators, to its characterization record.
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

// DecodeSiegfried decodes a siegfried JSON report into a Report. It returns
// an error if r is not JSON or has no top-level siegfried version.
func DecodeSiegfried(r io.Reader) (Report, error) {
	var out sfOutput
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, fmt.Errorf("parse siegfried report: %w", err)
	}
	// Any JSON object decodes into sfOutput without error. The version
	// string shows that the input is a siegfried report.
	if out.Siegfried == "" {
		return nil, errors.New(`not a siegfried report: missing top-level "siegfried" version`)
	}

	report := make(Report, len(out.Files))
	for _, f := range out.Files {
		rec := Record{MD5: f.MD5, Errors: f.Errors}
		// Only the first match is used. A non-nil Format always carries a
		// registry, because the PREMIS template reads FormatRegistry
		// without a nil check.
		if len(f.Matches) > 0 {
			m := f.Matches[0]
			fr := sip.NewFormatRegistry()
			fr.Name = m.NS
			fr.Key = m.ID
			rec.Format = &sip.Format{FormatRegistry: fr}
			rec.Mime = m.Mime
		}
		// sf records each path as it was given, which may start with ./
		// and has backslashes when sf ran on Windows. Report keys are
		// relative paths with slashes. Backslashes become slashes on every
		// platform, so a report generated on Windows matches the files on
		// macOS or Linux. A file name that itself contains a backslash then
		// matches no entry.
		report[path.Clean(strings.ReplaceAll(f.Filename, `\`, "/"))] = rec
	}
	return report, nil
}
