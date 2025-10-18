package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Package struct {
	Name        string         `yaml:"name"`
	Version     string         `yaml:"version"`
	Description string         `yaml:"description"`
	Author      string         `yaml:"author"`
	Homepage    string         `yaml:"homepage"`
	Repository  string         `yaml:"repository"`
	Source      string         `yaml:"source"`
	Parameters  map[string]any `yaml:"parameters"`
}

type UpdateResult struct {
	PackageName     string
	CurrentVersion  string
	LatestVersion   string
	ComposeChanged  bool
	ComposeChecksum string
	UpdatedFile     string
	Error           error
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func main() {
	paksDir := flag.String("paks-dir", "paks", "Directory containing package YAML files")
	cacheDir := flag.String("cache-dir", ".cache/compose-checksums", "Directory to store compose checksums")
	dryRun := flag.Bool("dry-run", false, "Check for updates without creating PRs")
	flag.Parse()

	if err := run(*paksDir, *cacheDir, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(paksDir, cacheDir string, dryRun bool) error {
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	results, err := checkAllPackages(paksDir, cacheDir)
	if err != nil {
		return fmt.Errorf("error checking packages: %w", err)
	}

	updatesFound := processResults(results, dryRun)
	printSummary(updatesFound, dryRun)
	return nil
}

func processResults(results []UpdateResult, dryRun bool) bool {
	updatesFound := false
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("✗ %s: %v\n", result.PackageName, result.Error)
			continue
		}

		versionChanged := result.LatestVersion != "" && result.LatestVersion != result.CurrentVersion
		composeChanged := result.ComposeChanged

		switch {
		case versionChanged && composeChanged:
			updatesFound = true
			fmt.Printf("✓ %s: %s → %s (compose file also changed)\n", result.PackageName, result.CurrentVersion, result.LatestVersion)
		case versionChanged:
			updatesFound = true
			fmt.Printf("✓ %s: %s → %s\n", result.PackageName, result.CurrentVersion, result.LatestVersion)
		case composeChanged:
			updatesFound = true
			fmt.Printf("✓ %s: compose file changed (version %s)\n", result.PackageName, result.CurrentVersion)
		default:
			fmt.Printf("  %s: up to date (%s)\n", result.PackageName, result.CurrentVersion)
			continue
		}

		if !dryRun {
			if err := createPR(result); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create PR for %s: %v\n", result.PackageName, err)
			}
		}
	}
	return updatesFound
}

func printSummary(updatesFound, dryRun bool) {
	switch {
	case !updatesFound:
		fmt.Println("\n✓ All packages are up to date.")
	case dryRun:
		fmt.Println("\n✓ Updates detected (dry-run mode, no PRs created)")
	default:
		fmt.Println("\n✓ PRs created for updated packages")
	}
}

func checkAllPackages(paksDir, cacheDir string) ([]UpdateResult, error) {
	entries, err := os.ReadDir(paksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read paks directory: %w", err)
	}

	paksRoot, err := os.OpenRoot(paksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open paks root: %w", err)
	}
	defer func() {
		if closeErr := paksRoot.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close paks root: %v\n", closeErr)
		}
	}()

	cacheRoot, err := os.OpenRoot(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache root: %w", err)
	}
	defer func() {
		if closeErr := cacheRoot.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close cache root: %v\n", closeErr)
		}
	}()

	results := make([]UpdateResult, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		pakName := strings.TrimSuffix(entry.Name(), ".yaml")
		if strings.Contains(pakName, "@") {
			continue
		}

		result := checkPackage(entry.Name(), pakName, paksRoot, cacheRoot, paksDir)
		results = append(results, result)
	}

	return results, nil
}

func checkPackage(pakFileName, pakName string, paksRoot, cacheRoot *os.Root, paksDir string) UpdateResult {
	data, err := paksRoot.ReadFile(pakFileName)
	if err != nil {
		return UpdateResult{
			PackageName: pakName,
			Error:       fmt.Errorf("failed to read package file: %w", err),
		}
	}

	var pkg Package
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return UpdateResult{
			PackageName: pakName,
			Error:       fmt.Errorf("failed to parse package YAML: %w", err),
		}
	}

	if pkg.Repository == "" {
		return UpdateResult{
			PackageName:    pakName,
			CurrentVersion: pkg.Version,
			Error:          fmt.Errorf("no repository field"),
		}
	}

	latestVersion, err := getLatestGitHubRelease(pkg.Repository)
	if err != nil {
		return UpdateResult{
			PackageName:    pakName,
			CurrentVersion: pkg.Version,
			Error:          fmt.Errorf("failed to get latest release: %w", err),
		}
	}

	composeChanged := false
	composeChecksum := ""
	if pkg.Source != "" {
		composeContent, err := fetchURL(pkg.Source)
		if err != nil {
			return UpdateResult{
				PackageName:    pakName,
				CurrentVersion: pkg.Version,
				LatestVersion:  latestVersion,
				Error:          fmt.Errorf("failed to fetch compose file: %w", err),
			}
		}

		composeChecksum = checksumBytes(composeContent)
		cacheFileName := pakName + ".sha256"

		if cachedChecksum, err := cacheRoot.ReadFile(cacheFileName); err != nil {
			if writeErr := cacheRoot.WriteFile(cacheFileName, []byte(composeChecksum), 0o600); writeErr != nil {
				return UpdateResult{
					PackageName:    pakName,
					CurrentVersion: pkg.Version,
					LatestVersion:  latestVersion,
					Error:          fmt.Errorf("failed to write cache: %w", writeErr),
				}
			}
		} else if string(cachedChecksum) != composeChecksum {
			composeChanged = true
			if writeErr := cacheRoot.WriteFile(cacheFileName, []byte(composeChecksum), 0o600); writeErr != nil {
				return UpdateResult{
					PackageName:    pakName,
					CurrentVersion: pkg.Version,
					LatestVersion:  latestVersion,
					Error:          fmt.Errorf("failed to update cache: %w", writeErr),
				}
			}
		}
	}

	updatedFile := ""
	if latestVersion != pkg.Version || composeChanged {
		updatedFile = filepath.Join(paksDir, pakFileName)
	}

	return UpdateResult{
		PackageName:     pakName,
		CurrentVersion:  pkg.Version,
		LatestVersion:   latestVersion,
		ComposeChanged:  composeChanged,
		ComposeChecksum: composeChecksum,
		UpdatedFile:     updatedFile,
	}
}

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func checksumBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func getLatestGitHubRelease(repoURL string) (string, error) {
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) < 3 {
		return "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
	}

	owner := matches[1]
	repo := matches[2]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func createPR(result UpdateResult) error {
	branchName := fmt.Sprintf("upstream-sync/%s-%s", result.PackageName, time.Now().Format("20060102"))

	if err := runCommand("git", "config", "user.name", "github-actions[bot]"); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}
	if err := runCommand("git", "config", "user.email", "github-actions[bot]@users.noreply.github.com"); err != nil {
		return fmt.Errorf("failed to configure git email: %w", err)
	}

	if err := runCommand("git", "checkout", "-b", branchName); err != nil {
		if err := runCommand("git", "checkout", branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	if err := updatePackageVersion(result.UpdatedFile, result.LatestVersion); err != nil {
		return fmt.Errorf("failed to update package file: %w", err)
	}

	if err := runCommand("git", "add", result.UpdatedFile); err != nil {
		return fmt.Errorf("failed to add package file: %w", err)
	}

	commitMsg := fmt.Sprintf("chore: bump %s to %s", result.PackageName, result.LatestVersion)
	commitBody := fmt.Sprintf("Update from %s to %s", result.CurrentVersion, result.LatestVersion)

	if err := runCommand("git", "commit", "-m", commitMsg, "-m", commitBody); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err := runCommand("git", "push", "origin", branchName, "--force"); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	prTitle := fmt.Sprintf("chore: bump %s to %s", result.PackageName, result.LatestVersion)
	prBody := fmt.Sprintf("Updates %s from %s to %s. This is an automated PR created by the upstream sync workflow.",
		result.PackageName, result.CurrentVersion, result.LatestVersion)

	if err := runCommand("gh", "pr", "create",
		"--title", prTitle,
		"--body", prBody,
		"--label", "upstream-sync",
		"--base", "main"); err != nil {
		fmt.Printf("Note: PR creation returned error (may already exist): %v\n", err)
	}

	if err := runCommand("git", "checkout", "main"); err != nil {
		return fmt.Errorf("failed to checkout main: %w", err)
	}

	return nil
}

func updatePackageVersion(filePath, newVersion string) error {
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("failed to open root directory: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close root: %v\n", closeErr)
		}
	}()

	data, err := root.ReadFile(fileName)
	if err != nil {
		return err
	}

	var pkg Package
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return err
	}

	pkg.Version = newVersion

	updated, err := yaml.Marshal(&pkg)
	if err != nil {
		return err
	}

	return root.WriteFile(fileName, updated, 0o600)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
