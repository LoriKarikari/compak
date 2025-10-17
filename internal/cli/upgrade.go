package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	"github.com/LoriKarikari/compak/internal/core/index"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [package]",
	Short: "Upgrade an installed package to a newer version",
	Long: `Upgrade an installed package to the latest or specified version.

For packages with pinned versions (versioned compose files), this will upgrade to a newer release.
For packages with floating versions (unversioned compose files), this updates the package metadata
while preserving your parameter settings.`,
	Example: `  # Upgrade to latest version
  compak upgrade immich

  # Upgrade to specific version
  compak upgrade immich --version 1.145.0

  # Upgrade all packages
  compak upgrade --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return fmt.Errorf("failed to get all flag: %w", err)
		}

		targetVersion, err := cmd.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("failed to get version flag: %w", err)
		}

		if all {
			return upgradeAll(ctx, targetVersion)
		}

		if len(args) == 0 {
			return fmt.Errorf("package name required (or use --all)")
		}

		return upgradePackage(ctx, args[0], targetVersion)
	},
}

func upgradePackage(ctx context.Context, packageName, targetVersion string) error {
	if err := validatePackageName(packageName); err != nil {
		return err
	}

	stateDir, err := config.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}

	client := pkg.NewClient(stateDir)

	installedPkg, err := client.GetInstalledPackage(packageName)
	if err != nil {
		return fmt.Errorf("package %s is not installed: %w", packageName, err)
	}

	latestPkg, err := fetchLatestPackage(ctx, packageName, targetVersion)
	if err != nil {
		return err
	}

	if shouldUpgrade, reason := compareVersions(installedPkg.Package.Version, latestPkg.Version); !shouldUpgrade {
		fmt.Printf("Package %s is already %s\n", packageName, reason)
		return nil
	}

	fmt.Printf("Upgrading %s: %s → %s\n", packageName, installedPkg.Package.Version, latestPkg.Version)

	composeClient, err := compose.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create compose client: %w", err)
	}

	manager := pkg.NewManager(client, composeClient, stateDir)

	return performUpgrade(manager, packageName, &installedPkg, latestPkg)
}

func fetchLatestPackage(ctx context.Context, packageName, targetVersion string) (pkg.Package, error) {
	if targetVersion != "" {
		if _, err := semver.NewVersion(targetVersion); err != nil && targetVersion != "latest" {
			return pkg.Package{}, fmt.Errorf("invalid target version %q: %w", targetVersion, err)
		}
	}

	indexClient := index.NewClient()
	lookupName := packageName
	if targetVersion != "" {
		lookupName = fmt.Sprintf("%s@%s", packageName, targetVersion)
	}

	packageData, err := indexClient.LoadPackageFromIndex(ctx, lookupName)
	if err != nil {
		return pkg.Package{}, fmt.Errorf("failed to load package from index: %w", err)
	}

	var latestPkg pkg.Package
	if err := yaml.Unmarshal(packageData, &latestPkg); err != nil {
		return pkg.Package{}, fmt.Errorf("failed to parse package: %w", err)
	}

	return latestPkg, nil
}

func performUpgrade(manager *pkg.Manager, packageName string, installedPkg *pkg.InstalledPackage, latestPkg pkg.Package) error {
	oldPkg := installedPkg.Package
	values := installedPkg.Values

	if err := manager.Stop(packageName); err != nil {
		return fmt.Errorf("failed to stop old version (aborting upgrade): %w", err)
	}

	if err := manager.Deploy(latestPkg, values); err != nil {
		fmt.Printf("Deployment failed, attempting rollback to %s...\n", oldPkg.Version)
		if rollbackErr := manager.Deploy(oldPkg, values); rollbackErr != nil {
			return fmt.Errorf("failed to deploy upgraded package: %w (rollback also failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("deployment failed, successfully rolled back to %s: %w", oldPkg.Version, err)
	}

	fmt.Printf("Successfully upgraded %s to %s\n", packageName, latestPkg.Version)
	return nil
}

func upgradeAll(ctx context.Context, targetVersion string) error {
	stateDir, err := config.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}

	client := pkg.NewClient(stateDir)
	packages, err := client.List()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed")
		return nil
	}

	fmt.Printf("Upgrading %d package(s)...\n", len(packages))

	var upgraded, skipped, failed int
	var failures []string

	for i, installedPkg := range packages {
		fmt.Printf("\n[%d/%d] Checking %s...\n", i+1, len(packages), installedPkg.Package.Name)
		err := upgradePackage(ctx, installedPkg.Package.Name, targetVersion)

		switch {
		case err == nil:
			upgraded++
		case strings.Contains(err.Error(), "already"):
			skipped++
		default:
			failed++
			failures = append(failures, fmt.Sprintf("%s: %v", installedPkg.Package.Name, err))
			fmt.Printf("  Failed: %v\n", err)
		}
	}

	fmt.Printf("\n%d upgraded, %d skipped, %d failed\n", upgraded, skipped, failed)
	if len(failures) > 0 {
		fmt.Println("\nFailed packages:")
		for _, failure := range failures {
			fmt.Printf("  - %s\n", failure)
		}
		return fmt.Errorf("%d package(s) failed to upgrade", failed)
	}
	return nil
}

func compareVersions(installed, latest string) (shouldUpgrade bool, reason string) {
	if installed == latest {
		return false, "up to date"
	}

	if latest == "latest" {
		return true, ""
	}

	installedVer, err1 := semver.NewVersion(installed)
	latestVer, err2 := semver.NewVersion(latest)

	if err1 != nil || err2 != nil {
		return true, ""
	}

	if latestVer.GreaterThan(installedVer) {
		return true, ""
	}

	if latestVer.LessThan(installedVer) {
		return false, fmt.Sprintf("would downgrade (%s → %s), use --force to downgrade", installed, latest)
	}

	return false, "up to date"
}

func init() {
	upgradeCmd.Flags().String("version", "", "target version to upgrade to")
	upgradeCmd.Flags().Bool("all", false, "upgrade all installed packages")
	rootCmd.AddCommand(upgradeCmd)
}
