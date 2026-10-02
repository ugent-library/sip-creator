package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

func init() {
	addProfileFlag(checkCmd)
	rootCmd.AddCommand(checkCmd)
}

// checkCmd validates an input folder without building. Checks on file
// contents (received PREMIS, the characterization report) run only in
// create. It reads no configuration (ADR-0010).
var checkCmd = &cobra.Command{
	Use:          "check [src]",
	Short:        "Check an input folder against the input specification without building",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // findings are the output, not a usage error
	RunE: func(cmd *cobra.Command, args []string) error {
		_, vocabulary, err := resolveProfile(cmd)
		if err != nil {
			return err
		}

		source, err := input.Read(args[0], vocabulary)
		if err != nil {
			return reportViolations(cmd, args[0], err)
		}

		files := 0
		for _, r := range source.Representations {
			files += len(r.Files)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "OK: %d representation(s), %d content file(s), %d documentation file(s)\n",
			len(source.Representations), files, len(source.Documentation))
		return nil
	},
}
