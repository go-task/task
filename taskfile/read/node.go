package read

import (
	"context"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
)

type Node interface {
	Read(ctx context.Context) ([]byte, error)
	Parent() Node
	Location() string
	Remote() bool
}

func NewNode(
	parent Node,
	uri string,
	allowInsecure bool,
) (Node, error) {
	var node Node
	var err error
	switch getScheme(uri) {
	case "http":
		node, err = NewHTTPNode(parent, uri, allowInsecure)
	case "https":
		node, err = NewHTTPNode(parent, uri, allowInsecure)
	default:
		// If no other scheme matches, we assume it's a file
		node, err = NewFileNode(parent, uri)
	}
	if node.Remote() && !experiments.RemoteTaskfiles {
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
