package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/formats"
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

		// No configured tool means format identification is skipped: the
		// builder treats a nil identificator as "don't enrich".
		var ffid formats.Identificator
		if config.Formats.Name != "" {
			var err error
			ffid, err = formats.New(config.Formats.Name, config.Formats.Command, config.Formats.Args)
			if err != nil {
				return err
			}
		}

		builder := profiles.New(&profiles.Config{
			Source:      args[0],
			Destination: args[1],
			Logger:      logger,
			Formats:     ffid,
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
			archive.Zip(pkg)
		}

		return nil
	},
}
