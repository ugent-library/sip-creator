package cli

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	_ "github.com/ugent-library/sip-creator/formats/siegfried"
	"github.com/ugent-library/sip-creator/services"
)

var (
	config *services.Config
	logger *slog.Logger

	rootCmd = &cobra.Command{
		Use:   "sip-creator",
		Short: "SIP Creator CLI",
	}
)

func Run() {
	// .env is optional configuration: a missing file is fine (all vars are
	// optional), a malformed one is an error.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		cobra.CheckErr(err)
	}

	var err error
	config, err = services.ConfigFromEnv()
	cobra.CheckErr(err)

	logger = services.NewLogger(config)

	cobra.CheckErr(rootCmd.Execute())
}
