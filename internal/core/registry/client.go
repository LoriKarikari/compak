package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/samber/lo"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	packageYAML       = "package.yaml"
	dockerComposeYAML = "docker-compose.yaml"
	warningFmt        = "Warning: failed to remove %s: %v\n"
	httpsPrefix       = "https://"
	mediaTypePackage  = "application/vnd.compak.package.config.v1+yaml"
	mediaTypeCompose  = "application/vnd.compak.compose.v1+yaml"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Pull(ctx context.Context, reference, destDir string) error {
	repo, err := c.createRepository(reference)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	fs, err := file.New(destDir)
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}
	defer func() {
		if closeErr := fs.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file store: %v\n", closeErr)
		}
	}()

	tag := extractTag(reference)
	if err := c.copyFromRegistry(ctx, repo, tag, fs); err != nil {
		return err
	}

	if err := c.extractFiles(ctx, fs, tag, destDir); err != nil {
		return err
	}

	cleanupFiles(destDir)
	return nil
}

func (c *Client) createRepository(reference string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: getDockerCredentials(),
	}
	return repo, nil
}

func extractTag(reference string) string {
	parts := strings.Split(reference, ":")
	return lo.Ternary(len(parts) > 1, parts[len(parts)-1], "latest")
}

func (c *Client) copyFromRegistry(ctx context.Context, repo *remote.Repository, tag string, fs *file.Store) error {
	_, err := oras.Copy(ctx, repo, tag, fs, "", oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to pull package: %w", err)
	}
	return nil
}

func (c *Client) extractFiles(ctx context.Context, fs *file.Store, tag, destDir string) error {
	_, manifestContent, err := oras.FetchBytes(ctx, fs, tag, oras.DefaultFetchBytesOptions)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	var errors []error
	for _, layer := range manifest.Layers {
		if err := c.processLayer(layer, destDir); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to process %d layer(s): %v", len(errors), errors[0])
	}

	return nil
}

func (c *Client) processLayer(layer ocispec.Descriptor, destDir string) error {
	layerPath := filepath.Join(destDir, "blobs", "sha256", layer.Digest.Hex())
	if _, err := os.Stat(layerPath); os.IsNotExist(err) {
		return nil
	}

	cleanPath := filepath.Clean(layerPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(destDir)) {
		return fmt.Errorf("layer path outside destination directory")
	}

	content, err := os.ReadFile(layerPath)
	if err != nil {
		return fmt.Errorf("failed to read layer %s: %w", layerPath, err)
	}

	filename := getFilenameFromMediaType(layer.MediaType)
	if filename == "" {
		return nil
	}

	targetPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}
	fmt.Printf("Downloaded: %s\n", filename)
	return nil
}

func getFilenameFromMediaType(mediaType string) string {
	switch mediaType {
	case mediaTypePackage:
		return packageYAML
	case mediaTypeCompose:
		return dockerComposeYAML
	default:
		return ""
	}
}

func (c *Client) Push(ctx context.Context, sourceDir, reference string) error {
	cleanSourceDir := filepath.Clean(sourceDir)
	packageFile := filepath.Join(cleanSourceDir, packageYAML)
	composeFile := filepath.Join(cleanSourceDir, dockerComposeYAML)

	if !strings.HasPrefix(filepath.Clean(packageFile), cleanSourceDir) {
		return fmt.Errorf("package file path outside source directory")
	}
	if !strings.HasPrefix(filepath.Clean(composeFile), cleanSourceDir) {
		return fmt.Errorf("compose file path outside source directory")
	}

	packageData, err := os.ReadFile(packageFile)
	if err != nil {
		return fmt.Errorf("failed to read package.yaml: %w", err)
	}
	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to read docker-compose.yaml: %w", err)
	}

	parts := strings.Split(reference, ":")
	repoRef, tag := lo.Ternary(
		len(parts) > 1,
		lo.T2(strings.Join(parts[:len(parts)-1], ":"), parts[len(parts)-1]),
		lo.T2(reference, "latest"),
	).Unpack()

	repo, err := remote.NewRepository(repoRef)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: getDockerCredentials(),
	}

	packageDesc, err := oras.PushBytes(ctx, repo, mediaTypePackage, packageData)
	if err != nil {
		return fmt.Errorf("failed to push package.yaml: %w", err)
	}
	packageDesc.Annotations = map[string]string{
		ocispec.AnnotationTitle: packageYAML,
	}

	composeDesc, err := oras.PushBytes(ctx, repo, mediaTypeCompose, composeData)
	if err != nil {
		return fmt.Errorf("failed to push docker-compose.yaml: %w", err)
	}
	composeDesc.Annotations = map[string]string{
		ocispec.AnnotationTitle: dockerComposeYAML,
	}

	packOpts := oras.PackManifestOptions{
		Layers: []ocispec.Descriptor{packageDesc, composeDesc},
	}
	artifactType := "application/vnd.compak.package.v1+tar"
	manifestDesc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, artifactType, packOpts)
	if err != nil {
		return fmt.Errorf("failed to pack manifest: %w", err)
	}

	err = repo.Tag(ctx, manifestDesc, tag)
	if err != nil {
		return fmt.Errorf("failed to tag manifest: %w", err)
	}

	return nil
}

func cleanupFiles(destDir string) {
	blobsDir := filepath.Join(destDir, "blobs")
	if err := os.RemoveAll(blobsDir); err != nil {
		fmt.Fprintf(os.Stderr, warningFmt, blobsDir, err)
	}
	indexFile := filepath.Join(destDir, "index.json")
	if err := os.Remove(indexFile); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, warningFmt, indexFile, err)
	}
	layoutFile := filepath.Join(destDir, "oci-layout")
	if err := os.Remove(layoutFile); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, warningFmt, layoutFile, err)
	}
}

func IsRegistryReference(ref string) bool {
	return strings.Contains(ref, "/") && (strings.Contains(ref, ".") || strings.Contains(ref, ":"))
}

func getDockerCredentials() auth.CredentialFunc {
	return func(ctx context.Context, registry string) (auth.Credential, error) {
		if cred := getGitHubCredential(registry); cred.Username != "" {
			return cred, nil
		}
		return getDockerConfigCredential(registry)
	}
}

func getGitHubCredential(registry string) auth.Credential {
	if !strings.Contains(registry, "ghcr.io") {
		return auth.EmptyCredential
	}

	token := os.Getenv("GITHUB_TOKEN")
	username := os.Getenv("GITHUB_USER")

	if token == "" || username == "" {
		return auth.EmptyCredential
	}

	fmt.Printf("Using GITHUB_TOKEN for authentication\n")
	return auth.Credential{
		Username: username,
		Password: token,
	}
}

func getDockerConfigCredential(registry string) (auth.Credential, error) {
	configPath := getDockerConfigPath()
	if configPath == "" {
		return auth.EmptyCredential, nil
	}

	config, err := loadDockerConfig(configPath)
	if err != nil {
		return auth.EmptyCredential, nil
	}

	return findRegistryCredential(config, registry), nil
}

func getDockerConfigPath() string {
	if dockerConfig := os.Getenv("DOCKER_CONFIG"); dockerConfig != "" {
		return filepath.Join(dockerConfig, "config.json")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".docker", "config.json")
}

func loadDockerConfig(configFile string) (DockerConfig, error) {
	var config DockerConfig

	cleanPath := filepath.Clean(configFile)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dockerDir := filepath.Join(homeDir, ".docker")
		if !strings.HasPrefix(cleanPath, dockerDir) {
			dockerConfigDir := os.Getenv("DOCKER_CONFIG")
			if dockerConfigDir == "" || !strings.HasPrefix(cleanPath, filepath.Clean(dockerConfigDir)) {
				return config, fmt.Errorf("config file outside docker directory")
			}
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}

func findRegistryCredential(config DockerConfig, registry string) auth.Credential {
	registryVariants := []string{"", "/v1/", "/v2/"}
	registryKeys := lo.FlatMap([]string{registry, httpsPrefix + registry}, func(base string, _ int) []string {
		return lo.Map(registryVariants, func(suffix string, _ int) string {
			if suffix == "" && base != registry {
				return base
			}
			return base + suffix
		})
	})

	return lo.Reduce(registryKeys, func(acc auth.Credential, key string, _ int) auth.Credential {
		if acc.Username != "" {
			return acc
		}
		return decodeAuth(config.Auths[key].Auth)
	}, auth.EmptyCredential)
}

func decodeAuth(authStr string) auth.Credential {
	if authStr == "" {
		return auth.EmptyCredential
	}

	decoded, err := base64.StdEncoding.DecodeString(authStr)
	if err != nil {
		return auth.EmptyCredential
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return auth.EmptyCredential
	}

	return auth.Credential{
		Username: parts[0],
		Password: parts[1],
	}
}

type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

type DockerAuth struct {
	Auth string `json:"auth"`
}
