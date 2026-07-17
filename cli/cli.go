package cli

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	_ "github.com/ugent-library/sip-creator/formats/siegfried"
)

var (
	cfg    *config
	logger *slog.Logger

	rootCmd = &cobra.Command{
		Use:   "sip-creator",
		Short: "SIP Creator CLI",
	}
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func Run() {
	// .env is optional configuration: a missing file is fine (all vars are
	// optional), a malformed one is an error.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		cobra.CheckErr(err)
	}

	var err error
	cfg, err = configFromEnv()
	cobra.CheckErr(err)

	logger = newLogger()

	cobra.CheckErr(rootCmd.Execute())
}
