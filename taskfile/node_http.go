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
func buildHTTPClient(insecure bool, caCert, cert, certKey, certKeyPass string) (*http.Client, error) {
	// Validate that cert and certKey are provided together
	if (cert != "" && certKey == "") || (cert == "" && certKey != "") {
		return nil, fmt.Errorf("both --cert and --cert-key must be provided together")
	}

	// If no TLS customization is needed, return the default client
	if !insecure && caCert == "" && cert == "" {
		return http.DefaultClient, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure, //nolint:gosec
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
		var clientCert tls.Certificate
		var err error

		if certKeyPass != "" {
			// Load encrypted private key
			clientCert, err = loadCertWithEncryptedKey(cert, certKey, certKeyPass)
		} else {
			clientCert, err = tls.LoadX509KeyPair(cert, certKey)
		}
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

// loadCertWithEncryptedKey loads a certificate with an encrypted private key.
func loadCertWithEncryptedKey(certFile, keyFile, passphrase string) (tls.Certificate, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read key file: %w", err)
	}

	// Try to decrypt the private key
	decryptedKey, err := decryptPEMKey(keyPEM, passphrase)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return tls.X509KeyPair(certPEM, decryptedKey)
}

// decryptPEMKey attempts to decrypt a PEM-encoded private key.
func decryptPEMKey(keyPEM []byte, passphrase string) ([]byte, error) {
	// For PKCS#8 encrypted keys, we need to parse and decrypt them
	// The standard library doesn't directly support encrypted PKCS#8,
	// so we try to parse it as-is first (in case it's not actually encrypted)
	// For now, we support unencrypted keys and return an error for encrypted ones
	// that require external libraries to decrypt.

	// Try to parse as unencrypted first
	_, err := tls.X509KeyPair([]byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"), keyPEM)
	if err == nil {
		return keyPEM, nil
	}

	// TODO: Add support for encrypted PKCS#8 keys using x/crypto/pkcs8
	// This would require adding a dependency on golang.org/x/crypto
	return nil, fmt.Errorf("encrypted private keys require the key to be decrypted externally, or use an unencrypted key")
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

	// Build HTTP client with TLS configuration from node options
	client, err := buildHTTPClient(insecure, base.caCert, base.cert, base.certKey, base.certKeyPass)
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
