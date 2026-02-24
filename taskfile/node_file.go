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
	// Find the entrypoint file.
	resolvedEntrypoint, err := fsext.Search(entrypoint, dir, DefaultTaskfiles)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if entrypoint == "" {
				return nil, errors.TaskfileNotFoundError{URI: entrypoint, Walk: true}
			} else {
				return nil, errors.TaskfileNotFoundError{URI: entrypoint, Walk: false}
			}
		} else if errors.Is(err, os.ErrPermission) {
			return nil, errors.TaskfileNotFoundError{URI: entrypoint, Walk: true, OwnerChange: true}
		}
		return nil, err
	}

	// Resolve the directory.
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
	// Resolve to entrypoint without adjustment.
	if isRemoteEntrypoint(entrypoint) {
		return entrypoint, nil
	}
	// Resolve relative to this nodes Taskfile location, or absolute.
	entrypoint, err := execext.ExpandLiteral(entrypoint)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(node.Location())
	return filepathext.SmartJoin(dir, entrypoint), nil
}

func (node *FileNode) ResolveDir(dir string) (string, error) {
	if len(dir) == 0 {
		// Resolve to the current node.Dir().
		return node.Dir(), nil
	} else {
		// Resolve include.Dir, relative to this node.Dir(), or absolute.
		dir, err := execext.ExpandLiteral(dir)
		if err != nil {
			return "", err
		}
		return filepathext.SmartJoin(node.Dir(), dir), nil
	}
}
