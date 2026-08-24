package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

// checkCmd validates a folder against every input rule without building
// anything. It deliberately needs no configuration: input rules are
// config-independent (ADR-0010).
var checkCmd = &cobra.Command{
	Use:          "check [src]",
	Short:        "Check an input folder against the input specification without building",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // findings are the output, not a usage error
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, err := input.Read(args[0])
		if err != nil {
			if v, ok := errors.AsType[input.Violations](err); ok {
				for _, line := range v {
					fmt.Fprintln(cmd.ErrOrStderr(), line)
				}
				return fmt.Errorf("%s: %d problem(s) found", args[0], len(v))
			}
			return err
		}

		for _, w := range pkg.Warnings {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", w)
		}

		files := 0
		for _, r := range pkg.Representations {
			files += len(r.Files)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "OK: %d representation(s), %d content file(s), %d documentation file(s)\n",
			len(pkg.Representations), files, len(pkg.Documentation))
		return nil
	},
}
