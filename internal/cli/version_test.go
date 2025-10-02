package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCmdArgs(t *testing.T) {
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
			name:    "with args",
			args:    []string{"extra"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			os.Stdout = nil
			defer func() { os.Stdout = old }()

			cmd := &cobra.Command{Use: "compak"}
			cmd.AddCommand(versionCmd)
			cmd.SetArgs(append([]string{"version"}, tt.args...))

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersionCmdOutput(t *testing.T) {
	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(versionCmd)
	cmd.SetArgs([]string{"version"})

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
		t.Fatalf("version command failed: %v", err)
	}

	if !strings.Contains(output, "compak") {
		t.Errorf("expected output to contain 'compak', got: %s", output)
	}

	if !strings.Contains(output, version) {
		t.Errorf("expected output to contain version '%s', got: %s", version, output)
	}

	if !strings.Contains(output, commit) {
		t.Errorf("expected output to contain commit '%s', got: %s", commit, output)
	}
}

func TestVersionVariables(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}

	if commit == "" {
		t.Error("commit variable should not be empty")
	}
}
