package taskfile

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/templater"
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

	// Apply headers if configured for this host
	// Security: Headers are matched by exact hostname (including port) from node.url.Host
	// to prevent credentials from being sent to unintended hosts.
	if node.headers != nil {
		host := node.url.Host
		if hostHeaders, ok := node.headers[host]; ok {
			cache := &templater.Cache{Vars: env.GetEnviron()}
			for key, value := range hostHeaders {
				// Security: Prevent overriding critical headers that could cause protocol
				// violations or security issues (e.g., request smuggling via Transfer-Encoding,
				// routing issues via Host header manipulation, response header spoofing).
				canonicalKey := strings.ToLower(strings.TrimSpace(key))
				switch canonicalKey {
				case "host", "content-length", "transfer-encoding", "trailer", "connection", "upgrade", "location", "set-cookie":
					// Warn and skip critical headers that should be managed by the HTTP library
					log.Printf("Warning: skipping forbidden header %q for host %q\n", key, host)
					continue
				}

				// Expand environment variables in the header value
				expandedValue := templater.Replace(value, cache)

				// Security: Prevent HTTP header injection by rejecting values with control
				// characters. Go's net/http also validates these, but we check explicitly.
				if strings.ContainsAny(expandedValue, "\r\n\x00") {
					log.Printf("Warning: skipping header %q for host %q due to invalid characters in value\n", key, host)
					continue
				}

				req.Header.Set(key, expandedValue)
			}
		}
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
