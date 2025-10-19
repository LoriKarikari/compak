package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var createCmd = &cobra.Command{
	Use:   "create [path]",
	Short: "Create a package.yaml from an existing docker-compose.yaml",
	Long: `Analyze a docker-compose.yaml file and generate a package.yaml template.

This command will:
- Parse your docker-compose.yaml file
- Extract environment variables used in the compose file
- Detect ports, volumes, and other configurable parameters
- Generate a package.yaml template with detected parameters
- Prompt you for package metadata (name, description, etc.)

The generated package.yaml will need manual review and editing before use.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCreate,
}

var (
	createOutputFile  string
	createInteractive bool
	createPackageName string
	createDescription string
	createAuthor      string
	createHomepage    string
	createRepository  string
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&createOutputFile, "output", "o", "package.yaml", "Output file for generated package.yaml")
	createCmd.Flags().BoolVarP(&createInteractive, "interactive", "i", true, "Interactive mode for package metadata")
	createCmd.Flags().StringVar(&createPackageName, "name", "", "Package name (non-interactive)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Package description (non-interactive)")
	createCmd.Flags().StringVar(&createAuthor, "author", "", "Package author (non-interactive)")
	createCmd.Flags().StringVar(&createHomepage, "homepage", "", "Package homepage URL (non-interactive)")
	createCmd.Flags().StringVar(&createRepository, "repository", "", "Package repository URL (non-interactive)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	composePath, err := findComposeFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to find docker-compose.yaml: %w", err)
	}

	fmt.Printf("Found docker-compose file: %s\n", composePath)

	c, err := pkg.NewCreator()
	if err != nil {
		return fmt.Errorf("failed to create package creator: %w", err)
	}

	fmt.Println("\nAnalyzing docker-compose.yaml...")
	template, err := c.AnalyzeComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to analyze compose file: %w", err)
	}

	if createInteractive {
		if err := promptPackageMetadata(template); err != nil {
			return fmt.Errorf("failed to get package metadata: %w", err)
		}
	} else {
		if createPackageName != "" {
			template.Name = createPackageName
		}
		if createDescription != "" {
			template.Description = createDescription
		}
		if createAuthor != "" {
			template.Author = createAuthor
		}
		if createHomepage != "" {
			template.Homepage = createHomepage
		}
		if createRepository != "" {
			template.Repository = createRepository
		}
	}

	output, err := c.GeneratePackageYAML(template)
	if err != nil {
		return fmt.Errorf("failed to generate package.yaml: %w", err)
	}

	outputPath := createOutputFile
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(filepath.Dir(composePath), outputPath)
	}

	if err := os.WriteFile(outputPath, []byte(output), 0o600); err != nil {
		return fmt.Errorf("failed to write package.yaml: %w", err)
	}

	fmt.Printf("\n✓ Generated package.yaml at: %s\n", outputPath)

	return nil
}

func findComposeFile(targetPath string) (string, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		if strings.Contains(filepath.Base(targetPath), "compose") {
			return targetPath, nil
		}
		return "", fmt.Errorf("%s is not a docker-compose file", targetPath)
	}

	candidates := []string{
		"docker-compose.yaml",
		"docker-compose.yml",
		"compose.yaml",
		"compose.yml",
	}

	for _, candidate := range candidates {
		path := filepath.Join(targetPath, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no docker-compose.yaml file found in %s", targetPath)
}

func promptPackageMetadata(template *pkg.PackageTemplate) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nPackage Metadata")
	fmt.Println("================")

	if template.Name == "" {
		fmt.Print("\nPackage name: ")
		name, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		template.Name = strings.TrimSpace(name)
	}

	if !isValidPackageName(template.Name) {
		return fmt.Errorf("invalid package name: must be lowercase alphanumeric with hyphens only")
	}

	if template.Version == "" {
		fmt.Print("Version [1.0.0]: ")
		version, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		version = strings.TrimSpace(version)
		if version == "" {
			version = "1.0.0"
		}
		template.Version = version
	}

	fmt.Print("Description: ")
	desc, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	template.Description = strings.TrimSpace(desc)

	fmt.Print("Author (optional): ")
	author, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	template.Author = strings.TrimSpace(author)

	fmt.Print("Homepage URL (optional): ")
	homepage, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	template.Homepage = strings.TrimSpace(homepage)

	fmt.Print("Repository URL (optional): ")
	repo, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	template.Repository = strings.TrimSpace(repo)

	return nil
}

func isValidPackageName(name string) bool {
	matched, err := regexp.MatchString(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`, name)
	if err != nil {
		return false
	}
	return matched
}
