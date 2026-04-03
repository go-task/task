package fingerprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	_, err := git.PlainInit(dir, false)
	require.NoError(t, err)
}

func TestGlobsWithGitignore(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp("", "task-gitignore-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	initGitRepo(t, dir)

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "included.txt"), []byte("included"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("ignored"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "also-included.txt"), []byte("also included"), 0o644))

	// Create .gitignore
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	// Without gitignore - should include all files
	filesWithout, err := Globs(dir, globs, false)
	require.NoError(t, err)

	// With gitignore - should exclude .log files
	filesWith, err := Globs(dir, globs, true)
	require.NoError(t, err)

	// The .log file should be in the unfiltered list
	hasLog := false
	for _, f := range filesWithout {
		if filepath.Base(f) == "ignored.log" {
			hasLog = true
			break
		}
	}
	assert.True(t, hasLog, "ignored.log should be present without gitignore filter")

	// The .log file should NOT be in the filtered list
	hasLog = false
	for _, f := range filesWith {
		if filepath.Base(f) == "ignored.log" {
			hasLog = true
			break
		}
	}
	assert.False(t, hasLog, "ignored.log should be excluded with gitignore filter")

	// .txt files should still be present
	txtCount := 0
	for _, f := range filesWith {
		if filepath.Ext(f) == ".txt" {
			txtCount++
		}
	}
	assert.Equal(t, 2, txtCount, "both .txt files should remain")
}

func TestGlobsWithGitignoreDisabled(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp("", "task-gitignore-disabled-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	initGitRepo(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.log"), []byte("content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	// WithGitignore(false, ...) should not filter
	files, err := Globs(dir, globs, false)
	require.NoError(t, err)

	hasLog := false
	for _, f := range files {
		if filepath.Base(f) == "file.log" {
			hasLog = true
			break
		}
	}
	assert.True(t, hasLog, "file.log should be present when gitignore is disabled")
}

func TestGlobsWithGitignoreNoRepo(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp("", "task-gitignore-norepo-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	// Should not error and should return all files
	files, err := Globs(dir, globs, true)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
