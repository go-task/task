package task_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
)

func TestAutomaticOverrideExecution(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, main               string
		explicit, subdir, absent bool
	}{
		{name: "automatic", main: "Taskfile.yml"},
		{name: "dist", main: "Taskfile.dist.yml"},
		{name: "subdirectory", main: "Taskfile.yml", subdir: true},
		{name: "explicit", main: "Taskfile.yml", explicit: true},
		{name: "absent", main: "Taskfile.yml", absent: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			main := filepath.Join(dir, tt.main)
			require.NoError(t, os.WriteFile(main, []byte(`version: '3'
vars:
  PORT: 3000
env:
  TASK_OVERRIDE_ENV: main
tasks:
  dev:
    cmds: ['echo {{.PORT}} $TASK_OVERRIDE_ENV']
  origin:
    cmds: ['echo {{.TASKFILE}} {{.ROOT_TASKFILE}}']
`), 0o600))
			override := filepath.Join(dir, "Taskfile.override.yml")
			if !tt.absent {
				require.NoError(t, os.WriteFile(override, []byte(`version: '3'
vars:
  PORT: 4000
env:
  TASK_OVERRIDE_ENV: local
tasks:
  added:
    cmds: ['echo {{.TASKFILE}} {{.ROOT_TASKFILE}}']
`), 0o600))
			}
			workdir := dir
			if tt.subdir {
				workdir = filepath.Join(dir, "child")
				require.NoError(t, os.Mkdir(workdir, 0o755))
				// Only the discovered root's directory is searched.
				require.NoError(t, os.WriteFile(filepath.Join(workdir, "Taskfile.override.yml"), []byte("bad: ["), 0o600))
			}
			var stdout bytes.Buffer
			e := task.NewExecutor(task.WithDir(workdir), task.WithStdout(&stdout), task.WithStderr(&stdout), task.WithSilent(true))
			if tt.explicit {
				e.Entrypoint = main
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: "dev"}))
			if tt.explicit || tt.absent {
				assert.Equal(t, "3000 main\n", stdout.String())
			} else {
				assert.Equal(t, "4000 local\n", stdout.String())
				stdout.Reset()
				require.NoError(t, e.Run(t.Context(), &task.Call{Task: "added"}, &task.Call{Task: "origin"}))
				assert.Equal(t, fmt.Sprintf("%s %s\n%s %s\n", filepath.ToSlash(override), filepath.ToSlash(main), filepath.ToSlash(main), filepath.ToSlash(main)), stdout.String())
			}
			assert.Equal(t, main, e.Entrypoint)
			assert.Equal(t, main, e.Taskfile.Location)
		})
	}
}
