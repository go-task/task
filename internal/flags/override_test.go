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

func TestAutomaticOverrideFlags(t *testing.T) { //nolint:paralleltest // changes process-wide flags and home directory
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
	for _, tt := range []struct { //nolint:paralleltest // subtests change process-wide flags
		name             string
		global           bool
		entrypoint, want string
	}{
		{name: "dir", want: "4000\n"},
		{name: "global", global: true, want: "4000\n"},
		{name: "global explicit", global: true, entrypoint: filepath.Join(dir, "Taskfile.yml"), want: "3000\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Global, Dir, Entrypoint, Timeout = tt.global, dir, tt.entrypoint, time.Second
			if tt.global {
				Dir = ""
			}
			var stdout bytes.Buffer
			e := task.NewExecutor(WithFlags(), task.WithStdout(&stdout), task.WithStderr(&stdout), task.WithSilent(true))
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: "dev"}))
			assert.Equal(t, tt.want, stdout.String())
		})
	}
}
