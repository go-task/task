package taskfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
)

func ExampleWithTTL() {
	c, _ := NewCache(os.TempDir(), WithTTL(2*time.Minute+30*time.Second))

	fmt.Println(c.ttl)
	// Output: 2m30s
}

var discardLogger = &logger.Logger{
	Stdout: io.Discard,
	Stderr: io.Discard,
}

func primeNewCache(t *testing.T, tempDir string, cacheOpts ...CacheOption) (*Cache, *FileNode) {
	t.Helper()

	cache, err := NewCache(tempDir, cacheOpts...)
	require.NoErrorf(t, err, "creating new cache in temporary directory '%s'", tempDir)

	// Prime the temporary cache directory with a basic Taskfile.
	filename := "Taskfile.yaml"
	srcTaskfile := filepath.Join("testdata", filename)
	dstTaskfile := filepath.Join(cache.dir, filename)

	taskfileBytes, err := os.ReadFile(srcTaskfile)
	require.NoErrorf(t, err, "reading from testdata Taskfile (%s)", srcTaskfile)

	err = os.WriteFile(dstTaskfile, taskfileBytes, 0o640)
	require.NoErrorf(t, err, "writing to temporary Taskfile (%s)", dstTaskfile)

	// Create a new file node in the cache, with the entrypoint copied above.
	fileNode, err := NewFileNode(discardLogger, dstTaskfile, cache.dir)
	require.NoError(t, err, "creating new file node")

	return cache, fileNode
}

func TestCache(t *testing.T) {
	cache, fileNode := primeNewCache(t, t.TempDir())

	// Attempt to read from cache, then write, then read again.
	_, err := cache.read(fileNode)
	require.ErrorAs(t, err, &os.ErrNotExist, "reading from cache before writing should match error type")

	writeBytes := []byte("some bytes")
	err = cache.write(fileNode, writeBytes)
	require.NoError(t, err, "writing bytes to cache")

	readBytes, err := cache.read(fileNode)
	require.NoError(t, err, "reading from cache after write should not error")
	require.Equal(t, writeBytes, readBytes, "bytes read from cache should match bytes written")
}

func TestCacheInsideTTL(t *testing.T) {
	// Prime a new Cache with a TTL of one minute.
	cache, fileNode := primeNewCache(t, t.TempDir(), WithTTL(time.Minute))

	// Write some bytes for the cached file.
	writeBytes := []byte("some bytes")
	err := cache.write(fileNode, writeBytes)
	require.NoError(t, err, "writing bytes to cache")

	// Reading from the cache while still inside the TTL should get the written bytes back.
	readBytes, err := cache.read(fileNode)
	require.NoError(t, err, "reading from cache inside TTL should not error")
	require.Equal(t, writeBytes, readBytes, "bytes read from cache should match bytes written")
}

func TestCacheOutsideTTL(t *testing.T) {
	// Prime a new Cache with a TTL of one second.
	cache, fileNode := primeNewCache(t, t.TempDir(), WithTTL(time.Second))

	// Write some bytes for the cached file.
	writeBytes := []byte("some bytes")
	err := cache.write(fileNode, writeBytes)
	require.NoError(t, err, "writing bytes to cache")

	// Sleep for two seconds so that the cached file is outside of TTL.
	time.Sleep(2 * time.Second)

	// Reading from the cache after sleeping past the end of TTL should get an error.
	readBytes, err := cache.read(fileNode)
	require.Empty(t, readBytes, "should not have read any bytes from cache")
	require.ErrorIs(t, err, ErrExpired, "should get 'expired' error when attempting to read from cache outside of TTL")
}
