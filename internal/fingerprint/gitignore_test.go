package fingerprint

import (
	"os"
	"path/filepath"
	"strings"
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

func TestGlobsWithGitignoreParentDirIgnored(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	buildDir := filepath.Join(dir, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "keep.txt"), []byte("keep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "other.txt"), []byte("other"), 0o644))

	// Git cannot re-include a file under an ignored directory.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, ".gitignore"), []byte("!keep.txt\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./**/*"},
	}

	files, err := Globs(dir, globs, true)
	require.NoError(t, err)

	for _, f := range files {
		base := filepath.Base(f)
		assert.NotEqual(t, "keep.txt", base, "keep.txt must stay excluded under ignored build/")
		assert.NotEqual(t, "other.txt", base, "other.txt must stay excluded under ignored build/")
	}
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

func TestGlobsWithGitignoreCrossFileNegation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(subDir, "debug.log"), []byte("debug"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "other.log"), []byte("other"), 0o644))

	// Root ignores all *.log; a nested .gitignore re-includes debug.log.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, ".gitignore"), []byte("!debug.log\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	files, err := Globs(subDir, globs, true)
	require.NoError(t, err)

	hasDebug, hasOther := false, false
	for _, f := range files {
		switch filepath.Base(f) {
		case "debug.log":
			hasDebug = true
		case "other.log":
			hasOther = true
		}
	}
	assert.True(t, hasDebug, "debug.log should be re-included by the nested negation")
	assert.False(t, hasOther, "other.log should remain excluded by the root *.log rule")
}

func TestGlobsWithGitignoreDeepGlob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(subDir, "keep.txt"), []byte("keep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "gen.out"), []byte("gen"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(subDir, ".gitignore"), []byte("*.out\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./**/*"},
	}

	files, err := Globs(dir, globs, true)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "gen.out", filepath.Base(f), "gen.out should be excluded by the nested .gitignore reached via deep glob")
	}
}

func TestGlobsWithGitignoreDoubleDotFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	// A ".."-prefixed name must not be skipped by the out-of-tree guard.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "..keep.log"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("y"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	files, err := Globs(dir, globs, true)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "..keep.log", filepath.Base(f), "..keep.log should be excluded by *.log")
	}
}

func TestGlobsWithGitignoreLongLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initGitRepo(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("y"), 0o644))

	// A line over bufio.Scanner's 64KB limit triggers a scan error; patterns
	// parsed before it must survive.
	longLine := strings.Repeat("a", 70*1024)
	content := "*.log\n" + longLine + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(content), 0o644))

	globs := []*ast.Glob{
		{Glob: "./*"},
	}

	files, err := Globs(dir, globs, true)
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, "ignored.log", filepath.Base(f), "*.log parsed before the long line should still apply")
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
