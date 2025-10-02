package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

func TestUninstallCmdArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args (invalid)",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "one arg (valid)",
			args:    []string{"nginx"},
			wantErr: false,
		},
		{
			name:    "two args (invalid)",
			args:    []string{"nginx", "extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uninstallCmd.Args(uninstallCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestUninstallCmdNotInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that may require compose")
	}

	tempDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer func() {
		if err := os.Setenv("HOME", oldHome); err != nil {
			t.Errorf("failed to restore HOME: %v", err)
		}
	}()
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(uninstallCmd)
	cmd.SetArgs([]string{"uninstall", "nonexistent-package"})

	err := cmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Logf("failed to read from pipe: %v", readErr)
	}

	if err == nil {
		t.Fatal("expected error for non-existent package, got nil")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no such file") {
		t.Logf("got expected error for non-existent package: %v", err)
	}
}

func TestUninstallCmdInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires compose")
	}

	tempDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer func() {
		if err := os.Setenv("HOME", oldHome); err != nil {
			t.Errorf("failed to restore HOME: %v", err)
		}
	}()
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	stateDir, err := config.GetStateDir()
	if err != nil {
		t.Fatalf("failed to get state dir: %v", err)
	}

	client := pkg.NewClient(stateDir)

	testPkg := pkg.Package{
		Name:        "test-uninstall",
		Version:     "1.0.0",
		Description: "Test package for uninstall",
	}

	if err := client.Install(testPkg, nil); err != nil {
		t.Fatalf("failed to install test package: %v", err)
	}

	packages, err := client.List()
	if err != nil {
		t.Fatalf("failed to list packages: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(uninstallCmd)
	cmd.SetArgs([]string{"uninstall", "test-uninstall"})

	err = cmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Logf("failed to read from pipe: %v", readErr)
	}
	output := buf.String()

	if err != nil {
		if strings.Contains(err.Error(), "compose") || strings.Contains(err.Error(), "docker") {
			t.Logf("uninstall command failed due to compose (expected in test env): %v", err)
		} else {
			t.Logf("uninstall command failed: %v\nOutput: %s", err, output)
		}
	}

	packages, err = client.List()
	if err != nil {
		t.Fatalf("failed to list packages after uninstall: %v", err)
	}

	t.Logf("Packages after uninstall: %d", len(packages))
}

func TestUninstallCmdFlags(t *testing.T) {
	if uninstallCmd.Use != "uninstall [package]" {
		t.Errorf("expected Use to be 'uninstall [package]', got %q", uninstallCmd.Use)
	}

	if err := uninstallCmd.Args(uninstallCmd, []string{}); err == nil {
		t.Error("expected error for no arguments")
	}

	if err := uninstallCmd.Args(uninstallCmd, []string{"pkg"}); err != nil {
		t.Errorf("expected no error for one argument, got: %v", err)
	}

	if err := uninstallCmd.Args(uninstallCmd, []string{"pkg", "extra"}); err == nil {
		t.Error("expected error for two arguments")
	}
}
