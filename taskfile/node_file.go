package taskfile

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// A FileNode is a node that reads a taskfile from the local filesystem.
type FileNode struct {
	*BaseNode
	Entrypoint string
}

func NewFileNode(entrypoint, dir string, opts ...NodeOption) (*FileNode, error) {
	var err error
	base := NewBaseNode(dir, opts...)
	entrypoint, base.dir, err = resolveFileNodeEntrypointAndDir(entrypoint, base.dir)
	if err != nil {
		return nil, err
	}
	return &FileNode{
		BaseNode:   base,
		Entrypoint: entrypoint,
	}, nil
}

func (node *FileNode) Location() string {
	return node.Entrypoint
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

// resolveFileNodeEntrypointAndDir resolves checks the values of entrypoint and dir and
// populates them with default values if necessary.
func resolveFileNodeEntrypointAndDir(entrypoint, dir string) (string, string, error) {
	var err error
	if entrypoint != "" {
		entrypoint, err = Exists(entrypoint)
		if err != nil {
			return "", "", err
		}
		if dir == "" {
			dir = filepath.Dir(entrypoint)
		}
		return entrypoint, dir, nil
	}
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	entrypoint, err = ExistsWalk(dir)
	if err != nil {
		return "", "", err
	}
	dir = filepath.Dir(entrypoint)
	return entrypoint, dir, nil
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
	entrypointDir := filepath.Dir(node.Entrypoint)
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
	entrypointDir := filepath.Dir(node.Entrypoint)
	return filepathext.SmartJoin(entrypointDir, path), nil
}

func (node *FileNode) FilenameAndLastDir() (string, string) {
	return "", filepath.Base(node.Entrypoint)
}
