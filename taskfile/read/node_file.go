package read

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
)

// A FileNode is a node that reads a taskfile from the local filesystem.
type FileNode struct {
	BaseNode
	Dir        string
	Entrypoint string
}

func NewFileNode(includedTaskfile taskfile.IncludedTaskfile, parent Node) (*FileNode, error) {
	path, err := includedTaskfile.FullTaskfilePath()
	if err != nil {
		return nil, err
	}

	path, err = exists(path)
	if err != nil {
		return nil, err
	}

	return &FileNode{
		BaseNode: BaseNode{
			parent:   parent,
			optional: includedTaskfile.Optional,
		},
		Dir:        filepath.Dir(path),
		Entrypoint: filepath.Base(path),
	}, nil
}

func (node *FileNode) Location() string {
	return filepathext.SmartJoin(node.Dir, node.Entrypoint)
}

func (node *FileNode) Read() (*taskfile.Taskfile, error) {
	if node.Dir == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		node.Dir = d
	}

	path, err := existsWalk(node.Location())
	if err != nil {
		return nil, err
	}
	node.Dir = filepath.Dir(path)
	node.Entrypoint = filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var t taskfile.Taskfile
	if err := yaml.NewDecoder(f).Decode(&t); err != nil {
		return nil, &errors.TaskfileInvalidError{URI: filepathext.TryAbsToRel(path), Err: err}
	}

	t.Location = path
	return &t, nil
}
