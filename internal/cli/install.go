package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
	"github.com/LoriKarikari/compak/internal/core/registry"
)

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Install a package",
	Long: `Install a Docker Compose package with optional parameter customization.

Packages can be installed from:
- OCI registries (e.g., ghcr.io/user/package:version)
- Local directories using the --path flag

Parameters can be customized using the --set flag, which accepts key=value pairs.`,
	Example: `  # Install from OCI registry
  compak install ghcr.io/compak/nginx:1.0.0
  compak install docker.io/myuser/wordpress:latest

  # Install from local directory
  compak install nginx --path ./examples/nginx

  # Install with custom parameters
  compak install nginx --path ./examples/nginx --set PORT=8080 --set SERVER_NAME=localhost

  # Install with multiple parameter overrides
  compak install ghcr.io/compak/nginx:1.0.0 \
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

		composeClient, err := compose.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create compose client: %w", err)
		}

		stateDir, err := config.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		client := pkg.NewClient(stateDir)
		manager := pkg.NewManager(client, composeClient, stateDir)

		packageToInstall, sourcePath, err := loadPackage(packageName, version, localPath, manager)
		if err != nil {
			return err
		}

		if sourcePath != "" && registry.IsRegistryReference(packageName) {
			defer func() {
				if err := os.RemoveAll(sourcePath); err != nil {
					fmt.Printf("Warning: failed to clean up temp directory: %v\n", err)
				}
			}()
		}

		if existingPkg, err := client.GetInstalledPackage(packageToInstall.Name); err == nil {
			fmt.Printf("Package %s@%s is already installed (installed: %s)\n",
				packageToInstall.Name, packageToInstall.Version, existingPkg.Package.Version)
			fmt.Println("Use 'compak uninstall' first to reinstall with different settings")
			return nil
		}

		fmt.Printf("Installing package: %s@%s\n", packageToInstall.Name, packageToInstall.Version)

		displayPackageInfo(packageToInstall)

		values := parseSetValues(setValues)

		if sourcePath != "" {
			return manager.DeployFromPath(*packageToInstall, values, sourcePath)
		}

		return manager.Deploy(*packageToInstall, values)
	},
}

func loadPackage(packageName, version, localPath string, manager *pkg.Manager) (*pkg.Package, string, error) {
	if localPath != "" {
		packageToInstall, err := manager.LoadPackageFromDir(localPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load package from %s: %w", localPath, err)
		}
		return packageToInstall, localPath, nil
	}

	if registry.IsRegistryReference(packageName) {
		fmt.Printf("Pulling package from registry: %s\n", packageName)

		registryClient := registry.NewClient()
		tempDir := filepath.Join(os.TempDir(), "compak-pull", fmt.Sprintf("%d", os.Getpid()))

		if err := registryClient.Pull(context.Background(), packageName, tempDir); err != nil {
			return nil, "", fmt.Errorf("failed to pull package: %w", err)
		}

		packageToInstall, err := manager.LoadPackageFromDir(tempDir)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load pulled package: %w", err)
		}
		return packageToInstall, tempDir, nil
	}

	packageToInstall := &pkg.Package{
		Name:        packageName,
		Version:     version,
		Description: "Local package",
		Parameters:  make(map[string]pkg.Param),
		Values:      make(map[string]string),
	}

	if version == "" {
		packageToInstall.Version = "latest"
	}

	return packageToInstall, "", nil
}

func displayPackageInfo(pkg *pkg.Package) {
	if len(pkg.Parameters) > 0 {
		fmt.Println("\nAvailable parameters:")
		for name, param := range pkg.Parameters {
			defaultValue := param.Default
			if pkg.Values != nil {
				if override, exists := pkg.Values[name]; exists {
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
}

func parseSetValues(setValues []string) map[string]string {
	parsed := lo.FilterMap(setValues, func(v string, _ int) (lo.Entry[string, string], bool) {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			fmt.Printf("Setting %s=%s\n", parts[0], parts[1])
			return lo.Entry[string, string]{Key: parts[0], Value: parts[1]}, true
		}
		return lo.Entry[string, string]{}, false
	})

	return lo.FromEntries(parsed)
}

func init() {
	installCmd.Flags().String("version", "", "package version to install")
	installCmd.Flags().String("path", "", "path to local package directory")
	installCmd.Flags().StringSlice("set", []string{}, "set values (e.g. --set PORT=9090 --set SERVER_NAME=myserver)")
	rootCmd.AddCommand(installCmd)
}
