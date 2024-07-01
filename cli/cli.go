package cli

import (
	"log/slog"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
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
	cobra.CheckErr(godotenv.Load())

	config = services.ConfigFromEnv()
	logger = services.NewLogger(config)

	cobra.CheckErr(rootCmd.Execute())
}
