package input

import (
	"fmt"
	"strings"
)

// Violations is the collected list of input-contract findings: one
// plain-language line per broken MUST rule, naming the file or folder
// concerned. All findings are gathered before reporting — deliberately the
// opposite of the library's fail-fast errors, because the audience is an
// operator fixing a folder in one pass, not a developer reading a stack.
type Violations []string

func (v Violations) Error() string {
	return strings.Join(v, "\n")
}

func (r *reader) violate(format string, args ...any) {
	r.violations = append(r.violations, fmt.Sprintf(format, args...))
}
