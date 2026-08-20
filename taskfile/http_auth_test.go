package taskfile

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAuthHeaders(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	tests := []struct {
		name        string
		hostHeaders HostHeaders
		host        string
		env         map[string]string
		want        map[string]string
		wantErr     string
	}{
		{
			name:        "no configuration",
			hostHeaders: nil,
			host:        "gitlab.com",
		},
		{
			name:        "host does not match",
			hostHeaders: HostHeaders{"gitlab.com": {"PRIVATE-TOKEN": "token"}},
			host:        "example.com",
		},
		{
			name:        "port is part of the host",
			hostHeaders: HostHeaders{"example.com": {"PRIVATE-TOKEN": "token"}},
			host:        "example.com:8080",
		},
		{
			name:        "literal value",
			hostHeaders: HostHeaders{"gitlab.com": {"PRIVATE-TOKEN": "token"}},
			host:        "gitlab.com",
			want:        map[string]string{"PRIVATE-TOKEN": "token"},
		},
		{
			name:        "braced environment variable",
			hostHeaders: HostHeaders{"gitlab.com": {"PRIVATE-TOKEN": "${TASK_TEST_TOKEN}"}}, //nolint:gosec // an env var reference, not a credential
			host:        "gitlab.com",
			env:         map[string]string{"TASK_TEST_TOKEN": "s3cret"},
			want:        map[string]string{"PRIVATE-TOKEN": "s3cret"},
		},
		{
			name:        "environment variable inside a longer value",
			hostHeaders: HostHeaders{"gitlab.com": {"Authorization": "Bearer $TASK_TEST_TOKEN"}},
			host:        "gitlab.com",
			env:         map[string]string{"TASK_TEST_TOKEN": "s3cret"},
			want:        map[string]string{"Authorization": "Bearer s3cret"},
		},
		{
			name:        "undefined environment variable expands to nothing",
			hostHeaders: HostHeaders{"gitlab.com": {"PRIVATE-TOKEN": "${TASK_TEST_UNSET}"}}, //nolint:gosec // an env var reference, not a credential
			host:        "gitlab.com",
			want:        map[string]string{"PRIVATE-TOKEN": ""},
		},
		{
			name:        "header name with a space",
			hostHeaders: HostHeaders{"gitlab.com": {"PRIVATE TOKEN": "token"}},
			host:        "gitlab.com",
			wantErr:     `remote auth for host "gitlab.com": invalid header name "PRIVATE TOKEN"`,
		},
		{
			name:        "header name outside the HTTP token grammar",
			hostHeaders: HostHeaders{"gitlab.com": {"X-Foo(bar)": "token"}},
			host:        "gitlab.com",
			wantErr:     `remote auth for host "gitlab.com": invalid header name "X-Foo(bar)"`,
		},
		{
			name:        "empty header name",
			hostHeaders: HostHeaders{"gitlab.com": {"": "token"}},
			host:        "gitlab.com",
			wantErr:     `remote auth for host "gitlab.com": invalid header name ""`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			headers, err := resolveAuthHeaders(test.hostHeaders, test.host)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, headers)
		})
	}
}

func TestAuthTransport(t *testing.T) {
	t.Parallel()

	transport := &authTransport{
		base:    roundTripperFunc(func(req *http.Request) (*http.Response, error) { return newResponse(req), nil }),
		host:    "gitlab.com",
		headers: map[string]string{"PRIVATE-TOKEN": "token"},
	}

	t.Run("sets the headers on the configured host", func(t *testing.T) {
		t.Parallel()
		req := newRequest(t, "https://gitlab.com/api/v4/Taskfile.yml")
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, "token", resp.Request.Header.Get("PRIVATE-TOKEN"))
		// The transport must leave the request it was given untouched.
		assert.Empty(t, req.Header.Get("PRIVATE-TOKEN"))
	})

	t.Run("leaves any other host alone", func(t *testing.T) {
		t.Parallel()
		req := newRequest(t, "https://example.com/Taskfile.yml")
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Empty(t, resp.Request.Header.Get("PRIVATE-TOKEN"))
	})
}

func TestWithAuthHeadersDoesNotMutateTheDefaultClient(t *testing.T) {
	t.Parallel()

	client := withAuthHeaders(http.DefaultClient, "gitlab.com", map[string]string{"PRIVATE-TOKEN": "token"})

	assert.NotSame(t, http.DefaultClient, client)
	assert.Nil(t, http.DefaultClient.Transport)
	assert.IsType(t, &authTransport{}, client.Transport)
}

// Both requests must carry the headers: RemoteExists probes with HEAD before
// ReadContext issues the GET.
func TestHTTPNodeAuthHeaders(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "s3cret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("version: '3'\n"))
	}))
	defer srv.Close()

	t.Setenv("TASK_TEST_TOKEN", "s3cret")
	node, err := NewHTTPNode(srv.URL+"/Taskfile.yml", "", true,
		WithAuthHeaders(HostHeaders{
			mustHost(t, srv.URL): {"PRIVATE-TOKEN": "${TASK_TEST_TOKEN}"}, //nolint:gosec // an env var reference, not a credential
		}),
	)
	require.NoError(t, err)

	b, err := node.Read()
	require.NoError(t, err)
	assert.Equal(t, "version: '3'\n", string(b))
	assert.Equal(t, []string{"HEAD", "GET"}, methods)
}

// A server bouncing the request must not get the credentials forwarded to it.
func TestHTTPNodeAuthHeadersNotSentOnRedirect(t *testing.T) {
	t.Parallel()

	var received []string
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.Header.Get("PRIVATE-TOKEN"))
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("version: '3'\n"))
	}))
	defer elsewhere.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/Taskfile.yml", http.StatusFound)
	}))
	defer srv.Close()

	node, err := NewHTTPNode(srv.URL+"/Taskfile.yml", "", true,
		WithAuthHeaders(HostHeaders{
			mustHost(t, srv.URL): {"PRIVATE-TOKEN": "s3cret"},
		}),
	)
	require.NoError(t, err)

	_, err = node.Read()
	require.NoError(t, err)
	require.NotEmpty(t, received)
	for _, header := range received {
		assert.Empty(t, header, "the token must not follow a redirect to another host")
	}
}

// A node must build without the credentials it would need to download, so that
// cached and offline runs do not require them.
func TestHTTPNodeAuthHeadersResolvedLazily(t *testing.T) { //nolint:paralleltest // t.Setenv cannot be used in parallel tests
	node, err := NewHTTPNode("https://gitlab.com/Taskfile.yml", "", false,
		WithAuthHeaders(HostHeaders{
			"gitlab.com": {"PRIVATE-TOKEN": "${TASK_TEST_LAZY}"}, //nolint:gosec // an env var reference, not a credential
		}),
	)
	require.NoError(t, err)

	// Defined only after the node was built: the value must still be picked up.
	t.Setenv("TASK_TEST_LAZY", "s3cret")

	headers, err := resolveAuthHeaders(node.authHeaders, node.url.Host)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PRIVATE-TOKEN": "s3cret"}, headers)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	return req
}

func newResponse(req *http.Request) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Request: req, Header: http.Header{}}
}

func mustHost(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed.Host
}
