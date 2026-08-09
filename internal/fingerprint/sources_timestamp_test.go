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

func TestTimestampCheckerValue(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source.txt"), []byte("content"), 0o644))
	checker := NewTimestampChecker(t.TempDir(), true)

	t.Run("reports the newest source modification time", func(t *testing.T) {
		t.Parallel()

		value, err := checker.Value(&ast.Task{
			Dir:     dir,
			Sources: []*ast.Glob{{Glob: "source.txt"}},
		})
		require.NoError(t, err)
		assert.IsType(t, time.Time{}, value)
		assert.False(t, value.(time.Time).IsZero())
	})

	t.Run("reports no value when no source matches", func(t *testing.T) {
		t.Parallel()

		value, err := checker.Value(&ast.Task{
			Dir:     dir,
			Sources: []*ast.Glob{{Glob: "gen/**/*.go"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "", value)
	})
}
