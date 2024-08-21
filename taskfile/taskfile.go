package taskfile

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sysinfo"
)

var (
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
	allowedContentTypes = []string{
		"text/plain",
		"text/yaml",
		"text/x-yaml",
		"application/yaml",
		"application/x-yaml",
		"application/zip",
	}
)

// RemoteExists will check if a file at the given URL Exists. If it does, it
// will return its URL. If it does not, it will search the search for any files
// at the given URL with any of the default Taskfile files names. If any of
// these match a file, the first matching path will be returned. If no files are
// found, an error will be returned.
func RemoteExists(ctx context.Context, l *logger.Logger, u *url.URL) (*url.URL, error) {
	// Create a new HEAD request for the given URL to check if the resource exists
	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: u.String()}
	}

	// Request the given URL
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: u.String()}
	}
	defer resp.Body.Close()

	// If the request was successful and the content type is allowed, return the
	// URL The content type check is to avoid downloading files that are not
	// Taskfiles It means we can try other files instead of downloading
	// something that is definitely not a Taskfile
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode == http.StatusOK && slices.ContainsFunc(allowedContentTypes, func(s string) bool {
		return strings.Contains(contentType, s)
	}) {
		return u, nil
	}

	// If the request was not successful, append the default Taskfile names to
	// the URL and return the URL of the first successful request
	for _, taskfile := range defaultTaskfiles {
		// Fixes a bug with JoinPath where a leading slash is not added to the
		// path if it is empty
		if u.Path == "" {
			u.Path = "/"
		}
		alt := u.JoinPath(taskfile)
		req.URL = alt

		// Try the alternative URL
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.TaskfileFetchFailedError{URI: u.String()}
		}
		defer resp.Body.Close()

		// If the request was successful, return the URL
		if resp.StatusCode == http.StatusOK {
			l.VerboseOutf(logger.Magenta, "task: [%s] Not found - Using alternative (%s)\n", alt.String(), taskfile)
			return alt, nil
		}
	}

	return nil, errors.TaskfileNotFoundError{URI: u.String(), Walk: false}
}

// Exists will check if a file at the given path Exists. If it does, it will
// return the path to it. If it does not, it will search for any files at the
// given path with any of the default Taskfile files names. If any of these
// match a file, the first matching path will be returned. If no files are
// found, an error will be returned.
func Exists(l *logger.Logger, path string) (string, error) {
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

	for _, taskfile := range defaultTaskfiles {
		alt := filepathext.SmartJoin(path, taskfile)
		if _, err := os.Stat(alt); err == nil {
			l.VerboseOutf(logger.Magenta, "task: [%s] Not found - Using alternative (%s)\n", path, taskfile)
			return filepath.Abs(alt)
		}
	}

	return "", errors.TaskfileNotFoundError{URI: path, Walk: false}
}

// ExistsWalk will check if a file at the given path exists by calling the
// exists function. If a file is not found, it will walk up the directory tree
// calling the exists function until it finds a file or reaches the root
// directory. On supported operating systems, it will also check if the user ID
// of the directory changes and abort if it does.
func ExistsWalk(l *logger.Logger, path string) (string, error) {
	origPath := path
	owner, err := sysinfo.Owner(path)
	if err != nil {
		return "", err
	}
	for {
		fpath, err := Exists(l, path)
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
