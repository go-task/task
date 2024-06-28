package taskfile

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
)

func TestCache(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewCache(cacheDir)
	require.NoErrorf(t, err, "creating new cache in temporary directory '%s'", cacheDir)

	// Prime the temporary cache directory with a basic Taskfile.
	filename := "Taskfile.yaml"
	srcTaskfile := filepath.Join("testdata", filename)
	dstTaskfile := filepath.Join(cache.dir, filename)
	err = os.Link(srcTaskfile, dstTaskfile)
	require.NoErrorf(t, err, "creating hardlink from testdata (%s) to temporary cache (%s)", srcTaskfile, dstTaskfile)

	discardLogger := &logger.Logger{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	// Create a new file node in the cache, with the entrypoint hardlinked above.
	fileNode, err := NewFileNode(discardLogger, dstTaskfile, cache.dir)
	require.NoError(t, err, "creating new file node")

	_, err = cache.read(fileNode)
	require.ErrorAs(t, err, &os.ErrNotExist, "reading from cache before writing should match error type")

	writeBytes := []byte("some bytes")
	err = cache.write(fileNode, writeBytes)
	require.NoError(t, err, "writing bytes to cache")

	readBytes, err := cache.read(fileNode)
	require.NoError(t, err, "reading from cache after write should not error")
	require.Equal(t, writeBytes, readBytes, "bytes read from cache should match bytes written")
}
