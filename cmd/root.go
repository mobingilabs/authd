package cmd

import (
	goflag "flag"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:   "authd",
		Short: "authorization and authentication service for Mobingi",
		Long:  "Authorization and authentication for Mobingi.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			goflag.Parse()
		},
	}
)

func init() {
	rootCmd.AddCommand(
		VersionCmd(),
		ServeCmd(),
		KeyCmd(),
	)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func Execute() error {
	return rootCmd.Execute()
}
