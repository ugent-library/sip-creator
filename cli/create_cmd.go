package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/formats"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/sip"
)

func init() {
	createCmd.Flags().String("profile", "", "Set the profile of the SIP")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create [src] [dest]",
	Short: "Create a new SIP package",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		flagProfile, _ := cmd.Flags().GetString("profile")

		ffid, err := formats.New(config.Formats.Name, config.Formats.Command, config.Formats.Args)
		if err != nil {
			return err
		}

		profile := profiles.New(&profiles.Config{
			Source:      args[0],
			Destination: args[1],
			Logger:      logger,
			Formats:     ffid,
		})

		archive := archive.New(&archive.Config{
			Destination: args[1],
			Logger:      logger,
		})

		var pkg *sip.Package

		switch flagProfile {
		case "basic":
			pkg = profile.Basic()
		default:
			return errors.New("no sip profile was set")
		}

		if pkg != nil {
			archive.Zip(pkg)
		}

		return nil
	},
}
