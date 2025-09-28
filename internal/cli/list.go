package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		composeCmd, err := compose.DetectComposeCommand()
		if err != nil {
			return fmt.Errorf("failed to detect compose command: %w", err)
		}

		stateDir, err := config.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		client := pkg.NewClient(composeCmd, stateDir)

		packages, err := client.List()
		if err != nil {
			return fmt.Errorf("failed to list packages: %w", err)
		}

		if len(packages) == 0 {
			fmt.Println("No packages installed")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tINSTALLED"); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}

		for _, pkg := range packages {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				pkg.Package.Name,
				pkg.Package.Version,
				pkg.Status,
				pkg.InstallTime.Format("2006-01-02 15:04:05"),
			); err != nil {
				return fmt.Errorf("failed to write package info: %w", err)
			}
		}

		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
