package registry

import (
	"context"
	"fmt"
	"os"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Pull(ctx context.Context, reference, destDir string) error {
	repo, err := remote.NewRepository(reference)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
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
			fmt.Printf("Warning: failed to close file store: %v\n", closeErr)
		}
	}()

	tag := "latest"
	if parts := strings.Split(reference, ":"); len(parts) > 1 {
		tag = parts[len(parts)-1]
	}

	_, err = oras.Copy(ctx, repo, tag, fs, "", oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to pull package: %w", err)
	}

	return nil
}

func IsRegistryReference(ref string) bool {
	return strings.Contains(ref, "/") && (strings.Contains(ref, ".") || strings.Contains(ref, ":"))
}
