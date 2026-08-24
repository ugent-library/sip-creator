package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/sip"
)

func init() {
	createCmd.Flags().String("profile", "", "Set the profile of the SIP")
	createCmd.Flags().Bool("no-zip", false, "Skip zipping; the package directory is the deliverable (e.g. for external bagging, see ADR-0008)")
	createCmd.Flags().String("status", "", "Record status of the package (SIP3 vocabulary: new, supplement, replacement, test, version, delete); omitted means new")
	createCmd.Flags().String("updates", "", "Identifier of the package this one updates; reused as this package's identifier (mets/@OBJID)")
	createCmd.Flags().String("content-category", "", "Content category of the package (mets/@TYPE, CSIP vocabulary); overrides SIP_CONTENT_CATEGORY and the profile default")
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

		// --status and --updates are coupled: an update-class status names
		// an earlier package, and naming one requires an update-class
		// status. Strict pairing is CLI policy; library
		// callers may supply a package identifier for other reasons.
		flagStatus, _ := cmd.Flags().GetString("status")
		updates, _ := cmd.Flags().GetString("updates")
		status := strings.ToUpper(flagStatus)
		if status != "" {
			if err := sip.ValidateRecordStatus(status); err != nil {
				return err
			}
		}
		switch {
		case updates != "" && !sip.IsUpdateRecordStatus(status):
			return fmt.Errorf("--updates names an earlier package, which needs --status supplement, replacement, version or delete")
		case updates == "" && sip.IsUpdateRecordStatus(status):
			return fmt.Errorf("--status %s updates an earlier package; pass its identifier with --updates", strings.ToLower(status))
		}
		def.Mets.RecordStatus = status

		// Content category precedence: flag, then configured default, then
		// the profile's registry value.
		if cc, _ := cmd.Flags().GetString("content-category"); cc != "" {
			def.Mets.Type = cc
		} else if cfg.ContentCategory != "" {
			def.Mets.Type = cfg.ContentCategory
		}

		pkg, err := input.Read(args[0])
		if err != nil {
			return fmt.Errorf("input folder %s does not conform to the input specification:\n%w", args[0], err)
		}

		builder := profiles.New(&profiles.Config{
			Destination: args[1],
			Logger:      logger,
		})

		zipper := archive.New(&archive.Config{
			Destination: args[1],
			Logger:      logger,
		})

		in := pkg.BuilderInput()
		in.PackageIdentifier = updates

		built, err := builder.Build(def, in)
		if err != nil {
			return err
		}

		if noZip, _ := cmd.Flags().GetBool("no-zip"); !noZip {
			return zipper.Zip(built)
		}

		return nil
	},
}
