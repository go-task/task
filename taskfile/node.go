package taskfile

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/logger"
)

type source struct {
	FileContent   []byte
	FileDirectory string
	Filename      string
}

type Node interface {
	Read(ctx context.Context) (*source, error)
	Parent() Node
	Location() string
	Dir() string
	ResolveEntrypoint(entrypoint string) (string, error)
	ResolveDir(dir string) (string, error)
	FilenameAndLastDir() (string, string)
}

func NewRootNode(
	l *logger.Logger,
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
) (Node, error) {
	dir = getDefaultDir(entrypoint, dir)
	// If the entrypoint is "-", we read from stdin
	if entrypoint == "-" {
		return NewStdinNode(dir)
	}
	return NewNode(l, entrypoint, dir, insecure, timeout)
}

func NewNode(
	l *logger.Logger,
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
	opts ...NodeOption,
) (Node, error) {
	remote, supported, err := NewRemoteNode(l, entrypoint, dir, insecure, timeout, opts...)
	if err != nil {
		return nil, err
	}

	if !supported {
		// If no other scheme matches, we assume it's a file
		return NewFileNode(l, entrypoint, dir, opts...)
	}

	if !experiments.RemoteTaskfiles.Enabled {
		return nil, errors.New("task: Remote taskfiles are not enabled. You can read more about this experiment and how to enable it at https://taskfile.dev/experiments/remote-taskfiles")
	}
	return remote, nil
}

func getDefaultDir(entrypoint, dir string) string {
	// If the entrypoint and dir are empty, we default the directory to the current working directory
	if dir == "" {
		if entrypoint == "" {
			wd, err := os.Getwd()
			if err != nil {
				return ""
			}
			dir = wd
		}
		return dir
	}

	// If the directory is set, ensure it is an absolute path
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return ""
	}

	return dir
}
