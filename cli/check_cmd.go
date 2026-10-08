package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/cli/input"
)

func init() {
	addProfileFlag(checkCmd)
	rootCmd.AddCommand(checkCmd)
}

// checkCmd checks an input folder without building a package. It applies
// the input specification's rules, then the profile's rules
// (Definition.ValidateSource) to the source package the folder holds.
// Checks that read every file, such as the characterization report's
// checksums, run only in create. It reads no configuration (ADR-0010).
var checkCmd = &cobra.Command{
	Use:          "check [src]",
	Short:        "Check an input folder against the input specification without building",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // findings are the output, not a usage error
	RunE: func(cmd *cobra.Command, args []string) error {
		def, mapper, err := resolveProfile(cmd)
		if err != nil {
			return err
		}

		report := checkReport{folder: args[0], profile: def.Name}
		source, err := input.Read(args[0], mapper, input.DocumentSpec{Name: def.DocumentName, Model: def.Model})
		report.source = source
		if violations, ok := errors.AsType[input.Violations](err); ok {
			report.findings = violations
		} else if err != nil {
			return err // the folder could not be read at all
		} else {
			// The profile's rules run only on a folder read without
			// violations. On a partly read folder they would report what
			// is missing only because input.Read could not read it.
			report.findings = findingLines(def.ValidateSource(source))
		}

		report.print(cmd.OutOrStdout())
		if len(report.findings) > 0 {
			return problemsFound{folder: args[0], count: len(report.findings)}
		}
		return nil
	},
}
