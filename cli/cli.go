// Package cli is the sip-creator command line: the create and check
// commands, the --profile flag that selects a profile and its mapping,
// and the configuration read from the environment and an optional .env
// file.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

var rootCmd = &cobra.Command{
	Use:   "sip-creator",
	Short: "SIP Creator CLI",
	// Execute would print a RunE error and then CheckErr prints it
	// again; silence the first so every error appears exactly once.
	SilenceErrors: true,
}

// reportViolations prints each violation in err on its own line to stderr
// and returns a one-line summary for the input folder src. An error that
// is not input.Violations is returned unchanged. create reports with it;
// check prints its own report (checkReport).
func reportViolations(cmd *cobra.Command, src string, err error) error {
	v, ok := errors.AsType[input.Violations](err)
	if !ok {
		return err
	}
	for _, line := range v {
		fmt.Fprintln(cmd.ErrOrStderr(), line)
	}
	return fmt.Errorf("%s: %d problem(s) found", src, len(v))
}

// Run executes the CLI. Configuration is read by the commands that need
// it, so check runs without it (ADR-0010). An error is printed on stderr,
// and the process exits with the status exitStatus gives it.
func Run() {
	cmd, err := rootCmd.ExecuteC()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(exitStatus(cmd, err))
}

// Exit statuses of the process. Scripts that run check rely on them, so
// their values never change.
const (
	// exitFailed is any command's status for an error, except check's.
	exitFailed = 1
	// exitProblemsFound is check's status when the folder breaks a rule
	// and the report lists the problems.
	exitProblemsFound = 1
	// exitNotChecked is check's status when it could not check the folder
	// at all: a wrong path, an unknown profile, a missing argument.
	exitNotChecked = 2
)

// exitStatus is the process exit status for err, which cmd ended with.
// check tells its two failures apart for a script that runs it; every
// other command has one status for any error.
func exitStatus(cmd *cobra.Command, err error) int {
	if cmd != checkCmd {
		return exitFailed
	}
	if _, ok := errors.AsType[problemsFound](err); ok {
		return exitProblemsFound
	}
	return exitNotChecked
}
