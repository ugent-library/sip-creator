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

// decode reads the files the walk found into source, in the order of the
// input specification: the sidecar and representations.csv (§2), then the
// description of each level (§3). mapper decodes a description.csv, and
// format checks a supplied document.
func (r *folderReader) decode(source *build.SourcePackage, inv inventory, mapper Mapper, format build.DocumentFormat) {
	if inv.sidecar != "" {
		source.Characterization = r.decodeSidecar(inv.sidecar)
	}
	if source.Characterization != nil {
		r.checkCharacterizationCoversContent(source.Characterization, source.Representations)
	}
	if inv.representationsCSV != "" {
		source.Representations = r.applyRepresentations(inv.representationsCSV, source.Representations)
	}
	source.Description = r.description(inv.pkg, true, mapper, format)
	for i := range source.Representations {
		rep := &source.Representations[i]
		rep.Description = r.description(inv.reps[rep.Name], false, mapper, format)
	}
}

// decodeSidecar decodes the pre-computed characterization report at src
// (ADR-0009). It records a violation and returns nil if the file cannot be
// opened or does not parse.
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

// checkCharacterizationCoversContent records a violation for each content
// file the report has no entry for. It repeats a rule of Builder.Build, so
// the check command reports what create would refuse. Documentation files
// need no entry. When two or more content files all lack an entry, the
// report was most likely made from another folder, so one violation with
// an example key replaces one per file.
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

// applyRepresentations decodes the representations.csv at src and applies
// it to reps, the representations read from representations/. Each row
// names a representation folder and supplies its label and type. Every row
// must match a folder, and every folder must have a row (input
// specification §2). It returns the representations in row order, which
// becomes the packaging order. An empty cell stays empty, because
// build.SourceRepresentation resolves the defaults.
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

	// A folder without a row is a violation, because leaving it out would
	// drop its content from the package. It stays in the list, so decode
	// still reads its description.
	for _, rep := range reps {
		if _, ok := covered[rep.Name]; !ok {
			r.violate("representations/%s is not listed in %s; add a row for it, or remove the folder", rep.Name, rel)
			ordered = append(ordered, rep)
		}
	}
	return ordered
}

// decodeRepresentations reads the representations.csv at src into rows and
// records a violation for each broken rule. It returns decoded false if
// the file cannot be used at all. A usable file without data rows gives an
// empty slice.
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
// recorded for it: the description.csv, mapped by mapper, or the supplied
// document, read as it is and checked against format. It returns nil when
// the level has neither.
func (r *folderReader) description(files descriptionFiles, packageLevel bool, mapper Mapper, format build.DocumentFormat) sip.Description {
	switch {
	case files.document != "":
		return r.readDocument(files.document, format)
	case files.rows != "":
		return r.decodeDescription(files.rows, packageLevel, mapper)
	}
	return nil
}

// decodeDescription decodes the description.csv at src into the profile's
// description and records a violation for each broken rule: the row
// syntax, each term the mapper cannot place, and the description's
// Validate. At the package level it also runs ValidateRequired. It returns
// nil when the file holds no terms.
func (r *folderReader) decodeDescription(src string, packageLevel bool, mapper Mapper) sip.Description {
	rel := r.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil
	}

	terms, lines, errs, err := parseTerms(data)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil
	}
	description, mapErrs := mapper.Map(terms)
	errs = append(errs, mapErrs...)
	// A term the mapper refused is reported once. The mapper keeps it as
	// written, so the description's rules would judge it again.
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

	// A finding about one row is reported at the row's line. The parser
	// names the line of a row that did not become a term. The mapper and
	// the description's rules name a term by its index, and lines maps
	// the index to its line. A finding about several rows names the key
	// and language, which locate the rows in the file.
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
		return nil
	}
	return description
}

// readDocument reads the supplied descriptive document at src as the
// level's description. It checks that the file parses as XML and that
// format accepts its root element, the same check Definition.ValidateSource
// makes. It checks nothing else in the document (ADR-0003). It returns a
// build.EncodedDescription for src, also when format refuses the root
// element. It records a violation and returns nil if the file cannot be
// opened or is not XML. format is never nil here, because the walk finds a
// document only under a profile that takes one.
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
