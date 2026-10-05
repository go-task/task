package taskfile

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitNode_ssh(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_sshWithAltRepo(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)

	entrypoint, err := node.ResolveEntrypoint("git@github.com:foo/other.git//Taskfile.yml?ref=dev")
	assert.NoError(t, err)
	assert.Equal(t, "git@github.com:foo/other.git//Taskfile.yml?ref=dev", entrypoint)
}

func TestGitNode_sshWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_https(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://github.com/foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git//Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "https://github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_httpsWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "https://github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_CacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		entrypoint  string
		expectedKey string
	}{
		{
			entrypoint:  "https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main",
			expectedKey: "git.github.com.directory.Taskfile.yml.f1ddddac425a538870230a3e38fc0cded4ec5da250797b6cab62c82477718fbb",
		},
		{
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			expectedKey: "git.github.com.Taskfile.yml.39d28c1ff36f973705ae188b991258bbabaffd6d60bcdde9693d157d00d5e3a4",
		},
		{
			entrypoint:  "https://github.com/foo/bar.git//multiple/directory/Taskfile.yml?ref=main",
			expectedKey: "git.github.com.directory.Taskfile.yml.1b6d145e01406dcc6c0aa572e5a5d1333be1ccf2cae96d18296d725d86197d31",
		},
	}

	for _, tt := range tests {
		node, err := NewGitNode(tt.entrypoint, "", false)
		require.NoError(t, err)
		key := node.CacheKey()
		assert.Equal(t, tt.expectedKey, key)
	}
}

func TestGitNode_buildURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entrypoint  string
		expectedURL string
	}{
		{
			name:        "HTTPS with ref",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			expectedURL: "git::https://github.com/foo/bar.git?ref=main&depth=1",
		},
		{
			name:        "SSH with ref",
			entrypoint:  "git@github.com:foo/bar.git//Taskfile.yml?ref=main",
			expectedURL: "git::ssh://git@github.com/foo/bar.git?ref=main&depth=1",
		},
		{
			name:        "HTTPS with tag ref",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=v1.0.0",
			expectedURL: "git::https://github.com/foo/bar.git?ref=v1.0.0&depth=1",
		},
		{
			name:        "HTTPS without ref (uses remote default branch)",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml",
			expectedURL: "git::https://github.com/foo/bar.git?depth=1",
		},
		{
			name:        "SSH with directory path",
			entrypoint:  "git@github.com:foo/bar.git//directory/Taskfile.yml?ref=dev",
			expectedURL: "git::ssh://git@github.com/foo/bar.git?ref=dev&depth=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node, err := NewGitNode(tt.entrypoint, "", false)
			require.NoError(t, err)
			gotURL := node.buildURL()
			assert.Equal(t, tt.expectedURL, gotURL)
		})
	}
}

func TestRepoCacheKey_SameRepoSameRef(t *testing.T) {
	t.Parallel()

	// Same repo, same ref, different files should have SAME cache key
	node1, err := NewGitNode("https://github.com/foo/bar.git//file1.yml?ref=main", "", false)
	require.NoError(t, err)

	node2, err := NewGitNode("https://github.com/foo/bar.git//dir/file2.yml?ref=main", "", false)
	require.NoError(t, err)

	key1 := node1.repoCacheKey()
	key2 := node2.repoCacheKey()

	assert.Equal(t, key1, key2, "Same repo+ref should generate same cache key regardless of file path")
}

func TestRepoCacheKey_SameRepoDifferentRef(t *testing.T) {
	t.Parallel()

	// Same repo, different ref should have DIFFERENT cache keys
	node1, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	node2, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=dev", "", false)
	require.NoError(t, err)

	key1 := node1.repoCacheKey()
	key2 := node2.repoCacheKey()

	assert.NotEqual(t, key1, key2, "Different refs should generate different cache keys")
}

func TestRepoCacheKey_DifferentRepos(t *testing.T) {
	t.Parallel()

	// Different repos should have DIFFERENT cache keys
	node1, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	node2, err := NewGitNode("https://github.com/foo/other.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	key1 := node1.repoCacheKey()
	key2 := node2.repoCacheKey()

	assert.NotEqual(t, key1, key2, "Different repos should generate different cache keys")
}

func TestRepoCacheKey_NoRefVsExplicitRef(t *testing.T) {
	t.Parallel()

	// No ref (uses default branch) vs explicit ref should have DIFFERENT cache keys
	node1, err := NewGitNode("https://github.com/foo/bar.git//file.yml", "", false)
	require.NoError(t, err)

	node2, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	key1 := node1.repoCacheKey()
	key2 := node2.repoCacheKey()

	assert.NotEqual(t, key1, key2, "No ref and explicit ref should generate different cache keys")
}

func TestRepoCacheKey_SSHvsHTTPS(t *testing.T) {
	t.Parallel()

	// SSH vs HTTPS pointing to same repo should have SAME cache key
	// They clone the same repo, so we want to share the cache
	node1, err := NewGitNode("git@github.com:foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	node2, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	key1 := node1.repoCacheKey()
	key2 := node2.repoCacheKey()

	assert.Equal(t, key1, key2, "SSH and HTTPS for same repo should share cache")
}

func TestRepoCacheKey_Consistency(t *testing.T) {
	t.Parallel()

	// Calling repoCacheKey multiple times on same node should return same key
	node, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=main", "", false)
	require.NoError(t, err)

	key1 := node.repoCacheKey()
	key2 := node.repoCacheKey()
	key3 := node.repoCacheKey()

	assert.Equal(t, key1, key2)
	assert.Equal(t, key2, key3)
}

func TestRepoCacheKey_BlocksTraversalRef(t *testing.T) {
	t.Parallel()

	// A malicious ref containing path traversal components must not be able to make
	// the cache directory escape the task-git-repos root (CWE-22, GHSA-g8jx-8vm6-phr8).
	node, err := NewGitNode("https://github.com/foo/bar.git//file.yml?ref=../../../../victim-delete-me", "", false)
	require.NoError(t, err)

	key := node.repoCacheKey()

	// The key itself must not carry traversal components.
	assert.NotContains(t, key, "..", "cache key must not contain traversal components")

	// Joined under the cache root, the resolved path must stay inside it.
	root, err := gitCacheRoot()
	require.NoError(t, err)
	resolved := filepath.Clean(filepath.Join(root, key))
	assert.True(t, strings.HasPrefix(resolved, root+string(os.PathSeparator)),
		"cache dir %q must stay within %q", resolved, root)
}

func setUserCacheDir(t *testing.T, dir string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", dir)
	case "darwin":
		t.Setenv("HOME", dir)
	default:
		t.Setenv("XDG_CACHE_HOME", dir)
	}
}

func unsetUserCacheDir(t *testing.T) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", "")
	case "darwin":
		t.Setenv("HOME", "")
	default:
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")
	}
}

func TestGitCacheRoot_NotInSharedTempDir(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	dir := t.TempDir()
	setUserCacheDir(t, dir)

	root, err := gitCacheRoot()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(root, dir), "cache root %q must live under the user cache dir %q", root, dir)
	assert.NotEqual(t, filepath.Join(os.TempDir(), gitCacheDirName), root)

	require.NoError(t, prepareGitCacheRoot(root))
	info, err := os.Stat(root)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), "cache root must be private to the current user")
	}
}

func TestGitCacheRoot_FallbackIsPrivate(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	// No user cache dir: the fallback must be private and unpredictable.
	unsetUserCacheDir(t)

	root, err := gitCacheRoot()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	assert.True(t, strings.HasPrefix(root, os.TempDir()), "fallback %q must live under the temp dir", root)
	assert.NotEqual(t, filepath.Join(os.TempDir(), gitCacheDirName), root, "fallback name must not be predictable")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(root)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), "fallback must be private to the current user")
	}
}

func TestPrepareGitCacheRoot_RejectsSymlink(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.Mkdir(target, 0o700))

	root := filepath.Join(dir, "task-git-repos")
	require.NoError(t, os.Symlink(target, root))

	err := prepareGitCacheRoot(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestGetOrCloneRepo_ReusesCompletedClone(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	setUserCacheDir(t, t.TempDir())

	node, err := NewGitNode("file:///nonexistent/repo.git//Taskfile.yml?ref=main", "", false)
	require.NoError(t, err)

	// A .git entry is reused without a clone; the remote does not exist, so a clone would fail.
	root, err := gitCacheRoot()
	require.NoError(t, err)
	cacheDir := filepath.Join(root, node.repoCacheKey())
	require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, ".git"), 0o700))

	got, err := node.getOrCloneRepo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, cacheDir, got)
}

func TestGetOrCloneRepo_ReclonesWithoutGitDir(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	setUserCacheDir(t, t.TempDir())

	node, err := NewGitNode("file:///nonexistent/repo.git//Taskfile.yml?ref=main", "", false)
	require.NoError(t, err)

	// A directory without .git must trigger a fresh clone.
	root, err := gitCacheRoot()
	require.NoError(t, err)
	cacheDir := filepath.Join(root, node.repoCacheKey())
	require.NoError(t, os.MkdirAll(cacheDir, 0o700))

	_, err = node.getOrCloneRepo(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

type blockingGitCacheReadNode struct {
	Node
	readyPath   string
	releasePath string
}

func (node blockingGitCacheReadNode) Read() ([]byte, error) {
	if err := os.WriteFile(node.readyPath, nil, 0o600); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(node.releasePath); err == nil {
			return node.Node.Read()
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, os.ErrDeadlineExceeded
}

func TestGitCacheReaderHelper(t *testing.T) {
	t.Parallel()
	if os.Getenv("GO_TASK_GIT_CACHE_READER_HELPER") != "1" {
		return
	}

	baseNode, err := NewFileNode(os.Getenv("GO_TASK_GIT_CACHE_TASKFILE"), "")
	require.NoError(t, err)
	node := blockingGitCacheReadNode{
		Node:        baseNode,
		readyPath:   os.Getenv("GO_TASK_GIT_CACHE_READY"),
		releasePath: os.Getenv("GO_TASK_GIT_CACHE_RELEASE"),
	}
	_, err = NewReader().Read(context.Background(), node)
	require.NoError(t, err)
}

func TestReaderReadPreventsConcurrentGitCacheCleanup(t *testing.T) { //nolint:paralleltest // t.Setenv changes the user cache directory
	cacheDir := t.TempDir()
	setUserCacheDir(t, cacheDir)
	root, err := gitCacheRoot()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(root, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cached-repo"), nil, 0o600))

	taskfile := filepath.Join(t.TempDir(), "Taskfile.yml")
	require.NoError(t, os.WriteFile(taskfile, []byte("version: '3'\n"), 0o600))
	readyPath := filepath.Join(t.TempDir(), "reader-ready")
	releasePath := filepath.Join(filepath.Dir(readyPath), "release-reader")

	//nolint:gosec // Re-execute this test binary to model a separate Task process.
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestGitCacheReaderHelper$")
	cmd.Env = append(os.Environ(),
		"GO_TASK_GIT_CACHE_READER_HELPER=1",
		"GO_TASK_GIT_CACHE_TASKFILE="+taskfile,
		"GO_TASK_GIT_CACHE_READY="+readyPath,
		"GO_TASK_GIT_CACHE_RELEASE="+releasePath,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(readyPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(readyPath); err != nil {
		t.Fatalf("reader helper did not start reading: %s", err)
	}

	cleanupDone := make(chan error, 1)
	go func() { cleanupDone <- cleanGitCache(root) }()
	select {
	case err := <-cleanupDone:
		_ = os.WriteFile(releasePath, nil, 0o600)
		t.Fatalf("git cache cleanup completed while another process was reading: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	require.NoError(t, os.WriteFile(releasePath, nil, 0o600))
	select {
	case err := <-cleanupDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("git cache cleanup did not resume after the reader finished")
	}
	require.NoError(t, cmd.Wait())
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected git cache root to be removed, stat error: %v", err)
	}
}
