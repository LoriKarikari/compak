package cli

import (
	"os"
	"path/filepath"
	"testing"

	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

func TestInstallCmdArgs(t *testing.T) {
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
			err := installCmd.Args(installCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestInstallCmdFlags(t *testing.T) {
	flags := []string{"version", "path", "set"}
	for _, flag := range flags {
		if installCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag to be defined", flag)
		}
	}
}

func TestDisplayPackageInfo(t *testing.T) {
	tests := []struct {
		name string
		pkg  *pkg.Package
	}{
		{
			name: "package with parameters",
			pkg: &pkg.Package{
				Name:        "test-pkg",
				Version:     "1.0.0",
				Description: "Test package",
				Parameters: map[string]pkg.Param{
					"PORT": {
						Type:        "integer",
						Default:     "8080",
						Description: "Port number",
						Required:    false,
					},
					"DB_PASSWORD": {
						Type:        "string",
						Default:     "",
						Description: "Database password",
						Required:    true,
					},
				},
				Values: map[string]string{
					"PORT": "9090",
				},
			},
		},
		{
			name: "package without parameters",
			pkg: &pkg.Package{
				Name:        "simple-pkg",
				Version:     "2.0.0",
				Description: "Simple package",
				Parameters:  make(map[string]pkg.Param),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			os.Stdout = nil
			defer func() { os.Stdout = old }()

			displayPackageInfo(tt.pkg)
		})
	}
}

func TestLoadPackageFromLocalPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := t.TempDir()

	packageContent := `name: test-local
version: 1.0.0
description: Test local package
parameters:
  PORT:
    type: integer
    default: "8080"
    description: Port number
`
	packagePath := filepath.Join(tempDir, "package.yaml")
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		t.Fatalf("failed to create package.yaml: %v", err)
	}

	composeContent := `services:
  app:
    image: nginx:alpine
    ports:
      - "${PORT}:80"
`
	composePath := filepath.Join(tempDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to create docker-compose.yaml: %v", err)
	}

	manager := pkg.NewManager(nil, nil, "")

	packageToInstall, err := manager.LoadPackageFromDir(tempDir)
	if err != nil {
		t.Fatalf("LoadPackageFromDir failed: %v", err)
	}

	if packageToInstall == nil {
		t.Fatal("expected package, got nil")
	}

	if packageToInstall.Name != "test-local" {
		t.Errorf("expected name 'test-local', got '%s'", packageToInstall.Name)
	}

	if len(packageToInstall.Parameters) == 0 {
		t.Error("expected parameters to be loaded")
	}
}

