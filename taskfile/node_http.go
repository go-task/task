package taskfile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	*baseNode
	url *url.URL // stores url pointing actual remote file. (e.g. with Taskfile.yml)
}

func NewHTTPNode(
	entrypoint string,
	dir string,
	insecure bool,
	opts ...NodeOption,
) (*HTTPNode, error) {
	base := NewBaseNode(dir, opts...)
	url, err := url.Parse(entrypoint)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: url.Redacted()}
	}
	return &HTTPNode{
		baseNode: base,
		url:      url,
	}, nil
}

func (node *HTTPNode) Location() string {
	return node.url.Redacted()
}

func (node *HTTPNode) Read() ([]byte, error) {
	return node.ReadContext(context.Background())
}

func (node *HTTPNode) ReadContext(ctx context.Context) ([]byte, error) {
	url, err := RemoteExists(ctx, *node.url)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: node.Location()}
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		return nil, errors.TaskfileFetchFailedError{URI: node.Location()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.TaskfileFetchFailedError{
			URI:            node.Location(),
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
	return node.url.ResolveReference(ref).String(), nil
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

func (node *HTTPNode) CacheKey() string {
	checksum := strings.TrimRight(checksum([]byte(node.Location())), "=")
	dir, filename := filepath.Split(node.url.Path)
	lastDir := filepath.Base(dir)
	prefix := filename
	// Means it's not "", nor "." nor "/", so it's a valid directory
	if len(lastDir) > 1 {
		prefix = fmt.Sprintf("%s.%s", lastDir, filename)
	}
	return fmt.Sprintf("http.%s.%s.%s", node.url.Host, prefix, checksum)
}
