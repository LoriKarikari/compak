package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: http.DefaultClient,
	}
}

func (c *Client) Pull(ctx context.Context, reference, destDir string) error {
	return fmt.Errorf("OCI registry pull not yet implemented - use local packages with --path flag")
}

func IsRegistryReference(ref string) bool {
	return strings.Contains(ref, "/") && (strings.Contains(ref, ".") || strings.Contains(ref, ":"))
}
