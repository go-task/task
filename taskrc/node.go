package taskrc

import (
	"github.com/go-task/task/v3/internal/fsext"
)

type Node struct {
	entrypoint string
}

func NewNode(
	entrypoint string,
	dir string,
) (*Node, error) {
	dir = fsext.DefaultDir(entrypoint, dir)
	resolvedEntrypoint, err := fsext.SearchPath(dir, defaultTaskRCs)
	if err != nil {
		return nil, err
	}
	return &Node{
		entrypoint: resolvedEntrypoint,
	}, nil
}
