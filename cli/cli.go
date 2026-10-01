// Package cli is the sip-creator command line: the create and check
// commands, the --profile flag that selects a profile and its vocabulary,
// and the configuration read from the environment and an optional .env
// file.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

var (
	logger *slog.Logger

	rootCmd = &cobra.Command{
		Use:   "sip-creator",
		Short: "SIP Creator CLI",
		// Execute would print a RunE error and then CheckErr prints it
		// again; silence the first so every error appears exactly once.
		SilenceErrors: true,
	}
)

// newLogger logs to stderr, so stdout carries only a command's output.
func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// reportViolations prints each violation in err on its own line to stderr
// and returns a one-line summary for the input folder src. An error that
// is not input.Violations is returned unchanged.
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
// it, so check runs without it (ADR-0010).
func Run() {
	logger = newLogger()

	cobra.CheckErr(rootCmd.Execute())
}
