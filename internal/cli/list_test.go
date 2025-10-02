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

func TestListCmd(t *testing.T) {
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

	testPkg1 := pkg.Package{
		Name:        "test-nginx",
		Version:     "1.0.0",
		Description: "Test nginx package",
	}

	testPkg2 := pkg.Package{
		Name:        "test-postgres",
		Version:     "14.0.0",
		Description: "Test postgres package",
	}

	if err := client.Install(testPkg1, nil); err != nil {
		t.Fatalf("failed to install test package 1: %v", err)
	}

	if err := client.Install(testPkg2, nil); err != nil {
		t.Fatalf("failed to install test package 2: %v", err)
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(listCmd)
	cmd.SetArgs([]string{"list"})

	err = cmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read from pipe: %v", readErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	expectedStrings := []string{
		"NAME",
		"VERSION",
		"STATUS",
		"INSTALLED",
		"test-nginx",
		"1.0.0",
		"test-postgres",
		"14.0.0",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output doesn't contain %q\nGot:\n%s", expected, output)
		}
	}
}

func TestListCmdEmpty(t *testing.T) {
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
	cmd.AddCommand(listCmd)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read from pipe: %v", readErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	if !strings.Contains(output, "No packages installed") {
		t.Errorf("expected 'No packages installed' message\nGot:\n%s", output)
	}
}

func TestListCmdArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args (valid)",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "with args (should still work as they're ignored)",
			args:    []string{"extra"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			os.Stdout = nil
			defer func() { os.Stdout = old }()

			cmd := &cobra.Command{Use: "compak"}
			cmd.AddCommand(listCmd)
			cmd.SetArgs(append([]string{"list"}, tt.args...))

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
