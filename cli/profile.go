package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
)

// addProfileFlag declares the required --profile flag on cmd. check and
// create share it: the profile says which vocabulary the folder's
// description.csv is in, so both read the folder the same way, and check
// reports exactly what create would.
func addProfileFlag(cmd *cobra.Command) {
	cmd.Flags().String("profile", "", "Profile of the SIP (one of: "+strings.Join(profiles.Names(), ", ")+")")
	_ = cmd.MarkFlagRequired("profile") // only fails for an undeclared flag
}

// resolveProfile looks the --profile value up in the registry.
func resolveProfile(cmd *cobra.Command) (build.Definition, error) {
	name, _ := cmd.Flags().GetString("profile")
	def, ok := profiles.Get(name)
	if !ok {
		return build.Definition{}, fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(profiles.Names(), ", "))
	}
	return def, nil
}
