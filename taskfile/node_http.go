package taskfile

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	*BaseNode
	URL        *url.URL // stores url pointing actual remote file. (e.g. with Taskfile.yml)
	entrypoint string   // stores entrypoint url. used for building graph vertices.
	timeout    time.Duration
}

func NewHTTPNode(
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
	opts ...NodeOption,
) (*HTTPNode, error) {
	base := NewBaseNode(dir, opts...)
	url, err := url.Parse(entrypoint)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: entrypoint}
	}

	return &HTTPNode{
		BaseNode:   base,
		URL:        url,
		entrypoint: entrypoint,
		timeout:    timeout,
	}, nil
}

func (node *HTTPNode) Location() string {
	return node.entrypoint
}

func (node *HTTPNode) Remote() bool {
	return true
}

func (node *HTTPNode) Read(ctx context.Context) ([]byte, error) {
	url, err := RemoteExists(ctx, node.URL, node.timeout)
	if err != nil {
		return nil, err
	}
	node.URL = url
	req, err := http.NewRequest("GET", node.URL.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: node.URL.String()}
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &errors.TaskfileNetworkTimeoutError{URI: node.URL.String(), Timeout: node.timeout}
		}
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

func (node *HTTPNode) ResolveEntrypoint(entrypoint string) (string, error) {
	ref, err := url.Parse(entrypoint)
	if err != nil {
		return "", err
	}
	return node.URL.ResolveReference(ref).String(), nil
}

func (node *HTTPNode) ResolveDir(dir string) (string, error) {
	path, err := execext.ExpandLiteral(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	parent := node.Dir()
	if node.Parent() != nil {
		parent = node.Parent().Dir()
	}

	return filepathext.SmartJoin(parent, path), nil
}

func (node *HTTPNode) FilenameAndLastDir() (string, string) {
	dir, filename := filepath.Split(node.entrypoint)
	return filepath.Base(dir), filename
}
