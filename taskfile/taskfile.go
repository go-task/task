package taskfile

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/go-task/task/v3/errors"
)

var (
	// DefaultTaskfiles is the list of Taskfile file names supported by default.
	DefaultTaskfiles = []string{
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
		"application/octet-stream",
	}
)

// RemoteExists will check if a file at the given URL Exists. If it does, it
// will return its URL. If it does not, it will search the search for any files
// at the given URL with any of the default Taskfile files names. If any of
// these match a file, the first matching path will be returned. If no files are
// found, an error will be returned.
func RemoteExists(ctx context.Context, u url.URL) (*url.URL, error) {
	// Create a new HEAD request for the given URL to check if the resource exists
	req, err := http.NewRequestWithContext(ctx, "HEAD", u.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: u.Redacted()}
	}

	// Request the given URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("checking remote file: %w", ctx.Err())
		}
		return nil, errors.TaskfileFetchFailedError{URI: u.Redacted()}
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
		return &u, nil
	}

	// If the request was not successful, append the default Taskfile names to
	// the URL and return the URL of the first successful request
	for _, taskfile := range DefaultTaskfiles {
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
			return nil, errors.TaskfileFetchFailedError{URI: u.Redacted()}
		}
		defer resp.Body.Close()

		// If the request was successful, return the URL
		if resp.StatusCode == http.StatusOK {
			return alt, nil
		}
	}

	return nil, errors.TaskfileNotFoundError{URI: u.Redacted(), Walk: false}
}
