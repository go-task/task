package taskrc

import (
	"os"
	"path/filepath"
)

type Node struct {
	entrypoint string
	dir        string
}

func NewNode(
	entrypoint string,
	dir string,
) (*Node, error) {
	dir = getDefaultDir(entrypoint, dir)
	var err error
	entrypoint, dir, err = resolveFileNodeEntrypointAndDir(entrypoint, dir)
	if err != nil {
		return nil, err
	}
	return &Node{
		entrypoint: entrypoint,
		dir:        dir,
	}, nil
}

// TODO: Merge with the taskfile functions

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
