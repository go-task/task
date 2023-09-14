package read

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/internal/filepathext"
)

// A FileNode is a node that reads a taskfile from the local filesystem.
type FileNode struct {
	*BaseNode
	Dir        string
	Entrypoint string
}

func NewFileNode(uri string, opts ...NodeOption) (*FileNode, error) {
	base := NewBaseNode(opts...)
	if uri == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		uri = d
	}
	path, err := Exists(uri)
	if err != nil {
		return nil, err
	}
	return &FileNode{
		BaseNode:   base,
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

func (node *FileNode) Read(ctx context.Context) ([]byte, error) {
	f, err := os.Open(node.Location())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
