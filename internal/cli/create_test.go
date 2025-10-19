package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `services:
  web:
    image: nginx:${NGINX_VERSION:-latest}
    ports:
      - "${WEB_PORT:-8080}:80"
    environment:
      - API_KEY=${API_KEY}
      - DB_PASSWORD=${DB_PASSWORD}
`

	composePath := filepath.Join(tmpDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write test compose file: %v", err)
	}

	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(createCmd)

	outputPath := filepath.Join(tmpDir, "package.yaml")
	cmd.SetArgs([]string{
		"create",
		tmpDir,
		"--interactive=false",
		"--name=test-pkg",
		"--description=Test package",
		"--output=" + outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Create command failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("package.yaml was not created")
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated package.yaml: %v", err)
	}

	contentStr := string(content)

	expectedStrings := []string{
		"name: test-pkg",
		"version: 1.0.0",
		"description: \"\"",
		"parameters:",
		"API_KEY:",
		"DB_PASSWORD:",
		"NGINX_VERSION:",
		"WEB_PORT:",
		"type: string",
		"type: number",
	}

	for _, expected := range expectedStrings {
		if !contains(contentStr, expected) {
			t.Errorf("Generated package.yaml missing expected content: %q", expected)
		}
	}
}

func TestFindComposeFile(t *testing.T) {
	t.Run("finds docker-compose.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFile(t, tmpDir, "docker-compose.yaml")
		result, err := findComposeFile(tmpDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filepath.Base(result) != "docker-compose.yaml" {
			t.Errorf("Expected docker-compose.yaml, got %q", filepath.Base(result))
		}
	})

	t.Run("finds docker-compose.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFile(t, tmpDir, "docker-compose.yml")
		result, err := findComposeFile(tmpDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filepath.Base(result) != "docker-compose.yml" {
			t.Errorf("Expected docker-compose.yml, got %q", filepath.Base(result))
		}
	})

	t.Run("prefers .yaml over .yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFile(t, tmpDir, "docker-compose.yaml")
		createTestFile(t, tmpDir, "docker-compose.yml")
		result, err := findComposeFile(tmpDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filepath.Base(result) != "docker-compose.yaml" {
			t.Errorf("Expected docker-compose.yaml, got %q", filepath.Base(result))
		}
	})

	t.Run("errors when no compose file", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFile(t, tmpDir, "random.txt")
		_, err := findComposeFile(tmpDir)
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func createTestFile(t *testing.T, dir, filename string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestIsValidPackageName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "mypackage", true},
		{"valid with hyphens", "my-package", true},
		{"valid with numbers", "package123", true},
		{"valid complex", "my-package-v2", true},
		{"invalid uppercase", "MyPackage", false},
		{"invalid underscore", "my_package", false},
		{"invalid starts with hyphen", "-package", false},
		{"invalid ends with hyphen", "package-", false},
		{"invalid special chars", "my@package", false},
		{"invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPackageName(tt.input)
			if got != tt.want {
				t.Errorf("isValidPackageName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
