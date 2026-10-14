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

func TestStatusShellOptions(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		globalOpts  string
		taskOpts    string
		commandOpts string
		status      string
		upToDate    bool
	}{
		{name: "default", status: "false | true", upToDate: true},
		{name: "global pipefail", globalOpts: "set: [pipefail]", status: "false | true"},
		{name: "task pipefail", taskOpts: "set: [pipefail]", status: "false | true"},
		{name: "merged options", globalOpts: "set: [pipefail]", taskOpts: "set: [noglob]", status: "false | true"},
		{name: "successful pipeline", globalOpts: "set: [pipefail]", status: "true | true", upToDate: true},
		{name: "command options stay local", commandOpts: "set: [pipefail]", status: "false | true", upToDate: true},
		{name: "global nullglob", globalOpts: "shopt: [nullglob]", status: `set -- *.missing; test "$#" -gt 0`},
		{name: "task nullglob", taskOpts: "shopt: [nullglob]", status: `set -- *.missing; test "$#" -gt 0`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			content := fmt.Sprintf(`version: '3'
%s
tasks:
  build:
    %s
    status:
      - '%s'
    cmds:
      - cmd: echo rebuilt
        %s
`, test.globalOpts, test.taskOpts, test.status, test.commandOpts)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0o600))

			var stdout bytes.Buffer
			e := task.NewExecutor(task.WithDir(dir), task.WithStdout(&stdout), task.WithSilent(true))
			require.NoError(t, e.Setup())
			call := &task.Call{Task: "build"}
			err := e.Status(t.Context(), call)
			if test.upToDate {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, `Task "build" is not up-to-date`)
			}
			require.NoError(t, e.Run(t.Context(), call))
			if test.upToDate {
				assert.Empty(t, stdout.String())
			} else {
				assert.Equal(t, "rebuilt\n", stdout.String())
			}
		})
	}
}
