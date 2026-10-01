package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/cli/input/vocabulary"
	"github.com/ugent-library/sip-creator/profiles"
)

// addProfileFlag declares the required --profile flag on cmd. check and
// create share it: the profile says which vocabulary the folder's
// description.csv is in.
func addProfileFlag(cmd *cobra.Command) {
	cmd.Flags().String("profile", "", "Profile of the SIP (one of: "+strings.Join(profiles.Names(), ", ")+")")
	_ = cmd.MarkFlagRequired("profile") // only fails for an undeclared flag
}

// resolveProfile looks the --profile value up in the registry and returns
// its definition with the vocabulary that gives a folder's rows their
// meaning under it. A registered profile without a vocabulary is a
// programming error, not the operator's; a test in cli/input/vocabulary
// pins that every profile has one.
func resolveProfile(cmd *cobra.Command) (build.Definition, input.Vocabulary, error) {
	name, _ := cmd.Flags().GetString("profile")
	def, ok := profiles.Get(name)
	if !ok {
		return build.Definition{}, nil, fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(profiles.Names(), ", "))
	}
	vocab, ok := vocabulary.For(name)
	if !ok {
		return build.Definition{}, nil, fmt.Errorf("profile %q has no vocabulary for description.csv", name)
	}
	return def, vocab, nil
}
