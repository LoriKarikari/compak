package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "compak",
	Short: "Package manager for Docker Compose applications",
	Long: `Compak is a CLI tool for managing Docker Compose applications as packages.

Compak allows you to install, manage, and deploy multi-container applications
using a simple package format. It supports both Docker Compose and Podman Compose,
automatically detecting the best available compose command.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: false,
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
