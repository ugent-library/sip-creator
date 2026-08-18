package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/profiles"
)

func init() {
	createCmd.Flags().String("profile", "", "Set the profile of the SIP")
	createCmd.Flags().Bool("no-zip", false, "Skip zipping; the package directory is the deliverable (e.g. for external bagging, see ADR-0008)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create [src] [dest]",
	Short: "Create a new SIP package",
	Args:  cobra.ExactArgs(2),
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

		// Format info comes from the input tree's siegfried.json sidecar
		// when present (ADR-0009); the CLI never runs a characterization
		// tool, so there is nothing to construct or configure here.
		builder := profiles.New(&profiles.Config{
			Source:      args[0],
			Destination: args[1],
			Logger:      logger,
		})

		archive := archive.New(&archive.Config{
			Destination: args[1],
			Logger:      logger,
		})

		pkg, err := builder.Build(def)
		if err != nil {
			return err
		}

		if noZip, _ := cmd.Flags().GetBool("no-zip"); !noZip {
			return archive.Zip(pkg)
		}

		return nil
	},
}
