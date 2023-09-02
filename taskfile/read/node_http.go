package read

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/go-task/task/v3/errors"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	BaseNode
	URL *url.URL
}

func NewHTTPNode(parent Node, urlString string, optional bool) (*HTTPNode, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	return &HTTPNode{
		BaseNode: BaseNode{
			parent:   parent,
			optional: optional,
		},
		URL: url,
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
