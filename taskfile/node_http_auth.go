package taskfile

import (
	"cmp"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
)

// HostHeaders maps a host to the HTTP headers to send when fetching a remote
// Taskfile from it. Values may reference environment variables using the
// `${VAR}` or `$VAR` syntax.
type HostHeaders map[string]map[string]string

// authTransport adds the configured headers to every request made to host.
type authTransport struct {
	base    http.RoundTripper
	host    string
	headers map[string]string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The headers are scoped to a single host. Checking here rather than once
	// at build time is what keeps a redirect from carrying the credentials
	// somewhere else: the client sends the redirected request through this same
	// transport, and Go only strips Authorization, WWW-Authenticate and Cookie
	// on its own.
	if !hostMatches(t.host, req.URL.Host) {
		return t.base.RoundTrip(req)
	}
	// A RoundTripper must not modify the request it is given.
	req = req.Clone(req.Context())
	for name, value := range t.headers {
		req.Header.Set(name, value)
	}
	return t.base.RoundTrip(req)
}

// authenticatedClient returns the node's client, wrapped so that it sends the
// configured headers. The environment variables the headers reference are read
// here rather than when the node is built, so that a run served from the cache
// does not require credentials it will never send.
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

// withAuthHeaders returns a copy of client that sends headers to host. The
// client is copied rather than mutated because buildHTTPClient returns the
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

// resolveAuthHeaders returns the headers configured for host, with their
// environment variable references expanded. It returns nil when no entry
// matches, leaving the request unauthenticated.
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

	resolved := make(map[string]string, len(headers))
	for _, name := range slices.Sorted(maps.Keys(headers)) {
		if err := validateHeaderName(name); err != nil {
			return nil, fmt.Errorf(`remote auth for host %q: %w`, host, err)
		}
		value, err := expandEnv(headers[name])
		if err != nil {
			return nil, fmt.Errorf(`remote auth for host %q: header %q: %w`, host, name, err)
		}
		resolved[name] = value
	}
	return resolved, nil
}

// expandEnv replaces ${VAR} and $VAR references with the value of the
// environment variable. An undefined variable is an error rather than an empty
// header, which would only surface later as an opaque 401. A literal dollar
// sign is written `$$`.
func expandEnv(value string) (string, error) {
	var missing []string
	expanded := os.Expand(value, func(name string) string {
		if name == "$" {
			return "$"
		}
		v, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return ""
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("environment variable $%s is not set", strings.Join(missing, ", $"))
	}
	return expanded, nil
}

// validateHeaderName rejects names that http.Header.Set would silently accept
// but the transport would later refuse, so that the error names the offending
// header instead of the request.
func validateHeaderName(name string) error {
	if name == "" {
		return fmt.Errorf("header name cannot be empty")
	}
	if strings.ContainsFunc(name, func(r rune) bool {
		return r <= ' ' || r == ':' || r == 0x7f
	}) {
		return fmt.Errorf("header name %q contains invalid characters", name)
	}
	return nil
}

// hostMatches reports whether a host matches a configured pattern. The
// comparison is exact and includes the port, as it does for trusted hosts.
func hostMatches(pattern, host string) bool {
	return pattern == host
}
