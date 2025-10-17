package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	"github.com/LoriKarikari/compak/internal/core/index"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Install a package",
	Long: `Install a Docker Compose package with optional parameter customization.

Packages can be installed from:
- Curated index (e.g., compak install nginx)
- Local directories using the --path flag

Parameters can be customized using the --set flag, which accepts key=value pairs.`,
	Example: `  # Install from curated index
  compak install nginx
  compak install immich@1.144

  # Install from local directory
  compak install nginx --path ./examples/nginx

  # Install with custom parameters
  compak install nginx --set PORT=8080 --set SERVER_NAME=localhost

  # Install with multiple parameter overrides
  compak install immich \
    --set DB_PASSWORD=secure123 \
    --set UPLOAD_LOCATION=/mnt/photos`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		packageName := args[0]

		if err := validatePackageName(packageName); err != nil {
			return err
		}

		version, err := cmd.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("failed to get version flag: %w", err)
		}

		localPath, err := cmd.Flags().GetString("path")
		if err != nil {
			return fmt.Errorf("failed to get path flag: %w", err)
		}

		if localPath != "" {
			normalizedPath, err := validateLocalPath(localPath)
			if err != nil {
				return err
			}
			localPath = normalizedPath
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

		packageToInstall, sourcePath, err := loadPackage(ctx, packageName, version, localPath, manager)
		if err != nil {
			return err
		}

		existingPkg, err := client.GetInstalledPackage(packageToInstall.Name)
		if err == nil {
			if existingPkg.Package.Version == packageToInstall.Version {
				fmt.Printf("Package %s@%s is already installed\n",
					packageToInstall.Name, packageToInstall.Version)
				fmt.Println("Use 'compak upgrade' to update or 'compak uninstall' to reinstall")
				return nil
			}

			return fmt.Errorf("package %s is already installed with version %s (requested: %s). Use 'compak upgrade' to update or 'compak uninstall' first",
				packageToInstall.Name, existingPkg.Package.Version, packageToInstall.Version)
		}

		fmt.Printf("Installing package: %s@%s\n", packageToInstall.Name, packageToInstall.Version)

		displayPackageInfo(packageToInstall)

		values, err := parseSetValues(setValues)
		if err != nil {
			return err
		}

		if err := validateParameters(packageToInstall, values); err != nil {
			return fmt.Errorf("parameter validation failed: %w", err)
		}

		if sourcePath != "" {
			return manager.DeployFromPath(*packageToInstall, values, sourcePath)
		}

		return manager.Deploy(*packageToInstall, values)
	},
}

func loadPackage(ctx context.Context, packageName, version, localPath string, manager *pkg.Manager) (*pkg.Package, string, error) {
	if localPath != "" {
		return loadFromLocalPath(localPath, manager)
	}

	return loadFromIndex(ctx, packageName, version)
}

func validateLocalPath(localPath string) (string, error) {
	if !filepath.IsAbs(localPath) {
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
		localPath = absPath
	}

	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("local path does not exist: %w", err)
	}

	return localPath, nil
}

func loadFromLocalPath(localPath string, manager *pkg.Manager) (*pkg.Package, string, error) {
	packageToInstall, err := manager.LoadPackageFromDir(localPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load package from %s: %w", localPath, err)
	}
	return packageToInstall, localPath, nil
}

func loadFromIndex(ctx context.Context, packageName, version string) (*pkg.Package, string, error) {
	lookupName := packageName
	if version != "" && !strings.Contains(packageName, "@") {
		lookupName = fmt.Sprintf("%s@%s", packageName, version)
	}

	indexClient := index.NewClient()
	packageData, err := indexClient.LoadPackageFromIndex(ctx, lookupName)
	if err != nil {
		return nil, "", fmt.Errorf("package %q not found in index: %w", lookupName, err)
	}

	var packageToInstall pkg.Package
	if err := yaml.Unmarshal(packageData, &packageToInstall); err != nil {
		return nil, "", fmt.Errorf("failed to parse package from index: %w", err)
	}

	fmt.Printf("Loaded %s from index (source: %s)\n", lookupName, packageToInstall.Source)
	return &packageToInstall, "", nil
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

func validateParameters(pkg *pkg.Package, values map[string]string) error {
	var errors []string

	for key := range values {
		if _, exists := pkg.Parameters[key]; !exists {
			errors = append(errors, fmt.Sprintf("unknown parameter: %s", key))
		}
	}

	for key, param := range pkg.Parameters {
		if param.Required {
			_, hasValue := values[key]
			_, hasDefault := pkg.Values[key]
			if !hasValue && !hasDefault && param.Default == "" {
				errors = append(errors, fmt.Sprintf("required parameter missing: %s", key))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

func parseSetValues(setValues []string) (map[string]string, error) {
	var errors []string
	parsed := lo.FilterMap(setValues, func(v string, _ int) (lo.Entry[string, string], bool) {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 && parts[0] != "" {
			fmt.Printf("Setting %s=%s\n", parts[0], parts[1])
			return lo.Entry[string, string]{Key: parts[0], Value: parts[1]}, true
		}
		errors = append(errors, v)
		return lo.Entry[string, string]{}, false
	})

	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid --set values (must be KEY=VALUE): %v", errors)
	}

	return lo.FromEntries(parsed), nil
}

func init() {
	installCmd.Flags().String("version", "", "package version to install")
	installCmd.Flags().String("path", "", "path to local package directory")
	installCmd.Flags().StringSlice("set", []string{}, "set values (e.g. --set PORT=9090 --set SERVER_NAME=myserver)")
	rootCmd.AddCommand(installCmd)
}
