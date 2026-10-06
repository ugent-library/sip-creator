package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// checkReport is what check prints on stdout for one input folder: the
// problems it found, a summary of what the folder holds, then the verdict.
// It is written for the operator who prepares the folder.
type checkReport struct {
	// folder is the input folder as the operator named it on the command line.
	folder string
	// profile is the name of the profile the folder was checked against.
	profile string
	// findings are the problems, one plain-language line each, in the order
	// the rules ran: the input specification's, then the profile's.
	findings []string
	// source is the source package the reader returned: complete when there
	// are no findings, as far as the reader got otherwise.
	source *build.SourcePackage
}

// print writes the report to w: the problems first, because they are what
// the operator acts on, then the summary, then the verdict.
func (r checkReport) print(w io.Writer) {
	if len(r.findings) > 0 {
		fmt.Fprintf(w, "%s in %s\n\n", countProblems(len(r.findings)), r.folder)
		for _, line := range r.findings {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	r.printSummary(w)

	if len(r.findings) == 0 {
		fmt.Fprintf(w, "OK: the folder meets the input specification for profile %s.\n", r.profile)
		return
	}
	fmt.Fprintf(w, "FAILED: fix the problems listed at the top and run check again.\n")
}

// printSummary writes what the folder holds in counts, so the operator can
// see that the tool read what they prepared: where the package description
// comes from, the representations and the files in them, and whether
// formats will be recorded.
func (r checkReport) printSummary(w io.Writer) {
	src := r.source
	described, essence := 0, 0
	documentation, premis := len(src.Documentation), len(src.Premis)
	for _, rep := range src.Representations {
		if rep.Description != nil {
			described++
		}
		essence += len(rep.Files)
		documentation += len(rep.Documentation)
		premis += len(rep.Premis)
	}

	representations := fmt.Sprint(len(src.Representations))
	if described > 0 {
		representations += fmt.Sprintf(" (%d with its own description)", described)
	}

	fmt.Fprintf(w, "Input folder:         %s\n", r.folder)
	fmt.Fprintf(w, "Profile:              %s\n\n", r.profile)
	fmt.Fprintf(w, "Descriptive metadata: %s\n", descriptionSource(src.Description))
	fmt.Fprintf(w, "Representations:      %s\n", representations)
	fmt.Fprintf(w, "Essence files:        %d\n", essence)
	fmt.Fprintf(w, "Documentation files:  %d\n", documentation)
	fmt.Fprintf(w, "PREMIS files:         %d\n", premis)
	fmt.Fprintf(w, "Format report:        %s\n\n", formatReport(src))
}

// descriptionSource names where the package description comes from: the
// rows of description.csv, from which the build generates a document, or
// a supplied document, which the build copies without reading it.
func descriptionSource(description sip.Description) string {
	switch d := description.(type) {
	case nil:
		return "none"
	case build.EncodedDescription:
		return filepath.Base(d.Source) + " (supplied document, copied as it is)"
	default:
		return "description.csv"
	}
}

// formatReport says whether the folder supplies a characterization report,
// without which the package records no formats.
func formatReport(src *build.SourcePackage) string {
	if src.Characterization == nil {
		return "not supplied (files carry no format information)"
	}
	return "siegfried.json"
}

// countProblems spells a count of problems, singular for one.
func countProblems(n int) string {
	if n == 1 {
		return "1 problem"
	}
	return fmt.Sprintf("%d problems", n)
}

// problemsFound is the error check ends with when the folder breaks a
// rule. The report has already listed every problem, so the error only
// names the folder and the count. Its type sets it apart from the errors
// that leave the folder unchecked: exitProblemsFound against
// exitNotChecked.
type problemsFound struct {
	folder string
	count  int
}

func (e problemsFound) Error() string {
	return fmt.Sprintf("%s: %d problem(s) found", e.folder, e.count)
}

// findingLines splits err into one line per finding. The profile's rules
// arrive as one error joined from several (errors.Join), and each part is
// a problem of its own.
func findingLines(err error) []string {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []string{err.Error()}
	}
	var lines []string
	for _, part := range joined.Unwrap() {
		lines = append(lines, findingLines(part)...)
	}
	return lines
}
