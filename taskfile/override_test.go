package taskfile

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile/ast"
)

func writeOverrideFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func readOverrideFixture(t *testing.T, dir string, opts ...ReaderOption) (*ast.Taskfile, error) {
	t.Helper()
	node, err := NewFileNode("", dir)
	require.NoError(t, err)
	graph, err := NewReader(opts...).Read(t.Context(), node)
	if err != nil {
		return nil, err
	}
	return graph.Merge()
}

func TestAutomaticOverrideMerge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	main := writeOverrideFixture(t, dir, "Taskfile.yml", `version: '3'
vars:
  PORT: 3000
  KEEP: original
  CONFIG: {map: {old: old, shared: original}}
env:
  FIRST: original
  KEEP: original
tasks:
  dev:
    aliases: [old]
    vars: {OLD: old}
    env: {OLD: old}
    deps: [old]
    preconditions: [exit 1]
    cmds: [exit 1]
    desc: old
    silent: true
  keep: echo keep
  empty: echo old
`)
	override := writeOverrideFixture(t, dir, "Taskfile.override.yml", `version: '3.0.0'
vars:
  PORT: 4000
  CONFIG: {map: {shared: local}}
  NEW: local
  NULL: null
env:
  FIRST: local
  NEW: local
tasks:
  dev:
    cmds: ['echo {{.PORT}}']
  added: echo added
  empty: null
`)
	tf, err := readOverrideFixture(t, dir, WithAutomaticOverride(true))
	require.NoError(t, err)
	assert.Equal(t, main, tf.Location)
	assert.Equal(t, []string{"PORT", "KEEP", "CONFIG", "NEW", "NULL"}, slices.Collect(tf.Vars.Keys()))
	port, _ := tf.Vars.Get("PORT")
	assert.Equal(t, 4000, port.Value)
	config, _ := tf.Vars.Get("CONFIG")
	assert.Equal(t, map[string]any{"shared": "local"}, config.Value)
	null, ok := tf.Vars.Get("NULL")
	assert.True(t, ok)
	assert.Nil(t, null.Value)
	assert.Equal(t, []string{"FIRST", "KEEP", "NEW"}, slices.Collect(tf.Env.Keys()))
	first, _ := tf.Env.Get("FIRST")
	assert.Equal(t, "local", first.Value)
	keep, _ := tf.Env.Get("KEEP")
	assert.Equal(t, "original", keep.Value)
	assert.Equal(t, []string{"dev", "keep", "empty", "added"}, slices.Collect(tf.Tasks.Keys(nil)))
	dev, _ := tf.Tasks.Get("dev")
	assert.Equal(t, &ast.Task{
		Task: "dev", Cmds: []*ast.Cmd{{Cmd: "echo {{.PORT}}"}},
		Location: &ast.Location{Taskfile: override, Line: 11, Column: 3},
	}, dev)
	added, _ := tf.Tasks.Get("added")
	assert.Equal(t, override, added.Location.Taskfile)
	unchanged, _ := tf.Tasks.Get("keep")
	assert.Equal(t, main, unchanged.Location.Taskfile)
	empty, ok := tf.Tasks.Get("empty")
	require.True(t, ok)
	assert.Empty(t, empty.Cmds)
	assert.Equal(t, override, empty.Location.Taskfile)
}

func TestAutomaticOverrideNames(t *testing.T) {
	t.Parallel()
	// Each suffix of the list checks both fallback and first-present priority.
	// Case-insensitive filesystems only exercise the .yml/.yaml priority.
	for i, name := range defaultOverrides {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeOverrideFixture(t, dir, "Taskfile.dist.yml", "version: '3'\n")
			for j := len(defaultOverrides) - 1; j >= i; j-- {
				writeOverrideFixture(t, dir, defaultOverrides[j], fmt.Sprintf("version: '3'\nvars: {WINNER: %d}\n", j))
			}
			tf, err := readOverrideFixture(t, dir, WithAutomaticOverride(true))
			require.NoError(t, err)
			winner, _ := tf.Vars.Get("WINNER")
			assert.Equal(t, i, winner.Value)
		})
	}
}

func TestAutomaticOverrideErrors(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, content, message string }{
		{"invalid YAML", "version: [", "yaml:"},
		{"missing version", "vars: {PORT: 4000}", "Missing schema version"},
		{"mismatched version", "version: '3.1'", "has version 3.1.0"},
		{"sequence", "- version: '3'", "taskfile override"},
	}
	for _, key := range []string{"includes", "unknown", "<<"} {
		tests = append(tests, struct{ name, content, message string }{key, "version: '3'\n" + key + ": null", "not allowed"})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeOverrideFixture(t, dir, "Taskfile.yml", "version: '3'\n")
			path := writeOverrideFixture(t, dir, "Taskfile.override.yml", tt.content)
			writeOverrideFixture(t, dir, "Taskfile.override.yaml", "version: '3'\n")
			_, err := readOverrideFixture(t, dir, WithAutomaticOverride(true))
			require.ErrorContains(t, err, tt.message)
			assert.Contains(t, err.Error(), "Taskfile.override.yml")
			if decodeErr, ok := errors.AsType[*errors.TaskfileDecodeError](err); ok {
				assert.Equal(t, path, decodeErr.Location)
				assert.Positive(t, decodeErr.Line)
			}
		})
	}
}

func TestAutomaticOverrideReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeOverrideFixture(t, dir, "Taskfile.yml", "version: '3'\n")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "Taskfile.override.yml"), 0o755))
	writeOverrideFixture(t, dir, "Taskfile.override.yaml", "version: '3'\n")
	_, err := readOverrideFixture(t, dir, WithAutomaticOverride(true))
	require.ErrorContains(t, err, "Taskfile.override.yml")
}

func TestAutomaticOverrideIncludes(t *testing.T) {
	t.Parallel()
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprint(conflict), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeOverrideFixture(t, dir, "Taskfile.yml", "version: '3'\nvars: {INCLUDE: missing}\nincludes:\n  docs: '{{.INCLUDE}}/Taskfile.yml'\n")
			override := "version: '3'\nvars: {INCLUDE: docs}\n"
			if conflict {
				override += "tasks:\n  docs:build: echo local\n"
			}
			writeOverrideFixture(t, dir, "Taskfile.override.yml", override)
			included := writeOverrideFixture(t, dir, "docs/Taskfile.yml", "version: '3'\ntasks:\n  build: echo included\n")
			writeOverrideFixture(t, dir, "docs/Taskfile.override.yml", "invalid YAML: [")
			tf, err := readOverrideFixture(t, dir, WithAutomaticOverride(true))
			if conflict {
				var conflictErr *errors.TaskNameFlattenConflictError
				require.ErrorAs(t, err, &conflictErr)
				assert.Equal(t, "docs:build", conflictErr.TaskName)
			} else {
				require.NoError(t, err)
				build, ok := tf.Tasks.Get("docs:build")
				require.True(t, ok)
				assert.Equal(t, included, build.Location.Taskfile)
			}
		})
	}
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
