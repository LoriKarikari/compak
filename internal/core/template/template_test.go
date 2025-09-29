package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngine_WriteEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		expected string
	}{
		{
			name: "simple values",
			values: map[string]string{
				"PORT":     "8080",
				"HOST":     "localhost",
				"DATABASE": "mydb",
			},
			expected: "DATABASE=mydb\nHOST=localhost\nPORT=8080\n",
		},
		{
			name: "values with quotes",
			values: map[string]string{
				"MESSAGE": "Hello \"World\"",
			},
			expected: "MESSAGE=\"Hello \\\"World\\\"\"\n",
		},
		{
			name:     "empty values",
			values:   map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			engine := NewEngine(tt.values)

			err := engine.WriteEnvFile(tempDir)
			if err != nil {
				t.Fatalf("WriteEnvFile failed: %v", err)
			}

			content, err := os.ReadFile(filepath.Join(tempDir, ".env"))
			if err != nil {
				t.Fatalf("Failed to read .env file: %v", err)
			}

			if string(content) != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, string(content))
			}
		})
	}
}

func TestEngine_SetEnvironment(t *testing.T) {
	originalValue := os.Getenv("TEST_VAR")
	defer func() {
		if originalValue != "" {
			if err := os.Setenv("TEST_VAR", originalValue); err != nil {
				t.Errorf("failed to restore env var: %v", err)
			}
		} else {
			if err := os.Unsetenv("TEST_VAR"); err != nil {
				t.Errorf("failed to unset env var: %v", err)
			}
		}
	}()

	if err := os.Setenv("TEST_VAR", "original"); err != nil {
		t.Fatalf("failed to set test env var: %v", err)
	}

	values := map[string]string{
		"TEST_VAR":    "modified",
		"NEW_VAR":     "new_value",
		"ANOTHER_VAR": "another_value",
	}

	engine := NewEngine(values)
	cleanup := engine.SetEnvironment()

	if os.Getenv("TEST_VAR") != "modified" {
		t.Errorf("Expected TEST_VAR to be 'modified', got '%s'", os.Getenv("TEST_VAR"))
	}

	if os.Getenv("NEW_VAR") != "new_value" {
		t.Errorf("Expected NEW_VAR to be 'new_value', got '%s'", os.Getenv("NEW_VAR"))
	}

	cleanup()

	if os.Getenv("TEST_VAR") != "original" {
		t.Errorf("Expected TEST_VAR to be restored to 'original', got '%s'", os.Getenv("TEST_VAR"))
	}

	if os.Getenv("NEW_VAR") != "" {
		t.Errorf("Expected NEW_VAR to be unset, got '%s'", os.Getenv("NEW_VAR"))
	}
}

func TestEngine_WriteEnvFile_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	values := map[string]string{
		"PASSWORD": "p@$$w0rd!",
		"PATH":     "/usr/bin:/usr/local/bin",
		"EMPTY":    "",
		"SPACES":   "hello world",
	}

	engine := NewEngine(values)
	err := engine.WriteEnvFile(tempDir)
	if err != nil {
		t.Fatalf("WriteEnvFile failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, ".env"))
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "PASSWORD=p@$$w0rd!") {
		t.Error("Password with special characters not written correctly")
	}

	if !strings.Contains(contentStr, "SPACES=hello world") {
		t.Error("Value with spaces not written correctly")
	}

	if !strings.Contains(contentStr, "EMPTY=") {
		t.Error("Empty value not written correctly")
	}
}
