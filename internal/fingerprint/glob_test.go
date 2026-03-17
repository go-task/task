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

func TestSplitGlobPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pattern        string
		expectedBase   string
		expectedSuffix string
	}{
		{
			name:           "double star with extension",
			pattern:        "/home/user/project/gqlgen/**/*.gql",
			expectedBase:   "/home/user/project/gqlgen",
			expectedSuffix: "**/*.gql",
		},
		{
			name:           "double star only",
			pattern:        "/home/user/project/**",
			expectedBase:   "/home/user/project",
			expectedSuffix: "**",
		},
		{
			name:           "single star",
			pattern:        "/home/user/project/*.go",
			expectedBase:   "/home/user/project",
			expectedSuffix: "*.go",
		},
		{
			name:           "no glob characters",
			pattern:        "/home/user/project/file.go",
			expectedBase:   "/home/user/project/file.go",
			expectedSuffix: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, suffix := splitGlobPattern(tt.pattern)
			assert.Equal(t, tt.expectedBase, base)
			assert.Equal(t, tt.expectedSuffix, suffix)
		})
	}
}

func TestMatchesGlobStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		pattern  string
		expected bool
	}{
		{
			name:     "matches double star with extension",
			filePath: "/project/src/deep/nested/file.gql",
			pattern:  "/project/src/**/*.gql",
			expected: true,
		},
		{
			name:     "does not match wrong extension",
			filePath: "/project/src/deep/file.go",
			pattern:  "/project/src/**/*.gql",
			expected: false,
		},
		{
			name:     "does not match outside base",
			filePath: "/other/src/file.gql",
			pattern:  "/project/src/**/*.gql",
			expected: false,
		},
		{
			name:     "matches double star catch-all",
			filePath: "/project/src/any/file.txt",
			pattern:  "/project/src/**",
			expected: true,
		},
		{
			name:     "does not match catch-all outside base",
			filePath: "/other/file.txt",
			pattern:  "/project/src/**",
			expected: false,
		},
		{
			name:     "no double star returns false",
			filePath: "/project/file.go",
			pattern:  "/project/*.go",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, matchesGlobStar(tt.filePath, tt.pattern))
		})
	}
}

// setupTestDir creates a temp directory with test files and returns the path
// and a cleanup function.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	for _, sub := range []string{"src/a", "src/b/nested", "other"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}

	// Create files
	files := []string{
		"src/a/one.gql",
		"src/a/two.go",
		"src/b/nested/three.gql",
		"src/b/four.gql",
		"other/five.gql",
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("test"), 0o644))
	}

	return dir
}

func TestAnyGlobNewerThan(t *testing.T) {
	t.Parallel()

	dir := setupTestDir(t)
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name          string
		globs         []*ast.Glob
		referenceTime time.Time
		expected      bool
	}{
		{
			name:          "files newer than past time",
			globs:         []*ast.Glob{{Glob: "src/**/*.gql"}},
			referenceTime: past,
			expected:      true,
		},
		{
			name:          "no files newer than future time",
			globs:         []*ast.Glob{{Glob: "src/**/*.gql"}},
			referenceTime: future,
			expected:      false,
		},
		{
			name:          "no matching files",
			globs:         []*ast.Glob{{Glob: "src/**/*.xyz"}},
			referenceTime: past,
			expected:      false,
		},
		{
			name: "multiple globs",
			globs: []*ast.Glob{
				{Glob: "src/**/*.gql"},
				{Glob: "other/**/*.gql"},
			},
			referenceTime: past,
			expected:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := anyGlobNewerThan(dir, tt.globs, tt.referenceTime)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobsMaxTime(t *testing.T) {
	t.Parallel()

	dir := setupTestDir(t)

	// Touch one file to have a known max time
	knownTime := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(
		filepath.Join(dir, "src/b/nested/three.gql"),
		knownTime, knownTime,
	))

	tests := []struct {
		name     string
		globs    []*ast.Glob
		expected time.Time
	}{
		{
			name:     "finds max time among matching files",
			globs:    []*ast.Glob{{Glob: "src/**/*.gql"}},
			expected: knownTime,
		},
		{
			name:     "no matching files returns zero time",
			globs:    []*ast.Glob{{Glob: "src/**/*.xyz"}},
			expected: time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := GlobsMaxTime(dir, tt.globs)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnyGlobNewerThanWithNegation(t *testing.T) {
	t.Parallel()

	dir := setupTestDir(t)
	past := time.Now().Add(-1 * time.Hour)

	// Negated globs should fall back to Globs+anyFileNewerThan
	globs := []*ast.Glob{
		{Glob: "src/**/*.gql"},
		{Glob: "src/b/**/*.gql", Negate: true},
	}
	result, err := anyGlobNewerThan(dir, globs, past)
	require.NoError(t, err)
	// src/a/one.gql should still match (not negated)
	assert.True(t, result)
}
