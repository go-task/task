package taskfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/gofrs/flock"
	"github.com/hashicorp/go-getter"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fsext"
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

const gitCacheDirName = "task-git-repos"

const gitCacheLockRetryDelay = 10 * time.Millisecond

var (
	fallbackGitCacheOnce sync.Once
	fallbackGitCacheDir  string
	fallbackGitCacheErr  error
)

// gitCacheRoot must stay private to the current user: a shared, predictable
// location lets another local user plant an entry Task would read as the remote's.
func gitCacheRoot() (string, error) {
	if userCacheDir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(userCacheDir, "task", gitCacheDirName), nil
	}
	// No user cache dir (e.g. a container with no HOME): a randomly named dir in
	// the shared temp dir is unpredictable, so it cannot be pre-planted.
	fallbackGitCacheOnce.Do(func() {
		fallbackGitCacheDir, fallbackGitCacheErr = os.MkdirTemp("", gitCacheDirName+"-")
	})
	return fallbackGitCacheDir, fallbackGitCacheErr
}

func prepareGitCacheRoot(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("failed to create git cache directory: %w", err)
	}

	// Lstat, not Stat: a symlink here would redirect both the clone and the cleanup.
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("failed to inspect git cache directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("git cache directory %q is not a directory", root)
	}

	return nil
}

func CleanGitCache() error {
	root, err := gitCacheRoot()
	if err != nil {
		return err
	}
	return cleanGitCache(root)
}

func cleanGitCache(root string) error {
	unlock, err := lockGitCache(context.Background(), root, false)
	if err != nil {
		return err
	}
	defer func() { _ = unlock() }()

	// Clear the in-memory locks map to prevent memory leak
	globalGitRepoCache.mu.Lock()
	globalGitRepoCache.locks = make(map[string]*sync.Mutex)
	globalGitRepoCache.mu.Unlock()

	return os.RemoveAll(root)
}

// lockGitCache coordinates readers with cleanup across Task processes. A
// reader holds a shared lock until it has read every remote Taskfile; cleanup
// takes an exclusive lock so one process cannot remove another process's clone.
func lockGitCache(ctx context.Context, root string, shared bool) (func() error, error) {
	lockDir := filepath.Dir(root)
	if err := prepareGitCacheRoot(lockDir); err != nil {
		return nil, fmt.Errorf("failed to prepare git cache lock directory: %w", err)
	}
	lock := flock.New(root+".lock",
		flock.SetPermissions(0o600),
		flock.SetFlag(os.O_CREATE|os.O_RDWR),
	)
	var locked bool
	var err error
	if shared {
		locked, err = lock.TryRLockContext(ctx, gitCacheLockRetryDelay)
	} else {
		locked, err = lock.TryLockContext(ctx, gitCacheLockRetryDelay)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock git cache: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("failed to lock git cache")
	}
	return lock.Unlock, nil
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

	// Always use git:: prefix for git URLs (following Terraform's pattern)
	// This forces go-getter to use git protocol
	if node.ref != "" {
		return fmt.Sprintf("git::%s?ref=%s&depth=1", baseURL, node.ref)
	}
	// When no ref is specified, omit it entirely to let git clone the default branch
	return fmt.Sprintf("git::%s?depth=1", baseURL)
}

// getOrCloneRepo returns the path to a cached git repository.
// If the repository is not cached, it clones it first.
// This function is thread-safe: multiple goroutines cloning the same repo+ref
// will synchronize, and only one clone operation will occur.
//
// The cache directory is {user cache dir}/task/task-git-repos/{cache_key}/
func (node *GitNode) getOrCloneRepo(ctx context.Context) (string, error) {
	cacheKey := node.repoCacheKey()

	repoMutex := globalGitRepoCache.getLockForRepo(cacheKey)
	repoMutex.Lock()
	defer repoMutex.Unlock()

	root, err := gitCacheRoot()
	if err != nil {
		return "", err
	}
	if err := prepareGitCacheRoot(root); err != nil {
		return "", err
	}
	cacheDir := filepath.Join(root, cacheKey)

	// A .git here means a completed clone: partial clones stay in staging and are
	// only published, whole, by the atomic rename below.
	gitDir := filepath.Join(cacheDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return cacheDir, nil
	}

	// Only check context if we need to clone (requires network)
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled while waiting for repository lock: %w", err)
	}

	parent := filepath.Dir(cacheDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", fmt.Errorf("failed to create git cache directory: %w", err)
	}

	// Clone into staging, publish with a single rename.
	staging, err := os.MkdirTemp(parent, ".staging-")
	if err != nil {
		return "", fmt.Errorf("failed to create git cache directory: %w", err)
	}
	defer os.RemoveAll(staging)

	clonedDir := filepath.Join(staging, "repo")
	client := &getter.Client{
		Ctx:  ctx,
		Src:  node.buildURL(),
		Dst:  clonedDir,
		Mode: getter.ClientModeDir,
	}

	if err := client.Get(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	if err := os.Rename(clonedDir, cacheDir); err != nil {
		// Another process may have published the same entry while we were cloning.
		if _, statErr := os.Stat(gitDir); statErr == nil {
			return cacheDir, nil
		}
		return "", fmt.Errorf("failed to publish git cache entry: %w", err)
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
	// If node.path is empty, search in repo root; otherwise search in the specified path
	// fsext.SearchPath handles both files and directories (searching for DefaultTaskfiles)
	searchPath := repoDir
	if node.path != "" {
		searchPath = filepath.Join(repoDir, node.path)
	}
	filePath, err := fsext.SearchPath(searchPath, DefaultTaskfiles)
	if err != nil {
		return nil, err
	}

	// Read file from cached repo
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (node *GitNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if IsRemoteEntrypoint(entrypoint) {
		return entrypoint, nil
	}

	dir, _ := path.Split(node.path)
	resolvedEntrypoint := fmt.Sprintf("%s//%s", node.url, path.Join(dir, entrypoint))
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
// The identity is hashed into a single, filesystem-safe path segment. This prevents an
// attacker-controlled ref (e.g. "../../victim") from being used as a raw path component,
// which would otherwise let the cache directory escape the task-git-repos root (CWE-22).
//
// Returns a path like: git/<sha256-hex>
func (node *GitNode) repoCacheKey() string {
	repoPath := strings.Trim(node.url.Path, "/")

	ref := node.ref
	if ref == "" {
		ref = "_default_" // Placeholder for the remote's default branch
	}

	identity := strings.Join([]string{node.url.Host, repoPath, ref}, "/")
	return filepath.Join("git", checksum([]byte(identity)))
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
