package fsext

import (
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/sysinfo"
)

// DefaultDir will return the default directory given an entrypoint or
// directory. If the directory is set, it will ensure it is an absolute path and
// return it. If the entrypoint is set, but the directory is not, it will leave
// the directory blank. If both are empty, it will default the directory to the
// current working directory.
func DefaultDir(entrypoint, dir string) string {
	// If the directory is set, ensure it is an absolute path
	if dir != "" {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return ""
		}
		return dir
	}

	// If the entrypoint and dir are empty, we default the directory to the current working directory
	if entrypoint == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		return wd
	}

	// If the entrypoint is set, but the directory is not, we leave the directory blank
	return ""
}

// Search will look for files with the given possible filenames using the given
// entrypoint and directory. If the entrypoint is set, it will check if the
// entrypoint matches a file or if it matches a directory containing one of the
// possible filenames. Otherwise, it will walk up the file tree starting at the
// given directory and perform a search in each directory for the possible
// filenames until it finds a match or reaches the root directory. If the
// entrypoint and directory are both empty, it will default the directory to the
// current working directory and perform a recursive search starting there. If a
// match is found, the absolute path to the file will be returned with its
// directory. If no match is found, an error will be returned.
func Search(entrypoint, dir string, possibleFilenames []string) (string, string, error) {
	var err error
	if entrypoint != "" {
		entrypoint, err = SearchPath(entrypoint, possibleFilenames)
		if err != nil {
			return "", "", err
		}
		if dir == "" {
			dir = filepath.Dir(entrypoint)
		} else {
			dir, err = filepath.Abs(dir)
			if err != nil {
				return "", "", err
			}
		}
		return entrypoint, dir, nil
	}
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	entrypoint, err = SearchPathRecursively(dir, possibleFilenames)
	if err != nil {
		return "", "", err
	}
	dir = filepath.Dir(entrypoint)
	return entrypoint, dir, nil
}

// Search will check if a file at the given path exists or not. If it does, it
// will return the path to it. If it does not, it will search for any files at
// the given path with any of the given possible names. If any of these match a
// file, the first matching path will be returned. If no files are found, an
// error will be returned.
func SearchPath(path string, possibleFilenames []string) (string, error) {
	// Get file info about the path
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// If the path exists and is a regular file, device, symlink, or named pipe,
	// return the absolute path to it
	if fi.Mode().IsRegular() ||
		fi.Mode()&os.ModeDevice != 0 ||
		fi.Mode()&os.ModeSymlink != 0 ||
		fi.Mode()&os.ModeNamedPipe != 0 {
		return filepath.Abs(path)
	}

	// If the path is a directory, check if any of the possible names exist
	// in that directory
	for _, filename := range possibleFilenames {
		alt := filepathext.SmartJoin(path, filename)
		if _, err := os.Stat(alt); err == nil {
			return filepath.Abs(alt)
		}
	}

	return "", os.ErrNotExist
}

// SearchRecursively will check if a file at the given path exists by calling
// the exists function. If a file is not found, it will walk up the directory
// tree calling the Search function until it finds a file or reaches the root
// directory. On supported operating systems, it will also check if the user ID
// of the directory changes and abort if it does.
func SearchPathRecursively(path string, possibleFilenames []string) (string, error) {
	owner, err := sysinfo.Owner(path)
	if err != nil {
		return "", err
	}
	for {
		fpath, err := SearchPath(path, possibleFilenames)
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
			return "", os.ErrNotExist
		}

		owner = parentOwner
		path = parentPath
	}
}
