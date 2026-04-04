package fingerprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
}

func TestGlobsWithGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "included.txt"), []byte("included"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("ignored"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "also-included.txt"), []byte("also included"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	filesWithout, err := Globs(dir, globs, false)
	require.NoError(t, err)

	filesWith, err := Globs(dir, globs, true)
	require.NoError(t, err)

	hasLog := false
	for _, f := range filesWithout {
		if filepath.Base(f) == "ignored.log" {
			hasLog = true
			break
		}
	}
	assert.True(t, hasLog, "ignored.log should be present without gitignore filter")

	hasLog = false
	for _, f := range filesWith {
		if filepath.Base(f) == "ignored.log" {
			hasLog = true
			break
		}
	}
	assert.False(t, hasLog, "ignored.log should be excluded with gitignore filter")

	txtCount := 0
	for _, f := range filesWith {
		if filepath.Ext(f) == ".txt" {
			txtCount++
		}
	}
	assert.Equal(t, 2, txtCount, "both .txt files should remain")
}

func TestGlobsWithGitignoreNested(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(subDir, "keep.txt"), []byte("keep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "build.out"), []byte("build"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, ".gitignore"), []byte("*.out\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	files, err := Globs(subDir, globs, true)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "build.out", filepath.Base(f), "build.out should be excluded by nested .gitignore")
	}
}

func TestGlobsWithGitignoreNoRepo(t *testing.T) {
	t.Parallel()

	// Cannot use t.TempDir() here because it creates a dir inside the
	// go-task repo which has a .git parent, defeating the "no repo" test.
	dir, err := os.MkdirTemp("", "task-gitignore-norepo-*") //nolint:usetesting
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	files, err := Globs(dir, globs, true)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
