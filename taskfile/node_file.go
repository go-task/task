package taskfile

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fsext"
)

// A FileNode is a node that reads a taskfile from the local filesystem.
type FileNode struct {
	*baseNode
	entrypoint string
}

func NewFileNode(entrypoint, dir string, opts ...NodeOption) (*FileNode, error) {
	var err error
	base := NewBaseNode(dir, opts...)
	entrypoint, base.dir, err = fsext.Search(entrypoint, base.dir, defaultTaskfiles)
	if err != nil {
		return nil, err
	}
	return &FileNode{
		baseNode:   base,
		entrypoint: entrypoint,
	}, nil
}

func (node *FileNode) Location() string {
	return node.entrypoint
}

func (node *FileNode) Read() ([]byte, error) {
	f, err := os.Open(node.Location())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (node *FileNode) ResolveEntrypoint(entrypoint string) (string, error) {
	// If the file is remote, we don't need to resolve the path
	if strings.Contains(entrypoint, "://") {
		return entrypoint, nil
	}
	if strings.HasPrefix(entrypoint, "git") {
		return entrypoint, nil
	}

	path, err := execext.ExpandLiteral(entrypoint)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	entrypointDir := filepath.Dir(node.entrypoint)
	return filepathext.SmartJoin(entrypointDir, path), nil
}

func (node *FileNode) ResolveDir(dir string) (string, error) {
	path, err := execext.ExpandLiteral(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	entrypointDir := filepath.Dir(node.entrypoint)
	return filepathext.SmartJoin(entrypointDir, path), nil
}
