package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/core/index"
)

func TestSearchCmd_Flags(t *testing.T) {
	if searchCmd.Flags().Lookup("limit") == nil {
		t.Error("Expected --limit flag to be defined")
	}
}

func TestSearchCmd_Args(t *testing.T) {
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
			err := searchCmd.Args(searchCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestSearchPackages_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	err := searchPackages("immich")
	if err != nil {
		t.Fatalf("searchPackages failed: %v", err)
	}
}

func TestParseSetValues(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:  "single value",
			input: []string{"PORT=8080"},
			expected: map[string]string{
				"PORT": "8080",
			},
		},
		{
			name:  "multiple values",
			input: []string{"PORT=8080", "HOST=localhost", "DB_NAME=mydb"},
			expected: map[string]string{
				"PORT":    "8080",
				"HOST":    "localhost",
				"DB_NAME": "mydb",
			},
		},
		{
			name:  "values with equals sign",
			input: []string{"CONNECTION_STRING=host=localhost;port=5432"},
			expected: map[string]string{
				"CONNECTION_STRING": "host=localhost;port=5432",
			},
		},
		{
			name:     "invalid format (no equals)",
			input:    []string{"PORT", "HOST=localhost"},
			expected: map[string]string{"HOST": "localhost"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			os.Stdout = nil
			defer func() { os.Stdout = old }()

			result := parseSetValues(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), len(result))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestDisplaySearchResult(t *testing.T) {
	tests := []struct {
		name           string
		result         index.SearchResult
		expectedOutput []string
	}{
		{
			name: "full result",
			result: index.SearchResult{
				Name:        "nginx",
				Version:     "1.0.0",
				Description: "Web server",
				Author:      "maintainer",
				Homepage:    "https://nginx.org",
			},
			expectedOutput: []string{
				"[nginx] v1.0.0",
				"Web server",
				"Author: maintainer",
				"Homepage: https://nginx.org",
			},
		},
		{
			name: "minimal result",
			result: index.SearchResult{
				Name:    "minimal",
				Version: "2.0.0",
			},
			expectedOutput: []string{
				"[minimal] v2.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			displaySearchResult(tt.result)

			if err := w.Close(); err != nil {
				t.Errorf("failed to close pipe: %v", err)
			}
			os.Stdout = old
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read from pipe: %v", err)
			}
			output := buf.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Output doesn't contain %q\nGot:\n%s", expected, output)
				}
			}
		})
	}
}

func TestSearchCmd_Execute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		wantInOutput string
	}{
		{
			name:         "search with limit",
			args:         []string{"search", "--limit", "5"},
			wantErr:      false,
			wantInOutput: "pak(s)",
		},
		{
			name:         "search specific package",
			args:         []string{"search", "immich"},
			wantErr:      false,
			wantInOutput: "Searching for paks",
		},
		{
			name:         "list all packages",
			args:         []string{"search"},
			wantErr:      false,
			wantInOutput: "Listing available paks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			cmd := &cobra.Command{Use: "compak"}
			cmd.AddCommand(searchCmd)

			searchLimit = 10

			cmd.SetArgs(tt.args)

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

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantInOutput != "" && !strings.Contains(output, tt.wantInOutput) {
				t.Errorf("Execute() output doesn't contain %q\nGot:\n%s", tt.wantInOutput, output)
			}
		})
	}
}
