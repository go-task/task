package read

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/internal/filepathext"
)

// A FileNode is a node that reads a taskfile from the local filesystem.
type FileNode struct {
	BaseNode
	Dir        string
	Entrypoint string
}

func NewFileNode(parent Node, path string, optional bool) (*FileNode, error) {
	path, err := exists(path)
	if err != nil {
		return nil, err
	}

	return &FileNode{
		BaseNode: BaseNode{
			parent:   parent,
			optional: optional,
		},
		Dir:        filepath.Dir(path),
		Entrypoint: filepath.Base(path),
	}, nil
}

func (node *FileNode) Location() string {
	return filepathext.SmartJoin(node.Dir, node.Entrypoint)
}

func (node *FileNode) Remote() bool {
	return false
}

func (node *FileNode) Read() ([]byte, error) {
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

	return io.ReadAll(f)
}
