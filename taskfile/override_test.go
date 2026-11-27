package taskfile

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeOverrideFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

// Settings have no visible effect on a task's output, so they are checked on
// the merged Taskfile.
func TestAutomaticOverrideSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeOverrideFixture(t, dir, "Taskfile.yml", `version: '3'
method: timestamp
run: once
interval: 1s
set: [pipefail]
shopt: [globstar]
use_gitignore: true
`)
	writeOverrideFixture(t, dir, "Taskfile.override.yml", `version: '3'
method: none
run: when_changed
interval: 2s
set: []
use_gitignore: false
`)
	node, err := NewFileNode("", dir)
	require.NoError(t, err)
	graph, err := NewReader(WithAutomaticOverride(true)).Read(t.Context(), node)
	require.NoError(t, err)
	tf, err := graph.Merge()
	require.NoError(t, err)
	assert.Equal(t, "none", tf.Method)
	assert.Equal(t, "when_changed", tf.Run)
	assert.Equal(t, 2*time.Second, tf.Interval)
	assert.Equal(t, []string{}, tf.Set)
	assert.Equal(t, []string{"globstar"}, tf.Shopt)
	assert.Equal(t, new(false), tf.UseGitignore)
}

func TestAutomaticOverrideRemote(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeOverrideFixture(t, dir, "Taskfile.override.yml", "invalid YAML: [")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "version: '3'\n")
	}))
	defer server.Close()
	node, err := NewHTTPNode(server.URL+"/Taskfile.yml", dir, true)
	require.NoError(t, err)
	_, err = NewReader(WithAutomaticOverride(true), WithTempDir(t.TempDir())).Read(t.Context(), node)
	require.NoError(t, err)
}

func TestAutomaticOverrideStdin(t *testing.T) { //nolint:paralleltest // replaces process-wide stdin
	dir := t.TempDir()
	writeOverrideFixture(t, dir, "Taskfile.override.yml", "invalid YAML: [")
	replaceStdin(t, []byte("version: '3'\n"))
	node, err := NewStdinNode(dir)
	require.NoError(t, err)
	_, err = NewReader(WithAutomaticOverride(true)).Read(t.Context(), node)
	require.NoError(t, err)
}
