package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/LoriKarikari/compak/internal/core/compose"
	"github.com/LoriKarikari/compak/internal/core/template"
)

type Manager struct {
	client      *Client
	composeCmd  *compose.ComposeCommand
	packagesDir string
}

func NewManager(client *Client, composeCmd *compose.ComposeCommand, stateDir string) *Manager {
	return &Manager{
		client:      client,
		composeCmd:  composeCmd,
		packagesDir: filepath.Join(stateDir, "packages"),
	}
}

func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("package name contains invalid characters")
	}
	if len(name) > 100 {
		return fmt.Errorf("package name too long")
	}
	return nil
}

func validatePath(path string) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal")
	}
	return nil
}

func (m *Manager) Deploy(pkg Package, values map[string]string) error {
	return m.DeployFromPath(pkg, values, "")
}

func (m *Manager) DeployFromPath(pkg Package, values map[string]string, sourcePath string) error {
	if err := m.validatePackageAndPath(pkg.Name, sourcePath); err != nil {
		return err
	}

	packageDir := filepath.Join(m.packagesDir, pkg.Name)
	if err := os.MkdirAll(packageDir, 0o750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	mergedValues := m.client.mergeValues(pkg, values)
	if err := m.setupPackageFiles(packageDir, sourcePath, pkg, mergedValues); err != nil {
		return err
	}

	fmt.Printf("Deploying %s...\n", pkg.Name)
	if err := m.composeCmd.ExecuteIn(packageDir, "up", "-d"); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	return m.client.Install(pkg, mergedValues)
}

func (m *Manager) validatePackageAndPath(packageName, sourcePath string) error {
	if err := validatePackageName(packageName); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}
	if sourcePath != "" {
		if err := validatePath(sourcePath); err != nil {
			return fmt.Errorf("invalid source path: %w", err)
		}
	}
	return nil
}

func (m *Manager) setupPackageFiles(packageDir, sourcePath string, pkg Package, mergedValues map[string]string) error {
	if sourcePath != "" {
		if err := copyDir(sourcePath, packageDir); err != nil {
			return fmt.Errorf("failed to copy package files: %w", err)
		}
	} else {
		composePath := filepath.Join(packageDir, "docker-compose.yaml")
		if err := m.writeComposeFile(composePath, pkg); err != nil {
			return fmt.Errorf("failed to write compose file: %w", err)
		}
	}

	engine := template.NewEngine(mergedValues)
	if err := engine.WriteEnvFile(packageDir); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

func (m *Manager) Stop(packageName string) error {
	if err := validatePackageName(packageName); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	installedPkg, err := m.client.GetInstalledPackage(packageName)
	if err != nil {
		return fmt.Errorf("package not found: %w", err)
	}

	packageDir := filepath.Join(m.packagesDir, packageName)
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return fmt.Errorf("package directory not found: %s", packageDir)
	}

	fmt.Printf("Stopping %s...\n", packageName)
	if err := m.composeCmd.ExecuteIn(packageDir, "down"); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	fmt.Printf("Cleaning up %s...\n", packageName)
	if err := os.RemoveAll(packageDir); err != nil {
		fmt.Printf("Warning: failed to remove package directory: %v\n", err)
	}

	return m.client.Uninstall(installedPkg.Package.Name)
}

func (m *Manager) Status(packageName string) (string, error) {
	if err := validatePackageName(packageName); err != nil {
		return "", fmt.Errorf("invalid package name: %w", err)
	}

	packageDir := filepath.Join(m.packagesDir, packageName)
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return "", fmt.Errorf("package not found")
	}

	output, err := m.composeCmd.ExecuteQuiet(packageDir, "ps")
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return output, nil
}

func (m *Manager) LoadPackageFromDir(dir string) (*Package, error) {
	if err := validatePath(dir); err != nil {
		return nil, fmt.Errorf("invalid directory path: %w", err)
	}

	packageFile := filepath.Join(dir, "package.yaml")
	data, err := os.ReadFile(packageFile)
	if err != nil {
		packageFile = filepath.Join(dir, "package.json")
		data, err = os.ReadFile(packageFile)
		if err != nil {
			return nil, fmt.Errorf("package.yaml or package.json not found")
		}

		var pkg Package
		if err := json.Unmarshal(data, &pkg); err != nil {
			return nil, fmt.Errorf("failed to parse package.json: %w", err)
		}
		return &pkg, nil
	}

	var pkg Package
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.yaml: %w", err)
	}

	return &pkg, nil
}

func (m *Manager) writeComposeFile(path string, _ Package) error {
	composeContent := `version: '3.8'

services:
  app:
    image: nginx:alpine
    ports:
      - "${PORT:-8080}:80"
    environment:
      - SERVICE_NAME=${SERVICE_NAME:-compak}
`

	return os.WriteFile(path, []byte(composeContent), 0o600)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
