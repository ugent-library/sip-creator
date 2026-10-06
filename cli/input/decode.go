package input

import (
	"errors"
	"maps"
	"os"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// decode reads the files the walk found, in the order of the input
// specification: the sidecar and representations.csv (§2), then the
// description of each level (§3). Only the description decoder needs the
// profile: the mapper for description.csv, the format for a supplied
// document. It returns the details of the read it can fill in: the rows
// as written and whether representations.csv was read.
func (r *folderReader) decode(source *build.SourcePackage, inv inventory, mapper Mapper, format build.DocumentFormat) ReadDetails {
	if inv.sidecar != "" {
		source.Characterization = r.decodeSidecar(inv.sidecar)
	}
	if source.Characterization != nil {
		r.checkCharacterizationCoversContent(source.Characterization, source.Representations)
	}
	if inv.representationsCSV != "" {
		source.Representations = r.applyRepresentations(inv.representationsCSV, source.Representations)
	}

	details := ReadDetails{RepresentationsCSV: inv.representationsCSV != ""}
	source.Description, details.PackageRows = r.description(inv.pkg, true, mapper, format)
	for i := range source.Representations {
		rep := &source.Representations[i]
		var rows []sip.Term
		rep.Description, rows = r.description(inv.reps[rep.Name], false, mapper, format)
		if rows != nil {
			if details.RepresentationRows == nil {
				details.RepresentationRows = map[string][]sip.Term{}
			}
			details.RepresentationRows[rep.Name] = rows
		}
	}
	return details
}

// decodeSidecar decodes the optional pre-computed characterization report.
// A present report must parse (ADR-0009);
// checkCharacterizationCoversContent looks up the content files in it, and
// the assembler verifies each entry's MD5.
func (r *folderReader) decodeSidecar(src string) characterization.Report {
	f, err := os.Open(src)
	if err != nil {
		r.violate("siegfried.json: %v", err)
		return nil
	}
	defer f.Close()

	report, err := characterization.DecodeSiegfried(f)
	if err != nil {
		r.violate("siegfried.json: %v; regenerate it from the input root with: sf -hash md5 -json .", err)
		return nil
	}
	return report
}

// checkCharacterizationCoversContent reports each content file the report
// has no entry for. It mirrors the assembler's rule (Builder.essenceRecord in
// build), which a program building a source package in Go meets there, so
// that check reports what create would refuse. It looks up paths only: the
// MD5 comparison, which reads every file, stays with the assembler.
// Documentation files need no entry. When no content file has an entry,
// the report was most likely made from another folder, so one line with an
// example key replaces a line per file.
func (r *folderReader) checkCharacterizationCoversContent(report characterization.Report, reps []build.SourceRepresentation) {
	const regenerate = "regenerate it from the input root with: sf -hash md5 -json ."

	var missing []string
	total := 0
	for _, rep := range reps {
		for _, f := range rep.Files {
			total++
			if _, ok := report[f.Key]; !ok {
				missing = append(missing, f.Key)
			}
		}
	}

	switch {
	case len(missing) == 0:
		return
	case len(report) == 0:
		r.violate("siegfried.json lists no files; %s", regenerate)
	case len(missing) > 1 && len(missing) == total:
		r.violate("siegfried.json has no entry for any content file (its paths look like %q); %s", firstKey(report), regenerate)
	default:
		for _, key := range missing {
			r.violate("siegfried.json has no entry for %s; %s", key, regenerate)
		}
	}
}

// firstKey returns the report's first key in sorted order, so a message
// that shows an example key is the same from run to run.
func firstKey(report characterization.Report) string {
	return slices.Sorted(maps.Keys(report))[0]
}

// applyRepresentations decodes representations.csv and applies it to the
// representations read from representations/: each row names a
// representation folder and supplies its label and type. The file is strict when present
// (input-spec.md): every row must match a folder, every folder must be
// covered by a row, and the row order becomes the packaging order. Empty
// cells stay empty: build.SourceRepresentation resolves the defaults.
func (r *folderReader) applyRepresentations(src string, reps []build.SourceRepresentation) []build.SourceRepresentation {
	rel := r.rel(src)
	rows, decoded := r.decodeRepresentations(src)
	if !decoded {
		return reps
	}
	if len(rows) == 0 {
		r.violate("%s: the file has no rows; list every representation folder, or delete the file", rel)
		return reps
	}

	byName := make(map[string]int, len(reps))
	for i, rep := range reps {
		byName[rep.Name] = i
	}

	covered := make(map[string]int, len(rows)) // folder → line of its row
	var ordered []build.SourceRepresentation
	for _, row := range rows {
		if prev, ok := covered[row.folder]; ok {
			r.violate("%s line %d: folder %q already has a row (line %d)", rel, row.line, row.folder, prev)
			continue
		}
		covered[row.folder] = row.line
		i, ok := byName[row.folder]
		if !ok {
			r.violate("%s line %d: there is no folder representations/%s; every row must name an existing representation folder", rel, row.line, row.folder)
			continue
		}
		rep := reps[i]
		rep.Label = row.label
		rep.Type = row.kind
		ordered = append(ordered, rep)
	}

	// A folder the file does not cover must fail loudly: skipping it
	// would silently drop content from the package.
	for _, rep := range reps {
		if _, ok := covered[rep.Name]; !ok {
			r.violate("representations/%s is not listed in %s; add a row for it, or remove the folder", rep.Name, rel)
			ordered = append(ordered, rep)
		}
	}
	return ordered
}

// decodeRepresentations reads the representations.csv at src into rows and
// records a violation per broken rule. Returns decoded=false when the file
// cannot be used at all; a usable file with no data rows returns an empty
// slice.
func (r *folderReader) decodeRepresentations(src string) (rows []repRow, decoded bool) {
	rel := r.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil, false
	}

	rows, errs, err := parseRepresentationRows(data)
	if err != nil {
		for _, e := range flatten(err) {
			r.violate("%s: %v", rel, e)
		}
		return nil, false
	}
	for _, e := range errs {
		if re, ok := errors.AsType[*rowError](e); ok {
			r.violate("%s line %d: %v", rel, re.line, re.err)
			continue
		}
		r.violate("%s: %v", rel, e)
	}
	return rows, true
}

// description returns the level's description from the one file the walk
// recorded for it: the rows file, mapped by the profile's mapper, or the
// profile's supplied document, read as it is and judged by format. It
// returns nil when the level has neither. rows are the rows as written
// when the description comes from a rows file.
func (r *folderReader) description(files descriptionFiles, packageLevel bool, mapper Mapper, format build.DocumentFormat) (description sip.Description, rows []sip.Term) {
	switch {
	case files.document != "":
		return r.readDocument(files.document, format), nil
	case files.rows != "":
		return r.decodeDescription(files.rows, packageLevel, mapper)
	}
	return nil, nil
}

// decodeDescription decodes the description.csv at src into the profile's
// description and records a violation per broken rule: the row syntax,
// the mapper's placement of each term, and the description's own
// rules, with ValidateRequired at the package level only. It also returns
// the rows that parsed, as written, before the mapper placed them.
func (r *folderReader) decodeDescription(src string, packageLevel bool, mapper Mapper) (sip.Description, []sip.Term) {
	rel := r.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil, nil
	}

	terms, lines, errs, err := parseTerms(data)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil, nil
	}
	// A mapper may return a description that shares the terms' backing
	// array (the eark mapping is the identity), and the build may rewrite a
	// description in place (build.IdentifierSwapper), so the rows as
	// written are a copy.
	rows := slices.Clone(terms)
	description, mapErrs := mapper.Map(terms)
	errs = append(errs, mapErrs...)
	// A term the mapper refused is reported once: the description's rules
	// would judge the same term again, as written.
	refused := map[int]bool{}
	for _, err := range mapErrs {
		if te, ok := errors.AsType[*sip.TermError](err); ok {
			refused[te.Index] = true
		}
	}
	for _, err := range flatten(description.Validate()) {
		if te, ok := errors.AsType[*sip.TermError](err); ok && refused[te.Index] {
			continue
		}
		errs = append(errs, err)
	}
	if packageLevel {
		errs = append(errs, flatten(description.ValidateRequired())...)
	}

	// A finding about one row is reported at the row's line: the parser
	// names the line of a row that did not become a term, and the
	// mapper and the description's rules name a term by its index,
	// which lines turns back into a line. A cross-row finding names the
	// key and language, which locates the rows in a keyed file.
	for _, err := range errs {
		if re, ok := errors.AsType[*rowError](err); ok {
			r.violate("%s line %d: %v", rel, re.line, re.err)
			continue
		}
		if te, ok := errors.AsType[*sip.TermError](err); ok && te.Index < len(lines) {
			r.violate("%s line %d: %v", rel, lines[te.Index], te.Err)
			continue
		}
		r.violate("%s: %v", rel, err)
	}
	if len(terms) == 0 {
		return nil, nil
	}
	return description, rows
}

// readDocument reads the supplied descriptive document at src as the
// level's description: a file the package copies as it is. It must parse
// as XML (xmldoc.Root) with a root in the model's document format, which
// the model judges as the engine will. Nothing else in the document is checked (ADR-0003):
// schema validity is left to the validators downstream. format is never
// nil here: the walk only finds a document under a profile that takes one.
func (r *folderReader) readDocument(src string, format build.DocumentFormat) sip.Description {
	rel := r.rel(src)

	f, err := os.Open(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil
	}
	defer f.Close()

	root, err := xmldoc.Root(f)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil
	}
	if err := format.ValidateDocumentRoot(root); err != nil {
		r.violate("%s: %v", rel, err)
	}
	return build.EncodedDescription{Source: src}
}
