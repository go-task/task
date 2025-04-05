package taskrc

import "github.com/go-task/task/v3/internal/fsext"

type Node struct {
	entrypoint string
	dir        string
}

func NewNode(
	entrypoint string,
	dir string,
) (*Node, error) {
	dir = fsext.DefaultDir(entrypoint, dir)
	var err error
	entrypoint, dir, err = fsext.Search(entrypoint, dir, defaultTaskRCs)
	if err != nil {
		return nil, err
	}
	return &Node{
		entrypoint: entrypoint,
		dir:        dir,
	}, nil
}
