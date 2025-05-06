package taskfile

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"

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

func (node *GitNode) ReadContext(_ context.Context) ([]byte, error) {
	fs := memfs.New()
	storer := memory.NewStorage()
	_, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           node.url.String(),
		ReferenceName: plumbing.ReferenceName(node.ref),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, err
	}
	file, err := fs.Open(node.path)
	if err != nil {
		return nil, err
	}
	// Read the entire response body
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (node *GitNode) ResolveEntrypoint(entrypoint string) (string, error) {
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
