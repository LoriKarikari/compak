package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func NewClient(composeCmd ComposeCommand, stateDir string) *Client {
	return &Client{
		composeCmd: composeCmd,
		stateDir:   stateDir,
	}
}

func (c *Client) Install(pkg Package, values map[string]string) error {
	if err := c.ensureStateDir(); err != nil {
		return fmt.Errorf("failed to ensure state directory: %w", err)
	}

	mergedValues := c.mergeValues(pkg.Values, values)

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
	installedPkg, err := c.getInstalledPackage(packageName)
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

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state map[string]InstalledPackage
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	packages := make([]InstalledPackage, 0, len(state))
	for _, pkg := range state {
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (c *Client) ensureStateDir() error {
	return os.MkdirAll(c.stateDir, 0o750)
}

func (c *Client) mergeValues(defaults, overrides map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

func (c *Client) validateParameters(params map[string]Param, values map[string]string) error {
	for name, param := range params {
		value, exists := values[name]
		if param.Required && (!exists || value == "") {
			return fmt.Errorf("required parameter '%s' is missing", name)
		}
	}
	return nil
}

func (c *Client) saveInstalledPackage(pkg InstalledPackage) error {
	stateFile := filepath.Join(c.stateDir, "installed.json")

	var state map[string]InstalledPackage
	if data, err := os.ReadFile(stateFile); err == nil {
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

func (c *Client) getInstalledPackage(name string) (InstalledPackage, error) {
	stateFile := filepath.Join(c.stateDir, "installed.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return InstalledPackage{}, err
	}

	var state map[string]InstalledPackage
	if err := json.Unmarshal(data, &state); err != nil {
		return InstalledPackage{}, err
	}

	pkg, exists := state[name]
	if !exists {
		return InstalledPackage{}, fmt.Errorf("package '%s' not found", name)
	}

	return pkg, nil
}

func (c *Client) removeInstalledPackage(name string) error {
	stateFile := filepath.Join(c.stateDir, "installed.json")
	data, err := os.ReadFile(stateFile)
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
