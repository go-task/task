package fingerprint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestChecksumCheckerRecursiveSourcesAndRequiredGenerates(t *testing.T) {
	t.Parallel()

	sourceRoot := filepath.Join("input tree", "folder-a")
	outputRoot := filepath.Join("output tree", "folder-a")
	includedSources := []string{
		filepath.Join(sourceRoot, ".hidden-file-a.aaa"),
		filepath.Join(sourceRoot, ".hidden-file-b.bbb"),
		filepath.Join(sourceRoot, "file-c.ccc"),
		filepath.Join(sourceRoot, "file-d.ddd"),
		filepath.Join(sourceRoot, ".hidden-folder-a", "file-e.eee"),
		filepath.Join(sourceRoot, "folder-b", "folder-c", "file-f.fff"),
		filepath.Join(sourceRoot, "folder-b", "folder-c", "file-g.ggg"),
		filepath.Join(sourceRoot, "folder-b", "folder-c", "file-h.hhh"),
		filepath.Join(sourceRoot, "folder-b", "folder-c", "file-i.iii"),
		filepath.Join(sourceRoot, "folder-b", "folder-c", "folder-d", ".hidden-file-j.jjj"),
		filepath.Join(sourceRoot, "folder-b", "folder-e", "file-k.kkk"),
	}
	excludedSources := []string{
		filepath.Join(sourceRoot, "excluded-file-a.xxx"),
		filepath.Join(sourceRoot, "excluded-folder-a", "file-b.yyy"),
		filepath.Join(sourceRoot, "excluded-folder-a", "folder-b", ".hidden-file-c.zzz"),
		filepath.Join(sourceRoot, "excluded-folder-a", ".hidden-folder-b", "file-d.www"),
	}
	generatedOutputs := []string{
		filepath.Join(outputRoot, ".hidden-file-a.aaa"),
		filepath.Join(outputRoot, "folder-b", "file-b.bbb"),
		filepath.Join(outputRoot, "folder-b", "file-c.ccc"),
		filepath.Join(outputRoot, "folder-b", "file-d.ddd"),
		filepath.Join(outputRoot, "folder-b", "file-e.eee"),
		filepath.Join(outputRoot, "folder-b", "file-f.fff"),
		filepath.Join(outputRoot, "folder-b", "file-g.ggg"),
		filepath.Join(outputRoot, "folder-b", "file-h.hhh"),
		filepath.Join(outputRoot, "folder-b", "folder-c", "folder-d", "file-i.iii"),
	}

	tests := []struct {
		name    string
		sources []routedGlob
	}{
		{
			name: "optimized inclusion and exclusion",
			sources: []routedGlob{
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "*")}, fast: true},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-file-a.xxx"), Negate: true}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-folder-a", "**", "*"), Negate: true}, fast: true},
			},
		},
		{
			name: "fallback inclusion",
			sources: []routedGlob{
				{glob: &ast.Glob{Glob: filepath.Join("input tree", "*", "**", "*")}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-file-a.xxx"), Negate: true}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-folder-a", "**", "*"), Negate: true}, fast: true},
			},
		},
		{
			name: "fallback exclusion",
			sources: []routedGlob{
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "*")}, fast: true},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "excluded-file-a.xxx"), Negate: true}, fast: false},
				{glob: &ast.Glob{Glob: filepath.Join(sourceRoot, "**", "excluded-folder-a", "**", "*"), Negate: true}, fast: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()
			writeFile := func(relativePath, contents string) {
				t.Helper()
				path := filepath.Join(rootDir, relativePath)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
			}
			for i, source := range includedSources {
				writeFile(source, fmt.Sprintf("included source %d\n", i))
			}
			for i, source := range excludedSources {
				writeFile(source, fmt.Sprintf("excluded source %d\n", i))
			}
			for i, output := range generatedOutputs {
				writeFile(output, fmt.Sprintf("generated output %d\n", i))
			}

			generates := make([]*ast.Glob, 0, len(generatedOutputs))
			for _, output := range generatedOutputs {
				generates = append(generates, &ast.Glob{Glob: output})
			}
			task := &ast.Task{
				Task:      "cache-" + tt.name,
				Dir:       rootDir,
				Sources:   requireGlobRoutes(t, rootDir, tt.sources),
				Generates: generates,
			}
			checker := NewChecksumChecker(filepath.Join(rootDir, ".task-cache"), false)
			isUpToDate := func() bool {
				t.Helper()
				upToDate, err := checker.IsUpToDate(task)
				require.NoError(t, err)
				return upToDate
			}

			require.False(t, isUpToDate(), "the first check should prime the source checksum")
			require.True(t, isUpToDate(), "an unchanged task should be cached")

			for i, source := range includedSources {
				writeFile(source, fmt.Sprintf("included source %d mutation\n", i))
				assert.False(t, isUpToDate(), "mutating included source %q should invalidate the cache", source)
				require.True(t, isUpToDate(), "the updated checksum for included source %q should be cached", source)
			}

			for i, source := range excludedSources {
				writeFile(source, fmt.Sprintf("excluded source %d mutation\n", i))
				assert.True(t, isUpToDate(), "mutating excluded source %q should preserve the cache", source)
				require.True(t, isUpToDate(), "the current checksum for excluded source %q should be cached", source)
			}

			for i, output := range generatedOutputs {
				path := filepath.Join(rootDir, output)
				require.NoError(t, os.Remove(path))
				require.False(t, isUpToDate(), "missing required output %q should invalidate the cache", output)
				writeFile(output, fmt.Sprintf("generated output %d\n", i))
				require.True(t, isUpToDate(), "restoring required output %q should restore the cache", output)
			}
		})
	}
}

func TestNormalizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		In, Out string
	}{
		{"foobarbaz", "foobarbaz"},
		{"foo/bar/baz", "foo-bar-baz"},
		{"foo@bar/baz", "foo-bar-baz"},
		{"foo1bar2baz3", "foo1bar2baz3"},
		{"foo\\bar", "foo-bar"},
		{"foo_bar", "foo-bar"},
		{"foo[bar]baz", "foo-bar-baz"},
		{"foo^bar`baz", "foo-bar-baz"},
	}
	for _, test := range tests {
		assert.Equal(t, test.Out, normalizeFilename(test.In))
	}
}
