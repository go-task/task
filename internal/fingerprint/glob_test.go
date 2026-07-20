package fingerprint

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fsext"
	"github.com/go-task/task/v3/taskfile/ast"
)

type routedGlob struct {
	glob *ast.Glob
	fast bool
}

func TestGlobsRecursiveIncludesAllFilesAndHonorsExclusions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{
		"input tree/folder-a/.hidden-file-a.aaa",
		"input tree/folder-a/file-b.bbb",
		"input tree/folder-a/folder-b/.hidden-file-c.ccc",
		"input tree/folder-a/folder-b/file-d.ddd",
		"input tree/folder-a/folder-c/.hidden-folder-a/file-e.eee",
		"input tree/folder-a/folder-c/folder-d/.hidden-file-f.fff",
		"input tree/folder-a/folder-c/folder-d/file-g.ggg",
		"input tree/folder-a/excluded-file-a.xxx",
		"input tree/folder-a/excluded-folder-a/.hidden-file-b.yyy",
		"input tree/folder-a/excluded-folder-a/.hidden-folder-b/file-c.zzz",
		"input tree/folder-a/excluded-folder-a/file-d.www",
		"input tree/folder-a/excluded-folder-a/folder-b/file-e.vvv",
	}
	writeGlobFiles(t, dir, files)

	sourceRoot := filepath.Join("input tree", "folder-a")
	tests := []struct {
		name  string
		globs []routedGlob
	}{
		{
			name: "optimized inclusion and exclusion",
			globs: []routedGlob{
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "*")}, fast: true},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-file-a.xxx"), Negate: true}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-folder-a", "**", "*"), Negate: true}, fast: true},
			},
		},
		{
			name: "optimized inclusion and fallback exclusion",
			globs: []routedGlob{
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "*")}, fast: true},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-file-a.xxx"), Negate: true}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "excluded-folder-a", "**", "*"), Negate: true}, fast: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Globs(dir, requireGlobRoutes(t, dir, tt.globs), false)
			require.NoError(t, err)
			require.Equal(t, globPaths(dir, files[:7]), got)
		})
	}
}

func TestGlobsFallbackPatternsIncludeHiddenEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{
		"input tree/folder-a/.hidden-file-a.aaa",
		"input tree/folder-a/file-a.aaa",
		"input tree/folder-a/file-a.ccc",
		"input tree/folder-a/file-a.bbb",
		"input tree/folder-a/nested/.hidden-file-b.bbb",
		"input tree/folder-a/nested/file-b.aaa",
		"input tree/folder-a/nested/deeper-folder-a/file-c.aaa",
		"input tree/folder-a/.hidden-folder-a/file-d.aaa",
		"input tree/folder-a/.hidden-folder-a/nested/.hidden-file-e.aaa",
		"input tree/folder-a/.hidden-folder-a/folder-b/file-f.bbb",
		"input tree/folder-a/folder-b/.hidden-file-g.bbb",
		"input tree/folder-a/folder-b/file-g.aaa",
		"input tree/folder-a/folder-b/deeper-folder-b/file-h.aaa",
		"input tree/.hidden-folder-b/.hidden-root-file-a.aaa",
		"input tree/.hidden-folder-b/nested/file-i.bbb",
		"input tree/.hidden-folder-b/folder-b/file-j.aaa",
		"input tree/folder-c/file-k.ccc",
		"input tree/folder-c/nested/file-k.aaa",
		"input tree/folder-c/folder-b/file-l.aaa",
	}
	writeGlobFiles(t, dir, files)

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:    "nonrecursive",
			pattern: filepath.Join("input tree", "folder-a", "*"),
			expected: []string{
				"input tree/folder-a/.hidden-file-a.aaa",
				"input tree/folder-a/file-a.aaa",
				"input tree/folder-a/file-a.ccc",
				"input tree/folder-a/file-a.bbb",
			},
		},
		{
			name:     "wildcard root",
			pattern:  filepath.Join("input tree", "*", "**", "*"),
			expected: files,
		},
		{
			name:    "nested suffix",
			pattern: filepath.Join("input tree", "**", "nested", "*"),
			expected: []string{
				"input tree/folder-a/nested/.hidden-file-b.bbb",
				"input tree/folder-a/nested/file-b.aaa",
				"input tree/folder-a/.hidden-folder-a/nested/.hidden-file-e.aaa",
				"input tree/.hidden-folder-b/nested/file-i.bbb",
				"input tree/folder-c/nested/file-k.aaa",
			},
		},
		{
			name:    "brace suffix",
			pattern: filepath.Join("input tree", "**", "*.{aaa,bbb}"),
			expected: []string{
				"input tree/folder-a/.hidden-file-a.aaa",
				"input tree/folder-a/file-a.aaa",
				"input tree/folder-a/file-a.bbb",
				"input tree/folder-a/nested/.hidden-file-b.bbb",
				"input tree/folder-a/nested/file-b.aaa",
				"input tree/folder-a/nested/deeper-folder-a/file-c.aaa",
				"input tree/folder-a/.hidden-folder-a/file-d.aaa",
				"input tree/folder-a/.hidden-folder-a/nested/.hidden-file-e.aaa",
				"input tree/folder-a/.hidden-folder-a/folder-b/file-f.bbb",
				"input tree/folder-a/folder-b/.hidden-file-g.bbb",
				"input tree/folder-a/folder-b/file-g.aaa",
				"input tree/folder-a/folder-b/deeper-folder-b/file-h.aaa",
				"input tree/.hidden-folder-b/.hidden-root-file-a.aaa",
				"input tree/.hidden-folder-b/nested/file-i.bbb",
				"input tree/.hidden-folder-b/folder-b/file-j.aaa",
				"input tree/folder-c/nested/file-k.aaa",
				"input tree/folder-c/folder-b/file-l.aaa",
			},
		},
		{
			name:    "multiple recursive markers",
			pattern: filepath.Join("input tree", "**", "folder-b", "**", "*"),
			expected: []string{
				"input tree/folder-a/.hidden-folder-a/folder-b/file-f.bbb",
				"input tree/folder-a/folder-b/.hidden-file-g.bbb",
				"input tree/folder-a/folder-b/file-g.aaa",
				"input tree/folder-a/folder-b/deeper-folder-b/file-h.aaa",
				"input tree/.hidden-folder-b/folder-b/file-j.aaa",
				"input tree/folder-c/folder-b/file-l.aaa",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			globs := requireGlobRoutes(t, dir, []routedGlob{{
				glob: &ast.Glob{Glob: tt.pattern},
				fast: false,
			}})
			got, err := Globs(dir, globs, false)
			require.NoError(t, err)
			require.Equal(t, globPaths(dir, tt.expected), got)
		})
	}
}

func requireGlobRoutes(t *testing.T, dir string, globs []routedGlob) []*ast.Glob {
	t.Helper()

	patterns := make([]*ast.Glob, 0, len(globs))
	for _, routed := range globs {
		_, fast, err := fsext.FastRecursiveGlob(filepathext.SmartJoin(dir, routed.glob.Glob))
		require.NoError(t, err)
		require.Equal(t, routed.fast, fast, "glob route for %q", routed.glob.Glob)
		patterns = append(patterns, routed.glob)
	}
	return patterns
}

func writeGlobFiles(t *testing.T, dir string, files []string) {
	t.Helper()

	for _, file := range files {
		path := filepath.Join(dir, filepath.FromSlash(file))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(file), 0o600))
	}
}

func globPaths(dir string, files []string) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, filepath.ToSlash(filepath.Join(dir, filepath.FromSlash(file))))
	}
	sort.Strings(paths)
	return paths
}
