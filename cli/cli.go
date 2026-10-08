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
	// Run prints a command's error itself, so cobra must not print it too.
	SilenceErrors: true,
}

// reportViolations prints each violation in err on its own line to stderr
// and returns an error that names the input folder src and the number of
// violations. An error that is not input.Violations is returned
// unchanged.
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

// Run executes the CLI. On an error it prints the error on stderr and
// exits the process with the status exitStatus returns.
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

// exitStatus returns the process exit status for err, the error cmd ended
// with. For check it tells a folder with problems apart from a folder it
// could not check. Every other command has one status for any error.
func exitStatus(cmd *cobra.Command, err error) int {
	if cmd != checkCmd {
		return exitFailed
	}
	if _, ok := errors.AsType[problemsFound](err); ok {
		return exitProblemsFound
	}
	return exitNotChecked
}
