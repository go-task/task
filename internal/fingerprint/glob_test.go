package fingerprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobHandlesApostropheInPath(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, "test'd")
	require.NoError(t, os.Mkdir(dir, 0o755))

	file := filepath.Join(dir, "test.in")
	require.NoError(t, os.WriteFile(file, []byte("input"), 0o644))

	matches, err := glob(dir, "test.in")
	require.NoError(t, err)
	require.Equal(t, []string{filepath.ToSlash(file)}, matches)
}
