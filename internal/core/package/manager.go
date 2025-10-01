package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/LoriKarikari/compak/internal/core/compose"
	"github.com/LoriKarikari/compak/internal/core/template"
)

type Manager struct {
	client        *Client
	composeClient *compose.Client
	packagesDir   string
}

func NewManager(client *Client, composeClient *compose.Client, stateDir string) *Manager {
	return &Manager{
		client:        client,
		composeClient: composeClient,
		packagesDir:   filepath.Join(stateDir, "packages"),
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

	ctx := context.Background()
	projectName := fmt.Sprintf("compak-%s", pkg.Name)

	project, err := m.composeClient.LoadProject(packageDir, projectName)
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	fmt.Printf("Deploying %s...\n", pkg.Name)

	if err := m.composeClient.Pull(ctx, project); err != nil {
		fmt.Printf("Warning: failed to pull images: %v\n", err)
	}

	if err := m.composeClient.Up(ctx, project, true, nil); err != nil {
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
	composePath := filepath.Join(packageDir, "docker-compose.yaml")

	if sourcePath != "" {
		if err := copyDir(sourcePath, packageDir); err != nil {
			return fmt.Errorf("failed to copy package files: %w", err)
		}
	} else if pkg.Source != "" {
		fmt.Printf("Downloading compose file from %s...\n", pkg.Source)
		if err := downloadComposeFile(pkg.Source, composePath); err != nil {
			return fmt.Errorf("failed to download compose file: %w", err)
		}
	} else {
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
	dirExists := true
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		fmt.Printf("Warning: package directory not found, cleaning up metadata only\n")
		dirExists = false
	}

	if dirExists {
		ctx := context.Background()
		projectName := fmt.Sprintf("compak-%s", packageName)

		fmt.Printf("Stopping %s...\n", packageName)
		if err := m.composeClient.Down(ctx, projectName); err != nil {
			return fmt.Errorf("failed to stop services: %w", err)
		}

		fmt.Printf("Cleaning up %s...\n", packageName)
		if err := os.RemoveAll(packageDir); err != nil {
			fmt.Printf("Warning: failed to remove package directory: %v\n", err)
		}
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

	ctx := context.Background()
	projectName := fmt.Sprintf("compak-%s", packageName)

	containers, err := m.composeClient.PS(ctx, projectName)
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	var output string
	for _, container := range containers {
		output += fmt.Sprintf("%s\t%s\t%s\n",
			container.Name,
			container.State,
			container.Status,
		)
	}

	return output, nil
}

func (m *Manager) LoadPackageFromDir(dir string) (result *Package, err error) {
	if err := validatePath(dir); err != nil {
		return nil, fmt.Errorf("invalid directory path: %w", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to create root: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if pkg, err := m.loadPackageFile(root, "package.yaml", yaml.Unmarshal); err == nil {
		return pkg, nil
	}

	if pkg, err := m.loadPackageFile(root, "package.json", json.Unmarshal); err == nil {
		return pkg, nil
	}

	return nil, fmt.Errorf("package.yaml or package.json not found")
}

func (m *Manager) loadPackageFile(root *os.Root, filename string, unmarshal func([]byte, interface{}) error) (pkg *Package, err error) {
	file, err := root.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	var p Package
	if err := unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	return &p, nil
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

func downloadComposeFile(url, destPath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download from %s: status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var composeCheck map[string]interface{}
	if err := yaml.Unmarshal(data, &composeCheck); err != nil {
		return fmt.Errorf("downloaded file is not valid YAML: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

func copyFile(src, dst string) (err error) {
	if err := validatePath(src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validatePath(dst); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	srcRoot, err := os.OpenRoot(filepath.Dir(src))
	if err != nil {
		return fmt.Errorf("failed to create source root: %w", err)
	}
	defer func() {
		if closeErr := srcRoot.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	file, err := srcRoot.Open(filepath.Base(src))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	data, err := io.ReadAll(file)
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
