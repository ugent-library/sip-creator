package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/sip"
)

func init() {
	addProfileFlag(createCmd)
	createCmd.Flags().Bool("no-zip", false, "Skip zipping; the package directory is the deliverable (e.g. for external bagging)")
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
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		def, mapper, err := resolveProfile(cmd)
		if err != nil {
			return err
		}

		// The submitting organization is configuration, not profile
		// data, so it is filled into the definition before building.
		def, err = def.WithSubmitter(cfg.Submitter.Name, cfg.Submitter.ORID)
		if err != nil {
			return fmt.Errorf("%w (set SIP_SUBMITTER_NAME and SIP_SUBMITTER_OR_ID)", err)
		}

		status, updates, err := recordStatusFromFlags(cmd)
		if err != nil {
			return err
		}
		contentCategory, _ := cmd.Flags().GetString("content-category")
		if contentCategory == "" {
			contentCategory = cfg.ContentCategory
		}

		source, err := input.Read(args[0], mapper, input.DocumentSpec{Name: def.DocumentName, Model: def.Model})
		if err != nil {
			return reportViolations(cmd, args[0], err)
		}

		// Progress goes to stderr, so stdout carries only the command's
		// output.
		logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
		builder, err := build.New(&build.Config{
			Profile:     def,
			Destination: args[1],
			Logger:      logger,
		})
		if err != nil {
			return err
		}

		// These values belong to this package, not to the folder. Left
		// empty, Builder.Build mints an identifier, and the profile's
		// record status and content category apply.
		source.PackageIdentifier = updates
		source.RecordStatus = status
		source.ContentCategory = contentCategory

		noZip, _ := cmd.Flags().GetBool("no-zip")
		zipper := archive.New(&archive.Config{
			Destination: args[1],
			Logger:      logger,
		})
		// An update reuses the earlier package's identifier, so its zip
		// may still be in dest. The check runs before building: if Zip
		// refused after the build, the new package directory would stay
		// behind in dest. A minted identifier is new, so there is
		// nothing to check.
		if !noZip && source.PackageIdentifier != "" {
			if err := zipper.ValidateDestination(source.PackageIdentifier); err != nil {
				return err
			}
		}

		built, err := builder.Build(source)
		if err != nil {
			return err
		}

		if noZip {
			return nil
		}
		return zipper.Zip(built)
	},
}

// recordStatusFromFlags returns the record status given with --status and
// the identifier given with --updates. The two come as a pair: an update
// status names an earlier package, and naming one needs an update status.
// It returns an error if the status is not in the SIP3 vocabulary or the
// flags do not come as a pair. The pairing is a rule of the CLI only: the
// library also accepts an identifier on a new package, for a program that
// mints identifiers itself.
func recordStatusFromFlags(cmd *cobra.Command) (sip.RecordStatus, string, error) {
	statusText, _ := cmd.Flags().GetString("status")
	updates, _ := cmd.Flags().GetString("updates")

	var status sip.RecordStatus
	if statusText != "" {
		var err error
		status, err = sip.ParseRecordStatus(statusText)
		if err != nil {
			return "", "", err
		}
	}

	switch {
	case updates != "" && !status.IsUpdate():
		return "", "", fmt.Errorf("--updates names an earlier package, which needs --status supplement, replacement, version or delete")
	case updates == "" && status.IsUpdate():
		return "", "", fmt.Errorf("--status %s updates an earlier package; pass its identifier with --updates", statusText)
	}
	return status, updates, nil
}
