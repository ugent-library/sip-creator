package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/profiles"
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

		src := args[0]
		dest := args[1]

		profile := profiles.New(src, dest)

		switch flagProfile {
		case "basic":
			profile.Basic()
		default:
			return errors.New("no sip profile was set")
		}

		return nil
	},
}
