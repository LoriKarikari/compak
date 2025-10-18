package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestUpgradeCmdArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args with --all flag (valid)",
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
			err := upgradeCmd.Args(upgradeCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation: wantErr=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestUpgradeCmdFlags(t *testing.T) {
	if upgradeCmd.Flags().Lookup("version") == nil {
		t.Error("Expected --version flag to be defined")
	}

	if upgradeCmd.Flags().Lookup("all") == nil {
		t.Error("Expected --all flag to be defined")
	}
}

func TestUpgradeCmdRequiresArg(t *testing.T) {
	cmd := &cobra.Command{Use: "compak"}
	cmd.AddCommand(upgradeCmd)

	cmd.SetArgs([]string{"upgrade"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no package name provided and --all not set")
	}

	expectedMsg := "package name required (or use --all)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name           string
		installed      string
		latest         string
		shouldUpgrade  bool
		expectedReason string
	}{
		{
			name:           "same version",
			installed:      "1.0.0",
			latest:         "1.0.0",
			shouldUpgrade:  false,
			expectedReason: "up to date",
		},
		{
			name:           "newer version available",
			installed:      "1.0.0",
			latest:         "1.1.0",
			shouldUpgrade:  true,
			expectedReason: "",
		},
		{
			name:           "downgrade attempt",
			installed:      "2.0.0",
			latest:         "1.0.0",
			shouldUpgrade:  false,
			expectedReason: "would downgrade (2.0.0 → 1.0.0), use --force to downgrade",
		},
		{
			name:          "latest tag",
			installed:     "1.0.0",
			latest:        "latest",
			shouldUpgrade: true,
		},
		{
			name:          "invalid installed version",
			installed:     "invalid",
			latest:        "1.0.0",
			shouldUpgrade: true,
		},
		{
			name:          "invalid latest version",
			installed:     "1.0.0",
			latest:        "invalid",
			shouldUpgrade: true,
		},
		{
			name:           "patch version upgrade",
			installed:      "1.0.0",
			latest:         "1.0.1",
			shouldUpgrade:  true,
			expectedReason: "",
		},
		{
			name:           "major version upgrade",
			installed:      "1.0.0",
			latest:         "2.0.0",
			shouldUpgrade:  true,
			expectedReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldUpgrade, reason := compareVersions(tt.installed, tt.latest)

			if shouldUpgrade != tt.shouldUpgrade {
				t.Errorf("compareVersions(%q, %q) shouldUpgrade = %v, want %v",
					tt.installed, tt.latest, shouldUpgrade, tt.shouldUpgrade)
			}

			if tt.expectedReason != "" && reason != tt.expectedReason {
				t.Errorf("compareVersions(%q, %q) reason = %q, want %q",
					tt.installed, tt.latest, reason, tt.expectedReason)
			}
		})
	}
}

func TestFetchLatestPackage_InvalidVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name          string
		packageName   string
		targetVersion string
		wantErr       bool
	}{
		{
			name:          "invalid semantic version",
			packageName:   "immich",
			targetVersion: "not-a-version",
			wantErr:       true,
		},
		{
			name:          "latest is valid",
			packageName:   "immich",
			targetVersion: "latest",
			wantErr:       false,
		},
		{
			name:          "empty version is valid",
			packageName:   "immich",
			targetVersion: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetchLatestPackage(context.Background(), tt.packageName, tt.targetVersion)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil && tt.targetVersion != "" {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
