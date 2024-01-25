package taskfile

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
)

type Node interface {
	Read(ctx context.Context) ([]byte, error)
	Parent() Node
	Location() string
	Optional() bool
	Remote() bool
	BaseDir() string
}

func NewRootNode(
	dir string,
	entrypoint string,
	insecure bool,
) (Node, error) {
	// Check if there is something to read on STDIN
	stat, _ := os.Stdin.Stat()
	if (stat.Mode()&os.ModeCharDevice) == 0 && stat.Size() > 0 {
		return NewStdinNode(dir)
	}
	// If no entrypoint is specified, search for a taskfile
	if entrypoint == "" {
		root, err := ExistsWalk(dir)
		if err != nil {
			return nil, err
		}
		return NewNode(root, insecure)
	}
	// Use the specified entrypoint
	uri := filepath.Join(dir, entrypoint)
	return NewNode(uri, insecure)
}

func NewNode(
	uri string,
	insecure bool,
	opts ...NodeOption,
) (Node, error) {
	var node Node
	var err error
	switch getScheme(uri) {
	case "http", "https":
		node, err = NewHTTPNode(uri, insecure, opts...)
	default:
		// If no other scheme matches, we assume it's a file
		node, err = NewFileNode(uri, opts...)
	}
	if node.Remote() && !experiments.RemoteTaskfiles.Enabled {
		return nil, errors.New("task: Remote taskfiles are not enabled. You can read more about this experiment and how to enable it at https://taskfile.dev/experiments/remote-taskfiles")
	}
	return node, err
}

func getScheme(uri string) string {
	if i := strings.Index(uri, "://"); i != -1 {
		return uri[:i]
	}
	return ""
}
