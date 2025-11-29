package taskfile

import (
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// A ByteNode is a node that reads a taskfile direct from a []byte object.
type ByteNode struct {
	*baseNode
	data []byte
}

func NewByteNode(data []byte, dir string) (*ByteNode, error) {
	return &ByteNode{
		baseNode: NewBaseNode(dir),
		data:     data,
	}, nil
}

func (node *ByteNode) Location() string {
	return "__bytes__"
}

func (node *ByteNode) Remote() bool {
	return true
}

func (node *ByteNode) Read() ([]byte, error) {
	return node.data, nil
}

func (node *ByteNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// A ByteNode has no presence on the local file system.
	return entrypoint, nil
}

func (node *ByteNode) ResolveDir(dir string) (string, error) {
	path, err := execext.ExpandLiteral(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	return filepathext.SmartJoin(node.Dir(), path), nil
}
