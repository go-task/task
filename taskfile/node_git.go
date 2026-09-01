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

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/hashicorp/go-getter"
	"golang.org/x/sys/unix"

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

// lockRepoCache takes an exclusive cross-process lock for one repo cache key.
// The in-process mutex (gitRepoCache) does not protect against multiple Task
// *processes* cloning the same repository concurrently (issue #3011), so we
// also flock a lock file that lives outside the cache dir (which CleanGitCache
// removes).
func lockRepoCache(cacheKey string) (func(), error) {
	lockDir := filepath.Join(os.TempDir(), "task-git-repos-locks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating git repo lock dir: %w", err)
	}
	lockFile := filepath.Join(lockDir, cacheKey+".lock")
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening git repo lock file: %w", err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("locking git repo cache: %w", err)
	}
	return func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
	}, nil
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
// The cache directory is /tmp/task-git-repos/{cache_key}/
func (node *GitNode) getOrCloneRepo(ctx context.Context) (string, error) {
	cacheKey := node.repoCacheKey()

	repoMutex := globalGitRepoCache.getLockForRepo(cacheKey)
	repoMutex.Lock()
	defer repoMutex.Unlock()

	cacheDir := filepath.Join(os.TempDir(), "task-git-repos", cacheKey)

	// Cross-process lock: an IDE extension, shell completion and a terminal can
	// invoke Task at the same time; the per-process mutex above does not
	// serialize them (issue #3011). Serialize the clone with a file lock.
	unlock, err := lockRepoCache(cacheKey)
	if err != nil {
		return "", err
	}
	defer unlock()

	// Check cache FIRST - if already cloned, no network needed, timeout irrelevant
	gitDir := filepath.Join(cacheDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return cacheDir, nil
	}

	// Only check context if we need to clone (requires network)
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled while waiting for repository lock: %w", err)
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
