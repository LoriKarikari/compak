package compose

import (
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
	}

	for _, cmd := range commands {
		if commandExists(cmd.Command) {
			if cmd.Command == "docker" {
				if err := checkDockerComposePlugin(); err != nil {
					continue
				}
			}
			return &cmd, nil
		}
	}

	return nil, ErrNoComposeFound
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func checkDockerComposePlugin() error {
	cmd := exec.Command("docker", "compose", "version")
	return cmd.Run()
}

func (c *ComposeCommand) Execute(args ...string) error {
	allArgs := make([]string, len(c.Args)+len(args))
	copy(allArgs, c.Args)
	copy(allArgs[len(c.Args):], args)
	cmd := exec.Command(c.Command, allArgs...)
	return cmd.Run()
}

func (c *ComposeCommand) String() string {
	if len(c.Args) > 0 {
		return c.Command + " " + strings.Join(c.Args, " ")
	}
	return c.Command
}
