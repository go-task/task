package taskfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/hashicorp/go-getter"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// An GitNode is a node that reads a Taskfile from a remote location via Git.
type GitNode struct {
	*baseNode
	url    *url.URL
	rawUrl string
	ref    string
	path   string
}

type gitRepoCache struct {
	mu    sync.Mutex             // Protects the locks map
	locks map[string]*sync.Mutex // One mutex per repo cache key
}

func (c *gitRepoCache) getLockForRepo(cacheKey string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.locks[cacheKey]; !exists {
		c.locks[cacheKey] = &sync.Mutex{}
	}

	return c.locks[cacheKey]
}

var globalGitRepoCache = &gitRepoCache{
	locks: make(map[string]*sync.Mutex),
}

func CleanGitCache() error {
	// Clear the in-memory locks map to prevent memory leak
	globalGitRepoCache.mu.Lock()
	globalGitRepoCache.locks = make(map[string]*sync.Mutex)
	globalGitRepoCache.mu.Unlock()

	cacheDir := filepath.Join(os.TempDir(), "task-git-repos")
	return os.RemoveAll(cacheDir)
}

func NewGitNode(
	entrypoint string,
	dir string,
	insecure bool,
	opts ...NodeOption,
) (*GitNode, error) {
	base := NewBaseNode(dir, opts...)
	u, err := giturls.Parse(entrypoint)
	if err != nil {
		return nil, err
	}

	basePath, path := splitURLOnDoubleSlash(u)
	ref := u.Query().Get("ref")

	rawUrl := u.Redacted()

	u.RawQuery = ""
	u.Path = basePath

	if u.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: u.Redacted()}
	}
	return &GitNode{
		baseNode: base,
		url:      u,
		rawUrl:   rawUrl,
		ref:      ref,
		path:     path,
	}, nil
}

func (node *GitNode) Location() string {
	return node.rawUrl
}

func (node *GitNode) Remote() bool {
	return true
}

func (node *GitNode) Read() ([]byte, error) {
	return node.ReadContext(context.Background())
}

func (node *GitNode) buildURL() string {
	// Get the base URL
	baseURL := node.url.String()

	ref := node.ref
	if ref == "" {
		ref = "HEAD"
	}
	// Always use git:: prefix for git URLs (following Terraform's pattern)
	// This forces go-getter to use git protocol
	return fmt.Sprintf("git::%s?ref=%s&depth=1", baseURL, ref)
}

// getOrCloneRepo returns the path to a cached git repository.
// If the repository is not cached, it clones it first.
// This function is thread-safe: multiple goroutines cloning the same repo+ref
// will synchronize, and only one clone operation will occur.
//
// The cache directory is /tmp/task-git-repos/{cache_key}/
func (node *GitNode) getOrCloneRepo(ctx context.Context) (string, error) {
	cacheKey := node.repoCacheKey()

	repoMutex := globalGitRepoCache.getLockForRepo(cacheKey)
	repoMutex.Lock()
	defer repoMutex.Unlock()

	// Check if context was cancelled while waiting for lock
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled while waiting for repository lock: %w", err)
	}

	cacheDir := filepath.Join(os.TempDir(), "task-git-repos", cacheKey)

	// check if repo is already cached (under the lock)
	gitDir := filepath.Join(cacheDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return cacheDir, nil
	}

	getterURL := node.buildURL()

	client := &getter.Client{
		Ctx:  ctx,
		Src:  getterURL,
		Dst:  cacheDir,
		Mode: getter.ClientModeDir,
	}

	if err := client.Get(); err != nil {
		_ = os.RemoveAll(cacheDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return cacheDir, nil
}

func (node *GitNode) ReadContext(ctx context.Context) ([]byte, error) {
	// Get or clone the repository into cache
	repoDir, err := node.getOrCloneRepo(ctx)
	if err != nil {
		return nil, err
	}

	// Build path to Taskfile in the cached repo
	taskfilePath := node.path
	if taskfilePath == "" {
		taskfilePath = "Taskfile.yml"
	}
	filePath := filepath.Join(repoDir, taskfilePath)

	// Read file from cached repo
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (node *GitNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if isRemoteEntrypoint(entrypoint) {
		return entrypoint, nil
	}

	dir, _ := filepath.Split(node.path)
	resolvedEntrypoint := fmt.Sprintf("%s//%s", node.url, filepath.Join(dir, entrypoint))
	if node.ref != "" {
		return fmt.Sprintf("%s?ref=%s", resolvedEntrypoint, node.ref), nil
	}
	return resolvedEntrypoint, nil
}

func (node *GitNode) ResolveDir(dir string) (string, error) {
	path, err := execext.ExpandLiteral(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	entrypointDir := filepath.Dir(node.Dir())
	return filepathext.SmartJoin(entrypointDir, path), nil
}

func (node *GitNode) CacheKey() string {
	checksum := strings.TrimRight(checksum([]byte(node.Location())), "=")
	lastDir := filepath.Base(filepath.Dir(node.path))
	prefix := filepath.Base(node.path)
	// Means it's not "", nor "." nor "/", so it's a valid directory
	if len(lastDir) > 1 {
		prefix = fmt.Sprintf("%s.%s", lastDir, prefix)
	}
	return fmt.Sprintf("git.%s.%s.%s", node.url.Host, prefix, checksum)
}

// repoCacheKey generates a unique cache key for the repository+ref combination.
// Unlike CacheKey() which includes the file path, this identifies the repository itself.
// Two GitNodes with the same repo+ref but different file paths will share the same cache.
//
// Returns a path like: github.com/user/repo.git/main
func (node *GitNode) repoCacheKey() string {
	repoPath := strings.Trim(node.url.Path, "/")

	ref := node.ref
	if ref == "" {
		ref = "HEAD"
	}

	return filepath.Join(node.url.Host, repoPath, ref)
}

func splitURLOnDoubleSlash(u *url.URL) (string, string) {
	x := strings.Split(u.Path, "//")
	switch len(x) {
	case 0:
		return "", ""
	case 1:
		return x[0], ""
	default:
		return x[0], x[1]
	}
}
