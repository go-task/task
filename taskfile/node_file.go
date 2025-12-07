package taskfile

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/errors"
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
	// Find the entrypoint file
	resolvedEntrypoint, err := fsext.Search(entrypoint, dir, DefaultTaskfiles)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.TaskfileNotFoundError{URI: entrypoint, Walk: false}
		}
		return nil, err
	}

	// Resolve the directory
	resolvedDir, err := fsext.ResolveDir(entrypoint, resolvedEntrypoint, dir)
	if err != nil {
		return nil, err
	}

	return &FileNode{
		baseNode:   NewBaseNode(resolvedDir, opts...),
		entrypoint: resolvedEntrypoint,
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
	if isRemoteEntrypoint(entrypoint) {
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
