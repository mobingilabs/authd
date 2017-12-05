package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "?"

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information.",
		Long:  `Print the version information.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(version)
		},
	}

	return cmd
}
