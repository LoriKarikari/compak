package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print version, commit hash, build date and other build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("compak %s (%s)\n", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
