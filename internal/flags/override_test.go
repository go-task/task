package flags

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
)

func TestAutomaticOverrideGlobal(t *testing.T) { //nolint:paralleltest // changes process-wide flags and home directory
	originalGlobal, originalDir, originalEntrypoint, originalTimeout := Global, Dir, Entrypoint, Timeout
	t.Cleanup(func() {
		Global, Dir, Entrypoint, Timeout = originalGlobal, originalDir, originalEntrypoint, originalTimeout
	})
	dir := t.TempDir()
	homeKey := "HOME"
	if runtime.GOOS == "windows" {
		homeKey = "USERPROFILE"
	}
	t.Setenv(homeKey, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\nvars: {PORT: 3000}\ntasks:\n  dev: echo {{.PORT}}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.override.yml"), []byte("version: '3'\nvars: {PORT: 4000}\n"), 0o600))

	Global, Dir, Entrypoint, Timeout = true, "", "", time.Second
	var stdout bytes.Buffer
	e := task.NewExecutor(WithFlags(), task.WithStdout(&stdout), task.WithStderr(&stdout), task.WithSilent(true))
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "dev"}))
	assert.Equal(t, "4000\n", stdout.String())
}
