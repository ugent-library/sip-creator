package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
)

func init() {
	createCmd.Flags().String("profile", "", "Set the profile of the SIP")
	createCmd.Flags().Bool("no-zip", false, "Skip zipping; the package directory is the deliverable (e.g. for external bagging, see ADR-0008)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:          "create [src] [dest]",
	Short:        "Create a new SIP package",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true, // a bad input folder is not a usage error
	RunE: func(cmd *cobra.Command, args []string) error {
		flagProfile, _ := cmd.Flags().GetString("profile")

		def, ok := profiles.Get(flagProfile)
		if !ok {
			return fmt.Errorf("unknown profile %q (available: %s)",
				flagProfile, strings.Join(profiles.Names(), ", "))
		}

		// The submitting organization is deployment config, not profile
		// data: fill it into the definition before building.
		def, err := def.WithSubmitter(cfg.Submitter.Name, cfg.Submitter.ORID)
		if err != nil {
			return fmt.Errorf("%w (set SIP_SUBMITTER_NAME and SIP_SUBMITTER_OR_ID)", err)
		}

		pkg, err := input.Read(args[0])
		if err != nil {
			return fmt.Errorf("input folder %s does not conform to the input specification:\n%w", args[0], err)
		}
		reportUnsupported(pkg)

		builder := profiles.New(&profiles.Config{
			Destination: args[1],
			Logger:      logger,
		})

		archive := archive.New(&archive.Config{
			Destination: args[1],
			Logger:      logger,
		})

		built, err := builder.Build(def, pkg.BuilderInput())
		if err != nil {
			return err
		}

		if noZip, _ := cmd.Flags().GetBool("no-zip"); !noZip {
			return archive.Zip(built)
		}

		return nil
	},
}

// reportUnsupported warns about legal input the tool cannot emit yet, so
// nothing an operator prepared vanishes without a trace. Each warning
// disappears when its emission step lands (input-convention plan I5/I6).
func reportUnsupported(pkg *input.Package) {
	if len(pkg.Premis) > 0 {
		logger.Warn("premis/ found but received-PREMIS pass-through is not supported yet; the files are not packaged")
	}
	for _, r := range pkg.Representations {
		if r.Descriptive != nil {
			logger.Warn("representation metadata.csv is not supported yet; it is not packaged", "representation", r.Label)
		}
		if len(r.Documentation) > 0 {
			logger.Warn("representation documentation/ is not supported yet; the files are not packaged", "representation", r.Label)
		}
		if len(r.Premis) > 0 {
			logger.Warn("representation premis/ is not supported yet; the files are not packaged", "representation", r.Label)
		}
	}
}
