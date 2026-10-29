package taskfile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/errors"
)

// recordingServer answers HEAD probes for a fixed set of paths and records
// every request it receives, so tests can assert which URLs were tried.
type recordingServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string
}

func (s *recordingServer) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// newRecordingServer serves HTTP 200 with the given content type for each
// listed path, and 404 for anything else.
func newRecordingServer(t *testing.T, contentTypes map[string]string) *recordingServer {
	t.Helper()
	s := &recordingServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.Method+" "+r.URL.Path)
		s.mu.Unlock()

		contentType, ok := contentTypes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestRemoteExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		contentTypes map[string]string
		wantPath     string
		wantRequests []string
	}{
		{
			name:         "direct hit returns the URL unchanged",
			path:         "/Taskfile.yml",
			contentTypes: map[string]string{"/Taskfile.yml": "text/yaml"},
			wantPath:     "/Taskfile.yml",
			wantRequests: []string{"HEAD /Taskfile.yml"},
		},
		{
			name:         "content type with charset is still accepted",
			path:         "/Taskfile.yml",
			contentTypes: map[string]string{"/Taskfile.yml": "application/yaml; charset=utf-8"},
			wantPath:     "/Taskfile.yml",
			wantRequests: []string{"HEAD /Taskfile.yml"},
		},
		{
			name: "directory root falls back to a default name",
			path: "",
			contentTypes: map[string]string{
				"/":             "text/html; charset=utf-8",
				"/Taskfile.yml": "text/plain",
			},
			wantPath:     "/Taskfile.yml",
			wantRequests: []string{"HEAD /", "HEAD /Taskfile.yml"},
		},
		{
			name: "sub path falls back to a default name",
			path: "/nested",
			contentTypes: map[string]string{
				"/nested/Taskfile.yml": "text/plain",
			},
			wantPath:     "/nested/Taskfile.yml",
			wantRequests: []string{"HEAD /nested", "HEAD /nested/Taskfile.yml"},
		},
		{
			name: "fallback follows DefaultTaskfiles precedence",
			path: "",
			contentTypes: map[string]string{
				"/taskfile.yml":  "text/plain",
				"/Taskfile.yaml": "text/plain",
			},
			wantPath:     "/taskfile.yml",
			wantRequests: []string{"HEAD /", "HEAD /Taskfile.yml", "HEAD /taskfile.yml"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			srv := newRecordingServer(t, test.contentTypes)
			u, err := url.Parse(srv.URL + test.path)
			require.NoError(t, err)

			got, err := RemoteExists(context.Background(), *u, srv.Client())
			require.NoError(t, err)
			assert.Equal(t, test.wantPath, got.Path)
			assert.Equal(t, test.wantRequests, srv.seen())
		})
	}
}

func TestRemoteExistsNotFound(t *testing.T) {
	t.Parallel()

	srv := newRecordingServer(t, nil)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	_, err = RemoteExists(context.Background(), *u, srv.Client())

	var notFound errors.TaskfileNotFoundError
	require.ErrorAs(t, err, &notFound)
	// The original URL, then every default name.
	assert.Len(t, srv.seen(), len(DefaultTaskfiles)+1)
}

// A 401 means the default names would need the same credentials, so trying
// them would only add rejected requests.
func TestRemoteExistsUnauthorizedDoesNotProbeDefaultNames(t *testing.T) {
	t.Parallel()

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	_, err = RemoteExists(context.Background(), *u, srv.Client())

	var fetchFailed errors.TaskfileFetchFailedError
	require.ErrorAs(t, err, &fetchFailed)
	assert.Equal(t, http.StatusUnauthorized, fetchFailed.HTTPStatusCode)
	assert.Equal(t, 1, requests)
}

func TestRemoteExistsCancelledContext(t *testing.T) {
	t.Parallel()

	srv := newRecordingServer(t, nil)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = RemoteExists(ctx, *u, srv.Client())
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, srv.seen())
}
