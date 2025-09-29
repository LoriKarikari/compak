package pkg

import (
	"testing"
)

type mockComposeCommand struct {
	executed [][]string
}

func (m *mockComposeCommand) Execute(args ...string) error {
	m.executed = append(m.executed, args)
	return nil
}

func (m *mockComposeCommand) String() string {
	return "mock-compose"
}

func TestClient_Install(t *testing.T) {
	tempDir := t.TempDir()
	mockCmd := &mockComposeCommand{}
	client := NewClient(mockCmd, tempDir)

	pkg := Package{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Parameters: map[string]Param{
			"port": {
				Description: "Port number",
				Type:        "string",
				Default:     "8080",
				Required:    false,
			},
		},
		Values: map[string]string{
			"port": "3000",
		},
	}

	values := map[string]string{
		"port": "9000",
	}

	err := client.Install(pkg, values)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	packages, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	installed := packages[0]
	if installed.Package.Name != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", installed.Package.Name)
	}

	if installed.Values["port"] != "9000" {
		t.Errorf("Expected port value '9000', got '%s'", installed.Values["port"])
	}
}

func TestClient_Uninstall(t *testing.T) {
	tempDir := t.TempDir()
	mockCmd := &mockComposeCommand{}
	client := NewClient(mockCmd, tempDir)

	pkg := Package{
		Name:    "test-package",
		Version: "1.0.0",
	}

	err := client.Install(pkg, nil)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	err = client.Uninstall("test-package")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	packages, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(packages) != 0 {
		t.Fatalf("Expected 0 packages after uninstall, got %d", len(packages))
	}
}

func TestClient_List_Empty(t *testing.T) {
	tempDir := t.TempDir()
	mockCmd := &mockComposeCommand{}
	client := NewClient(mockCmd, tempDir)

	packages, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(packages) != 0 {
		t.Fatalf("Expected 0 packages, got %d", len(packages))
	}
}

func TestClient_ValidateParameters(t *testing.T) {
	tempDir := t.TempDir()
	mockCmd := &mockComposeCommand{}
	client := NewClient(mockCmd, tempDir)

	params := map[string]Param{
		"required_param": {
			Required: true,
		},
		"optional_param": {
			Required: false,
		},
	}

	tests := []struct {
		name      string
		values    map[string]string
		expectErr bool
	}{
		{
			name: "valid with required param",
			values: map[string]string{
				"required_param": "value",
			},
			expectErr: false,
		},
		{
			name:      "missing required param",
			values:    map[string]string{},
			expectErr: true,
		},
		{
			name: "empty required param",
			values: map[string]string{
				"required_param": "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateParameters(params, tt.values)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClient_MergeValues(t *testing.T) {
	tempDir := t.TempDir()
	mockCmd := &mockComposeCommand{}
	client := NewClient(mockCmd, tempDir)

	pkg := Package{
		Parameters: map[string]Param{
			"port":    {Default: "8080"},
			"host":    {Default: "localhost"},
			"enabled": {Default: "true"},
		},
	}

	overrides := map[string]string{
		"port": "9000",
		"env":  "production",
	}

	result := client.mergeValues(pkg, overrides)

	expected := map[string]string{
		"port":    "9000",
		"host":    "localhost",
		"enabled": "true",
		"env":     "production",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}
