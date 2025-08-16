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

// ResolveDir returns an absolute path to the directory that the task should be
// run in. If the entrypoint and dir are BOTH set, then the Taskfile will not
// sit inside the directory specified by dir and we should ensure that the dir
// is absolute. Otherwise, the dir will always be the parent directory of the
// resolved entrypoint, so we should return that parent directory.
func ResolveDir(entrypoint, resolvedEntrypoint, dir string) (string, error) {
	if entrypoint != "" && dir != "" {
		return filepath.Abs(dir)
	}
	return filepath.Dir(resolvedEntrypoint), nil
}

// Search looks for files with the given possible filenames using the given
// entrypoint and directory. If the entrypoint is set, it checks if the
// entrypoint matches a file or if it matches a directory containing one of the
// possible filenames. Otherwise, it walks up the file tree starting at the
// given directory and performs a search in each directory for the possible
// filenames until it finds a match or reaches the root directory. If the
// entrypoint and directory are both empty, it defaults the directory to the
// current working directory and performs a recursive search starting there. If
// a match is found, the absolute path to the file is returned with its
// directory. If no match is found, an error is returned.
func Search(entrypoint, dir string, possibleFilenames []string) (string, error) {
	var err error
	if entrypoint != "" {
		entrypoint, err = SearchPath(entrypoint, possibleFilenames)
		if err != nil {
			return "", err
		}
		return entrypoint, nil
	}
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	entrypoint, err = SearchPathRecursively(dir, possibleFilenames)
	if err != nil {
		return "", err
	}
	return entrypoint, nil
}

// SearchAll looks for files with the given possible filenames using the given
// entrypoint and directory. If the entrypoint is set, it checks if the
// entrypoint matches a file or if it matches a directory containing one of the
// possible filenames and add it to a list of matches. It then walks up the file
// tree starting at the given directory and performs a search in each directory
// for the possible filenames until it finds a match or reaches the root
// directory. If the entrypoint and directory are both empty, it defaults the
// directory to the current working directory and performs a recursive search
// starting there. If matches are found, the absolute path to each file is added
// to the list and returned.
func SearchAll(entrypoint, dir string, possibleFilenames []string) ([]string, error) {
	var err error
	var entrypoints []string
	if entrypoint != "" {
		entrypoint, err = SearchPath(entrypoint, possibleFilenames)
		if err != nil {
			return nil, err
		}
		entrypoints = append(entrypoints, entrypoint)
	}
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	paths, err := SearchNPathRecursively(dir, possibleFilenames, -1)
	if err != nil {
		return nil, err
	}
	return append(entrypoints, paths...), nil
}

// SearchPath will check if a file at the given path exists or not. If it does,
// it will return the path to it. If it does not, it will search for any files
// at the given path with any of the given possible names. If any of these match
// a file, the first matching path will be returned. If no files are found, an
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

// SearchPathRecursively walks up the directory tree starting at the given
// path, calling the Search function in each directory until it finds a matching
// file or reaches the root directory. On supported operating systems, it will
// also check if the user ID of the directory changes and abort if it does.
func SearchPathRecursively(path string, possibleFilenames []string) (string, error) {
	paths, err := SearchNPathRecursively(path, possibleFilenames, 1)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", os.ErrNotExist
	}
	return paths[0], nil
}

// SearchNPathRecursively walks up the directory tree starting at the given
// path, calling the Search function in each directory and adding each matching
// file that it finds to a list until it reaches the root directory or the
// length of the list exceeds n. On supported operating systems, it will also
// check if the user ID of the directory changes and abort if it does.
func SearchNPathRecursively(path string, possibleFilenames []string, n int) ([]string, error) {
	var paths []string

	owner, err := sysinfo.Owner(path)
	if err != nil {
		return nil, err
	}

	for n == -1 || len(paths) < n {
		fpath, err := SearchPath(path, possibleFilenames)
		if err == nil {
			paths = append(paths, fpath)
		}

		// Get the parent path/user id
		parentPath := filepath.Dir(path)
		parentOwner, err := sysinfo.Owner(parentPath)
		if err != nil {
			return nil, err
		}

		// Error if we reached the root directory and still haven't found a file
		// OR if the user id of the directory changes
		if path == parentPath || (parentOwner != owner) {
			return paths, nil
		}

		owner = parentOwner
		path = parentPath
	}

	return paths, nil
}
