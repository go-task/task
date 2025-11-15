package taskfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func (node *GitNode) ReadContext(ctx context.Context) ([]byte, error) {
	return node.readWithGoGetter(ctx)
}

func (node *GitNode) readWithGoGetter(ctx context.Context) ([]byte, error) {
	// IMPORTANT: Do NOT create tmpDir in advance!
	// If the directory exists, go-getter will use update() instead of clone()
	// which is 3x slower (git init + fetch --tags + pull instead of a simple clone)
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("task-git-getter-%d", time.Now().UnixNano()))

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	getterURL := node.buildURL()

	client := &getter.Client{
		Ctx:  ctx,
		Src:  getterURL,
		Dst:  tmpDir,
		Mode: getter.ClientModeDir,
	}

	// Clone repository into tmpdir
	if err := client.Get(); err != nil {
		return nil, err
	}

	// Build path to Taskfile in tmpdir
	taskfilePath := node.path
	if taskfilePath == "" {
		taskfilePath = "Taskfile.yml"
	}
	filePath := filepath.Join(tmpDir, taskfilePath)

	// Read file
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
