package taskfile

import (
	"context"
	"strings"
	"time"

	giturls "github.com/chainguard-dev/git-urls"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/fsext"
)

type Node interface {
	Read() ([]byte, error)
	Parent() Node
	Location() string
	Dir() string
	Checksum() string
	Verify(checksum string) bool
	ResolveEntrypoint(entrypoint string) (string, error)
	ResolveDir(dir string) (string, error)
}

type RemoteNode interface {
	Node
	ReadContext(ctx context.Context) ([]byte, error)
	CacheKey() string
}

func NewRootNode(
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
) (Node, error) {
	dir = fsext.DefaultDir(entrypoint, dir)
	// If the entrypoint is "-", we read from stdin
	if entrypoint == "-" {
		return NewStdinNode(dir)
	}
	return NewNode(entrypoint, dir, insecure)
}

func NewNode(
	entrypoint string,
	dir string,
	insecure bool,
	opts ...NodeOption,
) (Node, error) {
	var node Node
	var err error

	scheme, err := getScheme(entrypoint)
	if err != nil {
		return nil, err
	}

	switch scheme {
	case "git":
		node, err = NewGitNode(entrypoint, dir, insecure, opts...)
	case "http", "https":
		node, err = NewHTTPNode(entrypoint, dir, insecure, opts...)
	default:
		node, err = NewFileNode(entrypoint, dir, opts...)
	}
	if _, isRemote := node.(RemoteNode); isRemote && !experiments.RemoteTaskfiles.Enabled() {
		return nil, errors.New("task: Remote taskfiles are not enabled. You can read more about this experiment and how to enable it at https://taskfile.dev/experiments/remote-taskfiles")
	}

	return node, err
}

func isRemoteEntrypoint(entrypoint string) bool {
	scheme, _ := getScheme(entrypoint)
	switch scheme {
	case "git", "http", "https":
		return true
	default:
		return false
	}
}

func getScheme(uri string) (string, error) {
	u, err := giturls.Parse(uri)
	if u == nil {
		return "", err
	}

	if strings.HasSuffix(strings.Split(u.Path, "//")[0], ".git") && (u.Scheme == "git" || u.Scheme == "ssh" || u.Scheme == "https" || u.Scheme == "http") {
		return "git", nil
	}

	if i := strings.Index(uri, "://"); i != -1 {
		return uri[:i], nil
	}

	return "", nil
}
