package compose

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	dockercli "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
)

type Client struct {
	service api.Service
}

func NewClient() (*Client, error) {
	dockerCli, err := dockercli.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	if err := dockerCli.Initialize(flags.NewClientOptions()); err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	service := compose.NewComposeService(dockerCli)

	return &Client{
		service: service,
	}, nil
}

func (c *Client) LoadProject(projectDir, projectName string) (*types.Project, error) {
	composeFile := filepath.Join(projectDir, "docker-compose.yaml")
	envFile := filepath.Join(projectDir, ".env")

	if err := loadAndExportEnv(projectDir, ".env"); err != nil {
		return nil, fmt.Errorf("failed to load .env: %w", err)
	}

	options, err := cli.NewProjectOptions(
		[]string{composeFile},
		cli.WithName(projectName),
		cli.WithWorkingDirectory(projectDir),
		cli.WithEnvFiles(envFile),
		cli.WithOsEnv,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	project, err := cli.ProjectFromOptions(context.Background(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	project.Name = projectName

	for i, service := range project.Services {
		if service.Labels == nil {
			service.Labels = make(types.Labels)
		}
		service.Labels[api.ProjectLabel] = projectName
		service.Labels[api.ServiceLabel] = service.Name
		service.Labels[api.VersionLabel] = api.ComposeVersion
		service.Labels[api.WorkingDirLabel] = project.WorkingDir
		service.Labels[api.ConfigFilesLabel] = strings.Join(project.ComposeFiles, ",")
		service.Labels[api.OneoffLabel] = "False"
		project.Services[i] = service
	}

	return project, nil
}

func (c *Client) Up(ctx context.Context, project *types.Project, detach bool, consumer api.LogConsumer) error {
	err := c.service.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			Recreate:             api.RecreateDiverged,
			RecreateDependencies: api.RecreateDiverged,
			RemoveOrphans:        true,
		},
		Start: api.StartOptions{},
	})
	if err != nil {
		return err
	}

	if !detach {
		if consumer == nil {
			return fmt.Errorf("log consumer required when not running detached")
		}
		return c.service.Logs(ctx, project.Name, consumer, api.LogOptions{
			Follow: true,
		})
	}

	return nil
}

func (c *Client) Down(ctx context.Context, projectName string) error {
	return c.service.Down(ctx, projectName, api.DownOptions{
		RemoveOrphans: true,
	})
}

func (c *Client) PS(ctx context.Context, projectName string) ([]api.ContainerSummary, error) {
	return c.service.Ps(ctx, projectName, api.PsOptions{
		All: true,
	})
}

func (c *Client) Pull(ctx context.Context, project *types.Project) error {
	return c.service.Pull(ctx, project, api.PullOptions{})
}

func (c *Client) Logs(ctx context.Context, projectName string, consumer api.LogConsumer, follow bool) error {
	return c.service.Logs(ctx, projectName, consumer, api.LogOptions{
		Follow: follow,
	})
}

func loadAndExportEnv(rootDir, filename string) (err error) {
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("failed to open root: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	f, err := root.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return parseAndSetEnv(f)
}

func parseAndSetEnv(f *os.File) error {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		if key != "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}
	return scanner.Err()
}
