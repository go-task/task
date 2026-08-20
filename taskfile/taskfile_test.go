package taskfile

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/errors"
)

// alwaysStatus answers every request with the given status.
func alwaysStatus(t *testing.T, status int) (*url.URL, *int) {
	t.Helper()
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return mustParse(t, srv.URL), &requests
}

func TestRemoteExistsUnauthorized(t *testing.T) {
	t.Parallel()

	u, requests := alwaysStatus(t, http.StatusUnauthorized)
	_, err := RemoteExists(t.Context(), *u, http.DefaultClient)

	var fetchErr errors.TaskfileFetchFailedError
	require.ErrorAs(t, err, &fetchErr)
	assert.Equal(t, http.StatusUnauthorized, fetchErr.HTTPStatusCode)
	assert.Equal(t, 1, *requests)
}

// A 403 is ambiguous, so it keeps the existing behaviour.
func TestRemoteExistsForbiddenEverywhere(t *testing.T) {
	t.Parallel()

	u, requests := alwaysStatus(t, http.StatusForbidden)
	_, err := RemoteExists(t.Context(), *u, http.DefaultClient)

	var notFoundErr errors.TaskfileNotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Greater(t, *requests, 1)
}

func TestRemoteExistsForbiddenDirectoryWithReadableTaskfile(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Taskfile.yml" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	found, err := RemoteExists(t.Context(), *mustParse(t, srv.URL), http.DefaultClient)
	require.NoError(t, err)
	assert.Equal(t, "/Taskfile.yml", found.Path)
}

func TestRemoteExistsNotFound(t *testing.T) {
	t.Parallel()

	u, _ := alwaysStatus(t, http.StatusNotFound)
	_, err := RemoteExists(t.Context(), *u, http.DefaultClient)

	var notFoundErr errors.TaskfileNotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func mustParse(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}
