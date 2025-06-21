package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestTimestampFileLocation(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "task-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test directories
	rootDir := filepath.Join(tempDir, "root")
	subDir := filepath.Join(rootDir, "subdir")
	require.NoError(t, os.MkdirAll(rootDir, 0755))
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Create source files
	rootSourceFile := filepath.Join(rootDir, "root.txt")
	subSourceFile := filepath.Join(subDir, "sub.txt")
	require.NoError(t, os.WriteFile(rootSourceFile, []byte("root source"), 0644))
	require.NoError(t, os.WriteFile(subSourceFile, []byte("sub source"), 0644))

	// Create generate files
	rootGenerateFile := filepath.Join(rootDir, "root.txt.processed")
	subGenerateFile := filepath.Join(subDir, "sub.txt.processed")
	require.NoError(t, os.WriteFile(rootGenerateFile, []byte("Processing root.txt"), 0644))
	require.NoError(t, os.WriteFile(subGenerateFile, []byte("Processing sub.txt"), 0644))

	// Set file times
	now := time.Now()
	sourceTime := now.Add(-1 * time.Hour)
	generateTime := now
	require.NoError(t, os.Chtimes(rootSourceFile, sourceTime, sourceTime))
	require.NoError(t, os.Chtimes(subSourceFile, sourceTime, sourceTime))
	require.NoError(t, os.Chtimes(rootGenerateFile, generateTime, generateTime))
	require.NoError(t, os.Chtimes(subGenerateFile, generateTime, generateTime))

	// Create tasks
	rootTask := &ast.Task{
		Task:      "root",
		Dir:       rootDir,
		Sources:   []*ast.Glob{{Glob: "*.txt"}},
		Generates: []*ast.Glob{{Glob: "*.txt.processed"}},
		Method:    "timestamp",
	}
	subTask := &ast.Task{
		Task:      "sub",
		Dir:       subDir,
		Sources:   []*ast.Glob{{Glob: "*.txt"}},
		Generates: []*ast.Glob{{Glob: "*.txt.processed"}},
		Method:    "timestamp",
	}

	// Create checker
	checker := NewTimestampChecker(tempDir, false)

	// Test root task
	rootUpToDate, err := checker.IsUpToDate(rootTask)
	require.NoError(t, err)
	assert.True(t, rootUpToDate, "Root task should be up-to-date")

	// Test sub task
	subUpToDate, err := checker.IsUpToDate(subTask)
	require.NoError(t, err)
	assert.True(t, subUpToDate, "Sub task should be up-to-date")

	// Verify timestamp files were created in the correct locations
	rootTimestampFile := filepath.Join(tempDir, rootDir, "timestamp", normalizeFilename(rootTask.Task))
	subTimestampFile := filepath.Join(tempDir, subDir, "timestamp", normalizeFilename(subTask.Task))

	_, err = os.Stat(rootTimestampFile)
	assert.NoError(t, err, "Root timestamp file should exist")
	_, err = os.Stat(subTimestampFile)
	assert.NoError(t, err, "Sub timestamp file should exist")

	// Test that modifying a source file makes the task not up-to-date
	newSourceTime := now.Add(1 * time.Hour)
	require.NoError(t, os.Chtimes(rootSourceFile, newSourceTime, newSourceTime))

	rootUpToDate, err = checker.IsUpToDate(rootTask)
	require.NoError(t, err)
	assert.False(t, rootUpToDate, "Root task should not be up-to-date after source file modification")

	// Sub task should still be up-to-date
	subUpToDate, err = checker.IsUpToDate(subTask)
	require.NoError(t, err)
	assert.True(t, subUpToDate, "Sub task should still be up-to-date")
}
