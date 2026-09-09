package taskfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileNodeResolveLiteralSpecialDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "Taskfile.yml")
	require.NoError(t, os.WriteFile(entrypoint, []byte("version: '3'\n"), 0o600))
	node, err := NewFileNode(entrypoint, "")
	require.NoError(t, err)

	for _, name := range []string{"project.ROOT_DIR", "project.TASKFILE_DIR", "project.USER_WORKING_DIR"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resolvedDir, err := node.ResolveDir(name)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, name), resolvedDir)

			resolvedEntrypoint, err := node.ResolveEntrypoint(filepath.Join(name, "Taskfile.yml"))
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, name, "Taskfile.yml"), resolvedEntrypoint)
		})
	}
}
