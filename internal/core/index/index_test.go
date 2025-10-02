package index

import (
	"context"
	"testing"
)

const (
	skipIntegrationMsg = "skipping integration test"
	searchFailedMsg    = "Search failed: %v"
)

func TestLoadPackageFromIndexRepo(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegrationMsg)
	}

	client := NewClient()
	ctx := context.Background()

	data, err := client.LoadPackageFromIndex(ctx, "immich")
	if err != nil {
		t.Fatalf("LoadPackageFromIndex failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected package data, got empty")
	}
}

func TestLoadPackageFromIndexNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegrationMsg)
	}

	client := NewClient()
	ctx := context.Background()

	_, err := client.LoadPackageFromIndex(ctx, "nonexistent-package-xyz")
	if err == nil {
		t.Error("Expected error for nonexistent package, got nil")
	}
}

func TestSearchRealRepo(t *testing.T) {
	if testing.Short() {
		t.Skip(skipIntegrationMsg)
	}

	client := NewClient()
	ctx := context.Background()

	testSearchAllPackages(ctx, t, client)
	testSearchForImmich(ctx, t, client)
	testSearchByDescription(ctx, t, client)
	testSearchWithLimit(ctx, t, client)
}

func testSearchAllPackages(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()
	results, err := client.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf(searchFailedMsg, err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one package in index")
	}
}

func testSearchForImmich(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()
	results, err := client.Search(ctx, "immich", 10)
	if err != nil {
		t.Fatalf(searchFailedMsg, err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for immich, got %d", len(results))
	}

	if len(results) > 0 {
		validateImmichResult(t, results[0])
	}
}

func validateImmichResult(t *testing.T, result SearchResult) {
	t.Helper()
	if result.Name != "immich" {
		t.Errorf("Expected name 'immich', got '%s'", result.Name)
	}
	if result.Version == "" {
		t.Error("Expected version to be set")
	}
	if result.Author == "" {
		t.Error("Expected author to be set")
	}
	if result.Homepage == "" {
		t.Error("Expected homepage to be set")
	}
	if result.Source == "" {
		t.Error("Expected source URL to be set")
	}
}

func testSearchByDescription(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()
	results, err := client.Search(ctx, "photo", 10)
	if err != nil {
		t.Fatalf(searchFailedMsg, err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result for 'photo' search")
	}

	found := false
	for _, r := range results {
		if r.Name == "immich" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find immich when searching for 'photo'")
	}
}

func testSearchWithLimit(ctx context.Context, t *testing.T, client *Client) {
	t.Helper()
	results, err := client.Search(ctx, "", 1)
	if err != nil {
		t.Fatalf(searchFailedMsg, err)
	}

	if len(results) > 1 {
		t.Errorf("Expected max 1 result due to limit, got %d", len(results))
	}
}
