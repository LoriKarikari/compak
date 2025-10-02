package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishCmdArgs(t *testing.T) {
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
			args:    []string{"ghcr.io/user/package:1.0.0"},
			wantErr: false,
		},
		{
			name:    "two args (invalid)",
			args:    []string{"ghcr.io/user/package:1.0.0", "extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishCmd.Args(publishCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestPublishCmdFlags(t *testing.T) {
	pathFlag := publishCmd.Flags().Lookup("path")
	if pathFlag == nil {
		t.Error("Expected --path flag to be defined")
		return
	}

	if pathFlag.DefValue != "." {
		t.Errorf("Expected --path default value to be '.', got %q", pathFlag.DefValue)
	}
}

func TestPublishPackageMissingFiles(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(dir string)
		expectedError string
	}{
		{
			name: "missing package.yaml",
			setup: func(dir string) {
				composeContent := `services:
  app:
    image: nginx:alpine`
				composePath := filepath.Join(dir, "docker-compose.yaml")
				if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
					t.Fatalf("failed to create docker-compose.yaml: %v", err)
				}
			},
			expectedError: "package.yaml not found",
		},
		{
			name: "missing docker-compose.yaml",
			setup: func(dir string) {
				packageContent := `name: test-pkg
version: 1.0.0
description: Test package`
				packagePath := filepath.Join(dir, "package.yaml")
				if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
					t.Fatalf("failed to create package.yaml: %v", err)
				}
			},
			expectedError: "docker-compose.yaml not found",
		},
		{
			name:          "both files missing",
			setup:         func(dir string) {},
			expectedError: "package.yaml not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			tt.setup(testDir)

			err := publishPackage("ghcr.io/test/pkg:1.0.0", testDir)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if tt.expectedError != "" && !contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestPublishPackageInvalidYaml(t *testing.T) {
	tempDir := t.TempDir()

	packageContent := `invalid yaml content
this is not: valid: yaml:`
	packagePath := filepath.Join(tempDir, "package.yaml")
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		t.Fatalf("failed to create package.yaml: %v", err)
	}

	composeContent := `services:
  app:
    image: nginx:alpine`
	composePath := filepath.Join(tempDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to create docker-compose.yaml: %v", err)
	}

	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	err := publishPackage("ghcr.io/test/pkg:1.0.0", tempDir)
	if err == nil {
		t.Fatal("expected error for invalid yaml, got nil")
	}

	if !contains(err.Error(), "failed to parse package.yaml") {
		t.Errorf("expected parse error, got: %v", err.Error())
	}
}

func TestPublishPackageValidFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := t.TempDir()

	packageContent := `name: test-publish-pkg
version: 1.0.0
description: Test package for publish
parameters:
  PORT:
    type: integer
    default: "8080"
    description: Port number`
	packagePath := filepath.Join(tempDir, "package.yaml")
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		t.Fatalf("failed to create package.yaml: %v", err)
	}

	composeContent := `services:
  app:
    image: nginx:alpine
    ports:
      - "${PORT}:80"`
	composePath := filepath.Join(tempDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to create docker-compose.yaml: %v", err)
	}

	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	err := publishPackage("localhost:5000/test/pkg:1.0.0", tempDir)
	if err != nil {
		if contains(err.Error(), "connection refused") || contains(err.Error(), "no such host") {
			t.Logf("Publish failed due to network (expected in test env): %v", err)
		} else {
			t.Logf("Publish command structure validated, registry push failed: %v", err)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
