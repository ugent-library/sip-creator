package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

func init() {
	addProfileFlag(checkCmd)
	rootCmd.AddCommand(checkCmd)
}

// checkCmd validates a folder against every input rule without building
// anything. It needs no configuration: input rules are independent of
// installation settings (ADR-0010). It does take the profile, because the
// profile says which vocabulary description.csv is in; a folder checks
// out for the profile it will be built with.
var checkCmd = &cobra.Command{
	Use:          "check [src]",
	Short:        "Check an input folder against the input specification without building",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // findings are the output, not a usage error
	RunE: func(cmd *cobra.Command, args []string) error {
		def, err := resolveProfile(cmd)
		if err != nil {
			return err
		}

		pkg, err := input.ReadDirectory(args[0], def)
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
