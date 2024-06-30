package taskfile_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile"
)

func TestNewCache(t *testing.T) {
	cacheDir := t.TempDir()
	_, err := taskfile.NewCache(cacheDir)
	require.NoErrorf(t, err, "creating new cache in temporary directory '%s'", cacheDir)
}

func TestNewCacheWithTTL(t *testing.T) {
	cacheDir := t.TempDir()
	_, err := taskfile.NewCache(cacheDir, taskfile.WithTTL(time.Hour))
	require.NoErrorf(t, err, "creating new cache in temporary directory '%s'", cacheDir)
}
