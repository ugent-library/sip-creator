package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/cli/input/mapping"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/ugent"
)

// mappers pairs each registered profile with the mapping of its
// description.csv. It is keyed by the definitions' own Name fields, so
// renaming a profile cannot leave an entry under the old name.
var mappers = map[string]input.Mapper{
	meemoo.Definition.Name:   mapping.Meemoo{},
	ugent.Basic.Name:         mapping.SimpleDC{},
	ugent.Bibliographic.Name: mapping.MODS{},
}

// addProfileFlag declares the required --profile flag on cmd.
func addProfileFlag(cmd *cobra.Command) {
	cmd.Flags().String("profile", "", "Profile of the SIP (one of: "+strings.Join(profiles.Names(), ", ")+")")
	_ = cmd.MarkFlagRequired("profile") // only fails for an undeclared flag
}

// resolveProfile looks the --profile value up in the registry and returns
// its definition with the mapper that maps a folder's rows onto its
// description. It returns an error if the profile is not registered or
// has no mapper. A registered profile without a mapper is a programming
// error, not the operator's. TestEveryProfileHasAMappingItsModelAccepts
// checks that every profile has one.
func resolveProfile(cmd *cobra.Command) (build.Definition, input.Mapper, error) {
	name, _ := cmd.Flags().GetString("profile")
	def, ok := profiles.Get(name)
	if !ok {
		return build.Definition{}, nil, fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(profiles.Names(), ", "))
	}
	mapper, ok := mappers[name]
	if !ok {
		return build.Definition{}, nil, fmt.Errorf("profile %q has no mapping for description.csv", name)
	}
	return def, mapper, nil
}
