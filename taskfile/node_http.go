package taskfile

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	*BaseNode
	URL *url.URL
}

func NewHTTPNode(l *logger.Logger, entrypoint, dir string, insecure bool, opts ...NodeOption) (*HTTPNode, error) {
	base := NewBaseNode(opts...)
	base.dir = dir
	url, err := url.Parse(entrypoint)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: entrypoint}
	}
	url, err = RemoteExists(l, url)
	if err != nil {
		return nil, err
	}
	return &HTTPNode{
		BaseNode: base,
		URL:      url,
	}, nil
}

func (node *HTTPNode) Location() string {
	return node.URL.String()
}

func (node *HTTPNode) Remote() bool {
	return true
}

func (node *HTTPNode) Read(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequest("GET", node.URL.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: node.URL.String()}
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: node.URL.String()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.TaskfileFetchFailedError{
			URI:            node.URL.String(),
			HTTPStatusCode: resp.StatusCode,
		}
	}

	// Read the entire response body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (node *HTTPNode) ResolveIncludeEntrypoint(include ast.Include) (string, error) {
	ref, err := url.Parse(include.Taskfile)
	if err != nil {
		return "", err
	}
	return node.URL.ResolveReference(ref).String(), nil
}

func (node *HTTPNode) ResolveIncludeDir(include ast.Include) (string, error) {
	path, err := execext.Expand(include.Dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	entrypointDir := filepath.Dir(node.Dir())
	return filepathext.SmartJoin(entrypointDir, path), nil
}
