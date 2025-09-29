package compose

import (
	"os"
	"os/exec"
	"strings"
)

type ComposeCommand struct {
	Command string
	Args    []string
}

func DetectComposeCommand() (*ComposeCommand, error) {
	commands := []ComposeCommand{
		{Command: "docker", Args: []string{"compose"}},
		{Command: "docker-compose", Args: []string{}},
		{Command: "podman-compose", Args: []string{}},
		{Command: "podman", Args: []string{"compose"}},
	}

	for _, cmd := range commands {
		if commandExists(cmd.Command) {
			if cmd.Command == "docker" || cmd.Command == "podman" {
				if err := checkComposePlugin(cmd.Command); err != nil {
					continue
				}
			}
			return &cmd, nil
		}
	}

	return nil, ErrNoComposeFound
}

func checkComposePlugin(command string) error {
	cmd := exec.Command(command, "compose", "version")
	return cmd.Run()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (c *ComposeCommand) Execute(args ...string) error {
	return c.ExecuteIn("", args...)
}

func (c *ComposeCommand) ExecuteIn(dir string, args ...string) error {
	allArgs := make([]string, len(c.Args)+len(args))
	copy(allArgs, c.Args)
	copy(allArgs[len(c.Args):], args)

	cmd := exec.Command(c.Command, allArgs...)
	if dir != "" {
		cmd.Dir = dir
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (c *ComposeCommand) ExecuteQuiet(dir string, args ...string) (string, error) {
	allArgs := make([]string, len(c.Args)+len(args))
	copy(allArgs, c.Args)
	copy(allArgs[len(c.Args):], args)

	cmd := exec.Command(c.Command, allArgs...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (c *ComposeCommand) String() string {
	if len(c.Args) > 0 {
		return c.Command + " " + strings.Join(c.Args, " ")
	}
	return c.Command
}
