package cli

import (
	"github.com/caarlos0/env/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"
)

var (
	version Version
	config  Config

	rootCmd = &cobra.Command{
		Use:   "sip-creator",
		Short: "SIP Creator CLI",
	}
)

func init() {
	cobra.OnInitialize(initVersion, initConfig)
}

func initVersion() {
	cobra.CheckErr(env.Parse(&version))
}

func initConfig() {
	cobra.CheckErr(env.ParseWithOptions(&config, env.Options{
		Prefix: "SIP_CREATOR_",
	}))
}

func Run() {
	cobra.CheckErr(rootCmd.Execute())
}
