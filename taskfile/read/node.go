package read

import (
	"strings"

	"github.com/go-task/task/v3/taskfile"
)

type Node interface {
	Read() (*taskfile.Taskfile, error)
	Parent() Node
	Optional() bool
	Location() string
}

func NewNodeFromIncludedTaskfile(parent Node, includedTaskfile taskfile.IncludedTaskfile) (Node, error) {
	switch getScheme(includedTaskfile.Taskfile) {
	// TODO: Add support for other schemes.
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
