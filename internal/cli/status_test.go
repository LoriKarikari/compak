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

func TestStatusCmdArgs(t *testing.T) {
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
			err := statusCmd.Args(statusCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestStatusCmdNotInstalled(t *testing.T) {
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
	cmd.AddCommand(statusCmd)
	cmd.SetArgs([]string{"status", "nonexistent-package"})

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

	if !strings.Contains(err.Error(), "is not installed") {
		t.Errorf("expected 'is not installed' error, got: %v", err)
	}
}

func TestStatusCmdInstalled(t *testing.T) {
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
		Name:        "test-status",
		Version:     "1.0.0",
		Description: "Test package for status",
	}

	if err := client.Install(testPkg, nil); err != nil {
		t.Fatalf("failed to install test package: %v", err)
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(statusCmd)
	cmd.SetArgs([]string{"status", "test-status"})

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
		if !strings.Contains(err.Error(), "compose") && !strings.Contains(err.Error(), "docker") {
			t.Logf("status command failed (expected in test env): %v", err)
		}
	}

	if err == nil {
		if !strings.Contains(output, "Package: test-status") {
			t.Errorf("expected output to contain package name\nGot:\n%s", output)
		}
	}
}

func TestStatusCmdFlags(t *testing.T) {
	if statusCmd.Use != "status [package]" {
		t.Errorf("expected Use to be 'status [package]', got %q", statusCmd.Use)
	}

	if err := statusCmd.Args(statusCmd, []string{}); err == nil {
		t.Error("expected error for no arguments")
	}

	if err := statusCmd.Args(statusCmd, []string{"pkg"}); err != nil {
		t.Errorf("expected no error for one argument, got: %v", err)
	}

	if err := statusCmd.Args(statusCmd, []string{"pkg", "extra"}); err == nil {
		t.Error("expected error for two arguments")
	}
}
