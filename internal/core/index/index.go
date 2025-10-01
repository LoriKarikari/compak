package index

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

const (
	defaultRepoURL    = "https://github.com/LoriKarikari/compak.git"
	defaultPaksSubdir = "paks"
)

type Index struct {
	Paks    map[string]PakMetadata `yaml:"paks"`
	Updated time.Time              `yaml:"updated"`
}

type PakMetadata struct {
	Name        string       `yaml:"name" validate:"required,alphanum|contains=-|contains=_"`
	Description string       `yaml:"description" validate:"required,max=500"`
	Maintainer  string       `yaml:"maintainer" validate:"required"`
	Source      string       `yaml:"source" validate:"required,url"`
	Homepage    string       `yaml:"homepage" validate:"omitempty,url"`
	Versions    []PakVersion `yaml:"versions" validate:"dive"`
	Tags        []string     `yaml:"tags" validate:"dive,alphanum"`
	Verified    bool         `yaml:"verified"`
}

type PakVersion struct {
	Version        string    `yaml:"version" validate:"required"`
	Digest         string    `yaml:"digest" validate:"omitempty"`
	Created        time.Time `yaml:"created" validate:"required"`
	ComposeVersion string    `yaml:"compose_version" validate:"omitempty"`
	Services       []string  `yaml:"services" validate:"dive,alphanum|contains=-|contains=_"`
}

type SearchResult struct {
	Name          string
	Description   string
	Source        string
	LatestVersion string
	Tags          []string
	Verified      bool
	Homepage      string
}

type Client struct {
	repoURL    string `validate:"required,url,startswith=https://"`
	repoPath   string `validate:"required,dirpath"`
	paksSubdir string `validate:"required"`
	cache      *Index
	validator  *validator.Validate
}

func NewClient() *Client {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	repoPath := filepath.Join(homeDir, ".compak", "index")

	repoURL := os.Getenv("COMPAK_INDEX_REPO")
	if repoURL == "" {
		repoURL = defaultRepoURL
	}

	paksSubdir := os.Getenv("COMPAK_INDEX_PATH")
	if paksSubdir == "" {
		paksSubdir = defaultPaksSubdir
	}

	return &Client{
		repoURL:    repoURL,
		repoPath:   repoPath,
		paksSubdir: paksSubdir,
		validator:  validator.New(),
	}
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if err := c.updateIndex(ctx); err != nil {
		return nil, fmt.Errorf("failed to update index: %w", err)
	}

	query = strings.ToLower(query)
	results := lo.FilterMap(lo.Values(c.cache.Paks), func(pak PakMetadata, _ int) (SearchResult, bool) {
		if query != "" {
			matches := strings.Contains(strings.ToLower(pak.Name), query) ||
				strings.Contains(strings.ToLower(pak.Description), query) ||
				lo.SomeBy(pak.Tags, func(tag string) bool {
					return strings.Contains(strings.ToLower(tag), query)
				})
			if !matches {
				return SearchResult{}, false
			}
		}

		latestVersion := ""
		if len(pak.Versions) > 0 {
			latestVersion = pak.Versions[0].Version
		}

		return SearchResult{
			Name:          pak.Name,
			Description:   pak.Description,
			Source:        pak.Source,
			LatestVersion: latestVersion,
			Tags:          pak.Tags,
			Verified:      pak.Verified,
			Homepage:      pak.Homepage,
		}, true
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (c *Client) GetPak(ctx context.Context, name string) (*PakMetadata, error) {
	if err := c.updateIndex(ctx); err != nil {
		return nil, fmt.Errorf("failed to update index: %w", err)
	}

	pak, exists := c.cache.Paks[name]
	if !exists {
		return nil, fmt.Errorf("pak %s not found in index", name)
	}

	return &pak, nil
}

func (c *Client) updateIndex(ctx context.Context) error {
	if c.cache != nil && time.Since(c.cache.Updated) < 1*time.Hour {
		return nil
	}

	if err := c.ensureRepo(ctx); err != nil {
		return fmt.Errorf("failed to ensure repo: %w", err)
	}

	paks := make(map[string]PakMetadata)
	paksPath := filepath.Join(c.repoPath, c.paksSubdir)

	if _, err := os.Stat(paksPath); err == nil {
		if err := c.loadLocalPaks(paks, paksPath); err != nil {
			return fmt.Errorf("failed to load paks: %w", err)
		}
	}

	c.cache = &Index{
		Paks:    paks,
		Updated: time.Now(),
	}

	return nil
}

func (c *Client) ensureRepo(ctx context.Context) error {
	if _, err := os.Stat(filepath.Join(c.repoPath, ".git")); err == nil {
		return nil
	}

	if err := c.validator.Struct(c); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(c.repoPath), 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	_, err := git.PlainCloneContext(ctx, c.repoPath, false, &git.CloneOptions{
		URL:           c.repoURL,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.HEAD,
	})
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func (c *Client) Update(ctx context.Context) error {
	if err := c.ensureRepo(ctx); err != nil {
		return err
	}

	if err := c.validator.Struct(c); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	repo, err := git.PlainOpen(c.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.PullContext(ctx, &git.PullOptions{
		RemoteName:    "origin",
		SingleBranch:  true,
		Force:         false,
		ReferenceName: plumbing.HEAD,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("git pull failed: %w", err)
	}

	c.cache = nil
	return nil
}

func (c *Client) loadLocalPaks(paks map[string]PakMetadata, paksPath string) (err error) {
	root, err := os.OpenRoot(c.repoPath)
	if err != nil {
		return fmt.Errorf("failed to create root: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	relPaksPath, err := filepath.Rel(c.repoPath, paksPath)
	if err != nil {
		return fmt.Errorf("failed to get relative paks path: %w", err)
	}

	dirFS := root.FS()
	entries, err := fs.ReadDir(dirFS, relPaksPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		pakPath := filepath.Join(relPaksPath, entry.Name())
		file, err := root.Open(pakPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", pakPath, err)
		}

		data, err := io.ReadAll(file)
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", pakPath, err)
		}

		var pak PakMetadata
		if err := yaml.Unmarshal(data, &pak); err != nil {
			return fmt.Errorf("failed to parse %s: %w", pakPath, err)
		}

		if err := c.validator.Struct(&pak); err != nil {
			return fmt.Errorf("validation failed for %s: %w", pakPath, err)
		}

		paks[pak.Name] = pak
	}

	return nil
}

func (c *Client) ListPaks(ctx context.Context) ([]string, error) {
	if err := c.updateIndex(ctx); err != nil {
		return nil, fmt.Errorf("failed to update index: %w", err)
	}

	return lo.Keys(c.cache.Paks), nil
}

func (c *Client) LoadPackageFromIndex(ctx context.Context, name string) (data []byte, err error) {
	if err := c.ensureRepo(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure repo: %w", err)
	}

	pakPath := filepath.Join(c.repoPath, c.paksSubdir, name+".yaml")

	root, err := os.OpenRoot(c.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create root: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	relPath, err := filepath.Rel(c.repoPath, pakPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	file, err := root.Open(relPath)
	if err != nil {
		return nil, fmt.Errorf("package %s not found in index", name)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	data, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read package: %w", err)
	}

	return data, nil
}
