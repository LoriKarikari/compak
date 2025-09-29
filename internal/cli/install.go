package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Install a package",
	Long: `Install a Docker Compose package with optional parameter customization.

Packages can be installed from local directories using the --path flag.
Parameters can be customized using the --set flag, which accepts key=value pairs.`,
	Example: `  # Install from local directory
  compak install nginx --path ./examples/nginx

  # Install with custom parameters
  compak install nginx --path ./examples/nginx --set PORT=8080 --set SERVER_NAME=localhost

  # Install with multiple parameter overrides
  compak install nginx --path ./examples/nginx \
    --set PORT=9090 \
    --set SERVER_NAME=myserver \
    --set MAX_BODY_SIZE=50m`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]
		version, err := cmd.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("failed to get version flag: %w", err)
		}

		localPath, err := cmd.Flags().GetString("path")
		if err != nil {
			return fmt.Errorf("failed to get path flag: %w", err)
		}
		setValues, err := cmd.Flags().GetStringSlice("set")
		if err != nil {
			return fmt.Errorf("failed to get set flag: %w", err)
		}

		composeCmd, err := compose.DetectComposeCommand()
		if err != nil {
			return fmt.Errorf("failed to detect compose command: %w", err)
		}

		fmt.Printf("Using compose command: %s\n", composeCmd.String())

		stateDir, err := config.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		client := pkg.NewClient(composeCmd, stateDir)
		manager := pkg.NewManager(client, composeCmd, stateDir)

		var packageToInstall *pkg.Package

		if localPath != "" {
			packageToInstall, err = manager.LoadPackageFromDir(localPath)
			if err != nil {
				return fmt.Errorf("failed to load package from %s: %w", localPath, err)
			}
		} else {
			packageToInstall = &pkg.Package{
				Name:        packageName,
				Version:     version,
				Description: "Remote package (not yet implemented)",
				Parameters:  make(map[string]pkg.Param),
				Values:      make(map[string]string),
			}

			if version == "" {
				packageToInstall.Version = "latest"
			}
		}

		if existingPkg, err := client.GetInstalledPackage(packageToInstall.Name); err == nil {
			fmt.Printf("Package %s@%s is already installed (installed: %s)\n",
				packageToInstall.Name, packageToInstall.Version, existingPkg.Package.Version)
			fmt.Println("Use 'compak uninstall' first to reinstall with different settings")
			return nil
		}

		fmt.Printf("Installing package: %s@%s\n", packageToInstall.Name, packageToInstall.Version)

		if len(packageToInstall.Parameters) > 0 {
			fmt.Println("\nAvailable parameters:")
			for name, param := range packageToInstall.Parameters {
				defaultValue := param.Default
				if packageToInstall.Values != nil {
					if override, exists := packageToInstall.Values[name]; exists {
						defaultValue = override
					}
				}
				required := ""
				if param.Required {
					required = " (required)"
				}
				fmt.Printf("  %s=%s%s - %s\n", name, defaultValue, required, param.Description)
			}
			fmt.Println()
		}

		values := make(map[string]string)
		for _, v := range setValues {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 {
				values[parts[0]] = parts[1]
				fmt.Printf("Setting %s=%s\n", parts[0], parts[1])
			}
		}

		if localPath != "" {
			return manager.DeployFromPath(*packageToInstall, values, localPath)
		}

		return manager.Deploy(*packageToInstall, values)
	},
}

func init() {
	installCmd.Flags().String("version", "", "package version to install")
	installCmd.Flags().String("path", "", "path to local package directory")
	installCmd.Flags().StringSlice("set", []string{}, "set values (e.g. --set PORT=9090 --set SERVER_NAME=myserver)")
	rootCmd.AddCommand(installCmd)
}
