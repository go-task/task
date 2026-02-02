package taskfile

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	*baseNode
	url    *url.URL     // stores url pointing actual remote file. (e.g. with Taskfile.yml)
	client *http.Client // HTTP client with optional TLS configuration
}

// buildHTTPClient creates an HTTP client with optional TLS configuration.
// If no certificate options are provided, it returns http.DefaultClient.
func buildHTTPClient(insecure bool, caCert, cert, certKey string) (*http.Client, error) {
	// Validate that cert and certKey are provided together
	if (cert != "" && certKey == "") || (cert == "" && certKey != "") {
		return nil, fmt.Errorf("both --cert and --cert-key must be provided together")
	}

	// If no TLS customization is needed, return the default client
	if !insecure && caCert == "" && cert == "" {
		return http.DefaultClient, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}

	// Load custom CA certificate if provided
	if caCert != "" {
		caCertData, err := os.ReadFile(caCert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertData) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if provided
	if cert != "" && certKey != "" {
		clientCert, err := tls.LoadX509KeyPair(cert, certKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
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

	client, err := buildHTTPClient(insecure, base.caCert, base.cert, base.certKey)
	if err != nil {
		return nil, err
	}

	return &HTTPNode{
		baseNode: base,
		url:      url,
		client:   client,
	}, nil
}

func (node *HTTPNode) Location() string {
	return node.url.Redacted()
}

func (node *HTTPNode) Read() ([]byte, error) {
	return node.ReadContext(context.Background())
}

func (node *HTTPNode) ReadContext(ctx context.Context) ([]byte, error) {
	url, err := RemoteExists(ctx, *node.url, node.client)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, errors.TaskfileFetchFailedError{URI: node.Location()}
	}

	resp, err := node.client.Do(req.WithContext(ctx))
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
	if len(dir) == 0 {
		// Resolve to the current node.Dir().
		return node.Dir(), nil
	} else {
		// Resolve include.Dir, relative to this node.Dir(), or absolute.
		dir, err := execext.ExpandLiteral(dir)
		if err != nil {
			return "", err
		}
		return filepathext.SmartJoin(node.Dir(), dir), nil
	}
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
