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
	*BaseNode
	URL    *url.URL
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

	basePath, path := func() (string, string) {
		x := strings.Split(u.Path, "//")
		return x[0], x[1]
	}()
	ref := u.Query().Get("ref")

	rawUrl := u.String()

	u.RawQuery = ""
	u.Path = basePath

	if u.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: entrypoint}
	}
	return &GitNode{
		BaseNode: base,
		URL:      u,
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

func (node *GitNode) Read(_ context.Context) ([]byte, error) {
	fs := memfs.New()
	storer := memory.NewStorage()
	_, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           node.URL.String(),
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
	resolvedEntrypoint := fmt.Sprintf("%s//%s", node.URL, filepath.Join(dir, entrypoint))
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

func (node *GitNode) FilenameAndLastDir() (string, string) {
	return filepath.Base(node.path), filepath.Base(filepath.Dir(node.path))
}
