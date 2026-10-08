package input

import (
	"fmt"
	"strings"
)

// Violations lists the broken MUST rules of the input specification, one
// plain-language line per finding, naming the file or folder concerned.
// It holds every finding of one read, unlike the library's fail-fast
// errors, so an operator can fix the folder in one pass.
type Violations []string

func (v Violations) Error() string {
	return strings.Join(v, "\n")
}

func (r *folderReader) violate(format string, args ...any) {
	r.violations = append(r.violations, fmt.Sprintf(format, args...))
}
