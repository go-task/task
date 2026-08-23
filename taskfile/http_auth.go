package taskfile

import (
	"cmp"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"golang.org/x/net/http/httpguts"

	"github.com/go-task/task/v3/internal/templater"
)

// HostHeaders maps a host to the HTTP headers to send when fetching a remote
// Taskfile from it. Values are templated, but no variables are available.
type HostHeaders map[string]map[string]string

type authTransport struct {
	base    http.RoundTripper
	host    string
	headers map[string]string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Re-checked per request: a redirect goes through this same transport, and
	// Go only strips Authorization, WWW-Authenticate and Cookie on its own.
	if !hostMatches(t.host, req.URL.Host) {
		return t.base.RoundTrip(req)
	}
	req = req.Clone(req.Context())
	for name, value := range t.headers {
		req.Header.Set(name, value)
	}
	return t.base.RoundTrip(req)
}

// authenticatedClient resolves on each read, not at build time, so a cached
// run needs no credentials.
func (node *HTTPNode) authenticatedClient() (*http.Client, error) {
	headers, err := resolveAuthHeaders(node.authHeaders, node.url.Host)
	if err != nil {
		return nil, err
	}
	if len(headers) == 0 {
		return node.client, nil
	}
	return withAuthHeaders(node.client, node.url.Host, headers), nil
}

// withAuthHeaders copies rather than mutates: buildHTTPClient returns the
// shared http.DefaultClient when no TLS option is set.
func withAuthHeaders(client *http.Client, host string, headers map[string]string) *http.Client {
	authenticated := *client
	authenticated.Transport = &authTransport{
		base:    cmp.Or(client.Transport, http.DefaultTransport),
		host:    host,
		headers: headers,
	}
	return &authenticated
}

// resolveAuthHeaders returns the expanded headers for host, or nil if none.
func resolveAuthHeaders(hostHeaders HostHeaders, host string) (map[string]string, error) {
	var headers map[string]string
	for pattern, patternHeaders := range hostHeaders {
		if hostMatches(pattern, host) {
			headers = patternHeaders
			break
		}
	}
	if len(headers) == 0 {
		return nil, nil
	}

	cache := &templater.Cache{}
	resolved := make(map[string]string, len(headers))
	for _, name := range slices.Sorted(maps.Keys(headers)) {
		if err := validateHeaderName(name); err != nil {
			return nil, fmt.Errorf(`remote auth for host %q: %w`, host, err)
		}
		resolved[name] = templater.Replace(headers[name], cache)
	}
	if err := cache.Err(); err != nil {
		return nil, fmt.Errorf(`remote auth for host %q: %w`, host, err)
	}
	return resolved, nil
}

// validateHeaderName names the offending header; ReadContext discards the
// transport's own error.
func validateHeaderName(name string) error {
	if !httpguts.ValidHeaderFieldName(name) {
		return fmt.Errorf("invalid header name %q", name)
	}
	return nil
}

// hostMatches compares exactly, port included, as trusted hosts do.
func hostMatches(pattern, host string) bool {
	return pattern == host
}
