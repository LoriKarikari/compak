package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	pkg "github.com/LoriKarikari/compak/internal/core/package"
	"github.com/LoriKarikari/compak/internal/core/registry"
)

var publishPath string

func init() {
	publishCmd.Flags().StringVar(&publishPath, "path", ".", "Path to the package directory containing package.yaml and docker-compose.yaml")
	rootCmd.AddCommand(publishCmd)
}

var publishCmd = &cobra.Command{
	Use:   "publish [REGISTRY/NAME:TAG]",
	Short: "Publish a package to an OCI registry",
	Long: `Publish a compak package to an OCI registry like Docker Hub or GitHub Container Registry.

Examples:
  compak publish ghcr.io/user/mypackage:1.0.0
  compak publish ghcr.io/user/mypackage:latest --path ./mypackage
  compak publish docker.io/user/mypackage:v2.1.0 --path /path/to/package`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reference := args[0]
		return publishPackage(reference, publishPath)
	},
}

func publishPackage(reference, packagePath string) error {
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	packageFile := filepath.Join(absPath, "package.yaml")
	composeFile := filepath.Join(absPath, "docker-compose.yaml")

	if _, err := os.Stat(packageFile); os.IsNotExist(err) {
		return fmt.Errorf("package.yaml not found in %s", absPath)
	}
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yaml not found in %s", absPath)
	}

	cleanPackageFile := filepath.Clean(packageFile)
	if !strings.HasPrefix(cleanPackageFile, absPath+string(filepath.Separator)) && cleanPackageFile != absPath {
		return fmt.Errorf("package file path outside directory")
	}

	data, err := os.ReadFile(packageFile)
	if err != nil {
		return fmt.Errorf("failed to read package.yaml: %w", err)
	}

	var packageConfig pkg.Package
	if err := yaml.Unmarshal(data, &packageConfig); err != nil {
		return fmt.Errorf("failed to parse package.yaml: %w", err)
	}

	fmt.Printf("Publishing package '%s' version %s to %s\n", packageConfig.Name, packageConfig.Version, reference)
	fmt.Printf("Package description: %s\n", packageConfig.Description)

	client := registry.NewClient()
	ctx := context.Background()

	fmt.Println("Pushing package to registry...")
	if err := client.Push(ctx, absPath, reference); err != nil {
		return fmt.Errorf("failed to publish package: %w", err)
	}

	fmt.Printf("✓ Successfully published package to %s\n", reference)
	fmt.Println("\nTo install this package, run:")
	fmt.Printf("  compak install %s\n", reference)

	return nil
}
