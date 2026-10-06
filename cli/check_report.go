package cli

import (
	"fmt"
	"io"
)

// checkReport is what check prints on stdout for one input folder: the
// problems it found, then the verdict. It is written for the operator who
// prepares the folder.
type checkReport struct {
	// folder is the input folder as the operator named it on the command line.
	folder string
	// profile is the name of the profile the folder was checked against.
	profile string
	// findings are the problems, one plain-language line each, in the order
	// the rules ran: the input specification's, then the profile's.
	findings []string
}

// print writes the report to w.
func (r checkReport) print(w io.Writer) {
	if len(r.findings) == 0 {
		fmt.Fprintf(w, "OK: the folder meets the input specification for profile %s.\n", r.profile)
		return
	}

	fmt.Fprintf(w, "%s in %s\n\n", countProblems(len(r.findings)), r.folder)
	for _, line := range r.findings {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintf(w, "\nFAILED: fix the problems above and run check again.\n")
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
