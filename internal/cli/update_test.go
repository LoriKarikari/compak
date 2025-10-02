package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUpdateCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(updateCmd)
	cmd.SetArgs([]string{"update"})

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
		t.Fatalf("update command failed: %v", err)
	}

	expectedStrings := []string{
		"Updating compak pak index",
		"Pak index updated successfully",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output doesn't contain %q\nGot:\n%s", expected, output)
		}
	}
}

func TestUpdateCmdArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args (valid)",
			args:    []string{"update"},
			wantErr: false,
		},
		{
			name:    "with args (should still work as they're ignored)",
			args:    []string{"update", "extra"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			os.Stdout = nil
			defer func() { os.Stdout = old }()

			cmd := &cobra.Command{Use: "compak"}
			cmd.AddCommand(updateCmd)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	err := updateIndex()
	if err != nil {
		t.Fatalf("updateIndex failed: %v", err)
	}
}
