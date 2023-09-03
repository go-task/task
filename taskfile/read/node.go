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
	if !experiments.RemoteTaskfiles {
		return NewFileNode(parent, uri)
	}
	switch getScheme(uri) {
	case "http":
		if !allowInsecure {
			return nil, &errors.TaskfileNotSecureError{URI: uri}
		}
		return NewHTTPNode(parent, uri)
	case "https":
		return NewHTTPNode(parent, uri)
	// If no other scheme matches, we assume it's a file.
	// This also allows users to explicitly set a file:// scheme.
	default:
		return NewFileNode(parent, uri)
	}
}

func getScheme(uri string) string {
	if i := strings.Index(uri, "://"); i != -1 {
		return uri[:i]
	}
	return ""
}
