package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"
)

func NewClient(stateDir string) *Client {
	return &Client{
		stateDir: stateDir,
	}
}

func (c *Client) safeReadFile(path string) (data []byte, err error) {
	root, err := os.OpenRoot(c.stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create root: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	relPath, err := filepath.Rel(c.stateDir, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	file, err := root.Open(relPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return io.ReadAll(file)
}

func (c *Client) Install(pkg Package, values map[string]string) error {
	if err := c.ensureStateDir(); err != nil {
		return fmt.Errorf("failed to ensure state directory: %w", err)
	}

	mergedValues := c.mergeValues(pkg, values)

	if err := c.validateParameters(pkg.Parameters, mergedValues); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	installedPkg := InstalledPackage{
		Package:     pkg,
		InstallTime: time.Now(),
		Values:      mergedValues,
		Status:      "installed",
	}

	if err := c.saveInstalledPackage(installedPkg); err != nil {
		return fmt.Errorf("failed to save package state: %w", err)
	}

	fmt.Printf("Successfully installed %s@%s\n", pkg.Name, pkg.Version)
	return nil
}

func (c *Client) Uninstall(packageName string) error {
	installedPkg, err := c.GetInstalledPackage(packageName)
	if err != nil {
		return fmt.Errorf("package not found: %w", err)
	}

	if err := c.removeInstalledPackage(packageName); err != nil {
		return fmt.Errorf("failed to remove package: %w", err)
	}

	fmt.Printf("Successfully uninstalled %s@%s\n", installedPkg.Package.Name, installedPkg.Package.Version)
	return nil
}

func (c *Client) List() ([]InstalledPackage, error) {
	if err := c.ensureStateDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure state directory: %w", err)
	}

	stateFile := filepath.Join(c.stateDir, "installed.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return []InstalledPackage{}, nil
	}

	data, err := c.safeReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if err := validateStateFile(data); err != nil {
		return nil, fmt.Errorf("corrupted state file: %w", err)
	}

	var state map[string]InstalledPackage
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return lo.Values(state), nil
}

func (c *Client) ensureStateDir() error {
	return os.MkdirAll(c.stateDir, 0o750)
}

func (c *Client) mergeValues(pkg Package, overrides map[string]string) map[string]string {
	defaults := lo.MapEntries(pkg.Parameters, func(name string, param Param) (string, string) {
		return name, param.Default
	})

	defaults = lo.PickBy(defaults, func(k, v string) bool {
		return v != ""
	})

	return lo.Assign(defaults, pkg.Values, overrides)
}

func (c *Client) validateParameters(params map[string]Param, values map[string]string) error {
	for name, param := range params {
		value, exists := values[name]
		if param.Required && (!exists || value == "") {
			return fmt.Errorf("required parameter '%s' is missing", name)
		}
		if exists && value != "" {
			if err := validateParameterValue(name, value, param); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateParameterValue(name, value string, param Param) error {
	if len(value) > 1000 {
		return fmt.Errorf("parameter '%s' value too long (max 1000 characters)", name)
	}

	if strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("parameter '%s' contains invalid characters", name)
	}

	switch param.Type {
	case "string":
		return nil
	case "number":
		matched, err := regexp.MatchString(`^-?\d+(\.\d+)?$`, value)
		if err != nil {
			return fmt.Errorf("parameter '%s' regex error: %w", name, err)
		}
		if !matched {
			return fmt.Errorf("parameter '%s' must be a valid number", name)
		}
	case "boolean":
		matched, err := regexp.MatchString(`^(true|false|yes|no|1|0)$`, strings.ToLower(value))
		if err != nil {
			return fmt.Errorf("parameter '%s' regex error: %w", name, err)
		}
		if !matched {
			return fmt.Errorf("parameter '%s' must be a boolean value (true/false)", name)
		}
	case "port":
		matched, err := regexp.MatchString(`^([1-9]\d{0,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$`, value)
		if err != nil {
			return fmt.Errorf("parameter '%s' regex error: %w", name, err)
		}
		if !matched {
			return fmt.Errorf("parameter '%s' must be a valid port number (1-65535)", name)
		}
	}

	return nil
}

func (c *Client) saveInstalledPackage(pkg InstalledPackage) error {
	stateFile := filepath.Join(c.stateDir, "installed.json")

	var state map[string]InstalledPackage
	if data, err := c.safeReadFile(stateFile); err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("failed to parse existing state file: %w", err)
		}
	}
	if state == nil {
		state = make(map[string]InstalledPackage)
	}

	state[pkg.Package.Name] = pkg

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0o600)
}

func (c *Client) GetInstalledPackage(name string) (InstalledPackage, error) {
	stateFile := filepath.Join(c.stateDir, "installed.json")
	data, err := c.safeReadFile(stateFile)
	if err != nil {
		return InstalledPackage{}, err
	}

	if err := validateStateFile(data); err != nil {
		return InstalledPackage{}, fmt.Errorf("corrupted state file: %w", err)
	}

	var state map[string]InstalledPackage
	if err := json.Unmarshal(data, &state); err != nil {
		return InstalledPackage{}, err
	}

	pkg, exists := state[name]
	if !exists {
		return InstalledPackage{}, fmt.Errorf("package '%s' not found", name)
	}

	if err := validateInstalledPackage(pkg); err != nil {
		return InstalledPackage{}, fmt.Errorf("invalid package data: %w", err)
	}

	return pkg, nil
}

func (c *Client) removeInstalledPackage(name string) error {
	stateFile := filepath.Join(c.stateDir, "installed.json")
	data, err := c.safeReadFile(stateFile)
	if err != nil {
		return err
	}

	var state map[string]InstalledPackage
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	delete(state, name)

	newData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, newData, 0o600)
}

func validateStateFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if len(data) > 10*1024*1024 {
		return fmt.Errorf("state file too large")
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("invalid JSON format")
	}

	return nil
}

func validateInstalledPackage(pkg InstalledPackage) error {
	if pkg.Package.Name == "" {
		return fmt.Errorf("package name is empty")
	}

	if len(pkg.Package.Name) > 100 {
		return fmt.Errorf("package name too long")
	}

	if strings.ContainsAny(pkg.Package.Name, "/\\..") {
		return fmt.Errorf("package name contains invalid characters")
	}

	if pkg.Package.Version == "" {
		return fmt.Errorf("package version is empty")
	}

	if pkg.Status == "" {
		return fmt.Errorf("package status is empty")
	}

	allowedStatuses := map[string]bool{
		"installed": true,
		"failed":    true,
		"updating":  true,
	}

	if !allowedStatuses[pkg.Status] {
		return fmt.Errorf("invalid package status: %s", pkg.Status)
	}

	return nil
}
