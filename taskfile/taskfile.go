package taskfile

import (
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/sysinfo"
)

var (
	// ErrIncludedTaskfilesCantHaveDotenvs is returned when a included Taskfile contains dotenvs
	ErrIncludedTaskfilesCantHaveDotenvs = errors.New("task: Included Taskfiles can't have dotenv declarations. Please, move the dotenv declaration to the main Taskfile")

	defaultTaskfiles = []string{
		"Taskfile.yml",
		"taskfile.yml",
		"Taskfile.yaml",
		"taskfile.yaml",
		"Taskfile.dist.yml",
		"taskfile.dist.yml",
		"Taskfile.dist.yaml",
		"taskfile.dist.yaml",
	}
)

// Exists will check if a file at the given path Exists. If it does, it will
// return the path to it. If it does not, it will search the search for any
// files at the given path with any of the default Taskfile files names. If any
// of these match a file, the first matching path will be returned. If no files
// are found, an error will be returned.
func Exists(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if fi.Mode().IsRegular() ||
		fi.Mode()&os.ModeDevice != 0 ||
		fi.Mode()&os.ModeSymlink != 0 ||
		fi.Mode()&os.ModeNamedPipe != 0 {
		return filepath.Abs(path)
	}

	for _, n := range defaultTaskfiles {
		fpath := filepathext.SmartJoin(path, n)
		if _, err := os.Stat(fpath); err == nil {
			return filepath.Abs(fpath)
		}
	}

	return "", errors.TaskfileNotFoundError{URI: path, Walk: false}
}

// ExistsWalk will check if a file at the given path exists by calling the
// exists function. If a file is not found, it will walk up the directory tree
// calling the exists function until it finds a file or reaches the root
// directory. On supported operating systems, it will also check if the user ID
// of the directory changes and abort if it does.
func ExistsWalk(path string) (string, error) {
	origPath := path
	owner, err := sysinfo.Owner(path)
	if err != nil {
		return "", err
	}
	for {
		fpath, err := Exists(path)
		if err == nil {
			return fpath, nil
		}

		// Get the parent path/user id
		parentPath := filepath.Dir(path)
		parentOwner, err := sysinfo.Owner(parentPath)
		if err != nil {
			return "", err
		}

		// Error if we reached the root directory and still haven't found a file
		// OR if the user id of the directory changes
		if path == parentPath || (parentOwner != owner) {
			return "", errors.TaskfileNotFoundError{URI: origPath, Walk: false}
		}

		owner = parentOwner
		path = parentPath
	}
}
