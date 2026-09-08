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

// HeadersByHost configures HTTP headers per host for remote Taskfiles.
// Values support template functions, but not Taskfile variables.
type HeadersByHost map[string]map[string]string

type headersTransport struct {
	base    http.RoundTripper
	host    string
	headers map[string]string
}

func (t *headersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Scope all configured headers to this host, including after redirects.
	if !hostMatches(t.host, req.URL.Host) {
		return t.base.RoundTrip(req)
	}
	req = req.Clone(req.Context())
	for name, value := range t.headers {
		req.Header.Set(name, value)
	}
	return t.base.RoundTrip(req)
}

// Resolve headers only when downloading, so cached runs need no credentials.
func (node *HTTPNode) clientWithHeaders() (*http.Client, error) {
	headers, err := resolveHeaders(node.headersByHost, node.url.Host)
	if err != nil {
		return nil, err
	}
	if len(headers) == 0 {
		return node.client, nil
	}
	return withHeaders(node.client, node.url.Host, headers), nil
}

func withHeaders(client *http.Client, host string, headers map[string]string) *http.Client {
	// The client may be http.DefaultClient; leave it unchanged.
	configured := *client
	configured.Transport = &headersTransport{
		base:    cmp.Or(client.Transport, http.DefaultTransport),
		host:    host,
		headers: headers,
	}
	return &configured
}

func resolveHeaders(headersByHost HeadersByHost, host string) (map[string]string, error) {
	var headers map[string]string
	for pattern, patternHeaders := range headersByHost {
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
			return nil, fmt.Errorf(`remote headers for host %q: %w`, host, err)
		}
		resolved[name] = templater.Replace(headers[name], cache)
	}
	if err := cache.Err(); err != nil {
		return nil, fmt.Errorf(`remote headers for host %q: %w`, host, err)
	}
	return resolved, nil
}

// Validate names here because ReadContext hides transport errors.
func validateHeaderName(name string) error {
	if !httpguts.ValidHeaderFieldName(name) {
		return fmt.Errorf("invalid header name %q", name)
	}
	return nil
}

// Match the host and port exactly for both headers and trusted hosts.
func hostMatches(pattern, host string) bool {
	return pattern == host
}
