package read

import (
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

type Node interface {
	Read() ([]byte, error)
	Parent() Node
	Optional() bool
	Location() string
	Remote() bool
}

func NewNodeFromIncludedTaskfile(
	parent Node,
	includedTaskfile taskfile.IncludedTaskfile,
	allowInsecure bool,
	tempDir string,
	l *logger.Logger,
) (Node, error) {
	if !experiments.RemoteTaskfiles {
		path, err := includedTaskfile.FullTaskfilePath()
		if err != nil {
			return nil, err
		}
		return NewFileNode(parent, path, includedTaskfile.Optional)
	}
	switch getScheme(includedTaskfile.Taskfile) {
	case "http":
		if !allowInsecure {
			return nil, &errors.TaskfileNotSecureError{URI: includedTaskfile.Taskfile}
		}
		return NewHTTPNode(parent, includedTaskfile.Taskfile, includedTaskfile.Optional)
	case "https":
		return NewHTTPNode(parent, includedTaskfile.Taskfile, includedTaskfile.Optional)
	// If no other scheme matches, we assume it's a file.
	// This also allows users to explicitly set a file:// scheme.
	default:
		path, err := includedTaskfile.FullTaskfilePath()
		if err != nil {
			return nil, err
		}
		return NewFileNode(parent, path, includedTaskfile.Optional)
	}
}

func getScheme(uri string) string {
	if i := strings.Index(uri, "://"); i != -1 {
		return uri[:i]
	}
	return ""
}
