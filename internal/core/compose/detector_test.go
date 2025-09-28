package compose

import (
	"testing"
)

func TestDetectComposeCommand(t *testing.T) {
	cmd, err := DetectComposeCommand()
	if err != nil {
		t.Skipf("No compose command found on system: %v", err)
	}

	if cmd == nil {
		t.Fatal("Expected compose command, got nil")
	}

	if cmd.Command == "" {
		t.Fatal("Expected non-empty command")
	}

	t.Logf("Detected compose command: %s", cmd.String())
}

func TestCommandExists(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "existing command",
			command:  "echo",
			expected: true,
		},
		{
			name:     "non-existing command",
			command:  "nonexistentcommand12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := commandExists(tt.command)
			if result != tt.expected {
				t.Errorf("commandExists(%s) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestComposeCommandString(t *testing.T) {
	tests := []struct {
		name     string
		cmd      ComposeCommand
		expected string
	}{
		{
			name:     "docker compose",
			cmd:      ComposeCommand{Command: "docker", Args: []string{"compose"}},
			expected: "docker compose",
		},
		{
			name:     "docker-compose",
			cmd:      ComposeCommand{Command: "docker-compose", Args: []string{}},
			expected: "docker-compose",
		},
		{
			name:     "podman-compose",
			cmd:      ComposeCommand{Command: "podman-compose", Args: []string{}},
			expected: "podman-compose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.String()
			if result != tt.expected {
				t.Errorf("ComposeCommand.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}
