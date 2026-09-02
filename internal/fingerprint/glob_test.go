package fingerprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestGlobsInDirectoryContainingQuote(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "test'd")
	require.NoError(t, os.Mkdir(dir, 0o755))
	file := filepath.Join(dir, "test.in")
	require.NoError(t, os.WriteFile(file, nil, 0o644))

	files, err := Globs(dir, []*ast.Glob{{Glob: "test.in"}}, false)

	require.NoError(t, err)
	require.Equal(t, []string{filepath.ToSlash(file)}, files)
}
