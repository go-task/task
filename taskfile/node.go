package taskfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	giturls "github.com/chainguard-dev/git-urls"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/logger"
)

type Node interface {
	Read(ctx context.Context) ([]byte, error)
	Parent() Node
	Location() string
	Dir() string
	Remote() bool
	ResolveEntrypoint(entrypoint string) (string, error)
	ResolveDir(dir string) (string, error)
	FilenameAndLastDir() (string, string)
}

func NewRootNode(
	l *logger.Logger,
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
) (Node, error) {
	dir = getDefaultDir(entrypoint, dir)
	// If the entrypoint is "-", we read from stdin
	if entrypoint == "-" {
		return NewStdinNode(dir)
	}
	return NewNode(l, entrypoint, dir, insecure, timeout)
}

func NewNode(
	l *logger.Logger,
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
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
		node, err = NewHTTPNode(l, entrypoint, dir, insecure, timeout, opts...)
	default:
		node, err = NewFileNode(l, entrypoint, dir, opts...)

	}

	if node.Remote() && !experiments.RemoteTaskfiles.Enabled {
		return nil, errors.New("task: Remote taskfiles are not enabled. You can read more about this experiment and how to enable it at https://taskfile.dev/experiments/remote-taskfiles")
	}
	return node, err
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

func getDefaultDir(entrypoint, dir string) string {
	// If the entrypoint and dir are empty, we default the directory to the current working directory
	if dir == "" {
		if entrypoint == "" {
			wd, err := os.Getwd()
			if err != nil {
				return ""
			}
			dir = wd
		}
		return dir
	}

	// If the directory is set, ensure it is an absolute path
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return ""
	}

	return dir
}
