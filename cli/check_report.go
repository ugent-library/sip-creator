package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// checkReport is what check prints on stdout for one input folder.
type checkReport struct {
	// folder is the input folder as the operator named it on the command line.
	folder string
	// profile is the name of the profile the folder was checked against.
	profile string
	// findings are the problems, one plain-language line each. They are the
	// input specification's violations, or the findings of the profile's
	// rules when the folder has no violations.
	findings []string
	// source is the source package input.Read returned. It is complete when
	// the folder breaks no rule of the input specification. Otherwise it
	// holds the part of the folder input.Read could read.
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

// printSummary writes what the folder holds: where the package description
// comes from, the number of representations and of files, and whether
// formats will be recorded. The counts let the operator see that the tool
// read what they prepared.
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
// rows of description.csv, from which Builder.Build generates a document,
// or a supplied document, which Builder.Build copies unchanged. It returns
// "none" when the folder has no description.
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

// formatReport names the characterization report the folder supplies, or
// says that there is none.
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
// names the folder and the count. It is a type of its own so that
// exitStatus can tell it apart from an error that left the folder
// unchecked.
type problemsFound struct {
	folder string
	count  int
}

func (e problemsFound) Error() string {
	return fmt.Sprintf("%s: %d problem(s) found", e.folder, e.count)
}

// findingLines splits err into one line per finding and returns the
// lines. Definition.ValidateSource joins the profile's findings into one
// error with errors.Join, and each part is a problem of its own.
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
