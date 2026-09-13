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

func TestIncludedDotenvScopes(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"Taskfile.yml": `version: '3'
dotenv: [root.env]
env:
  DOTENV_EXPLICIT: explicit
includes:
  app:
    taskfile: app/Taskfile.yml
    dir: work
  sibling: sibling/Taskfile.yml
  a:
    taskfile: shared/Taskfile.yml
    vars: {FLAVOR: a}
  b:
    taskfile: shared/Taskfile.yml
    vars: {FLAVOR: b}
  flat:
    taskfile: flat/Taskfile.yml
    flatten: true
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"root.env":         "DOTENV_CHOICE=root\nDOTENV_ROOT=root\n",
		"without-root.yml": "version: '3'\nincludes:\n  app: app/Taskfile.yml\n",
		"app/Taskfile.yml": `version: '3'
vars:
  CONFIG: config
env:
  CONFIG_ENV: config
dotenv: [override.env, '{{.CONFIG}}/base.env', '{{.CONFIG_ENV}}/env-path.env', missing.env]
includes:
  child: child/Taskfile.yml
  common: ../common/Taskfile.yml
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
  task-env:
    env: {DOTENV_CHOICE: task}
    cmds: ['echo "$DOTENV_CHOICE"']
  task-dotenv:
    dotenv: ['{{.TASKFILE_DIR}}/task.env']
    cmds: ['echo "$DOTENV_CHOICE"']
`,
		"app/override.env":        "DOTENV_CHOICE=app\nDOTENV_APP=app\nDOTENV_EXPLICIT=dotenv\n",
		"app/config/base.env":     "DOTENV_CHOICE=base\nDOTENV_BASE=base\n",
		"app/config/env-path.env": "DOTENV_ENV_PATH=env-path\n",
		"app/task.env":            "DOTENV_CHOICE=task-dotenv\n",
		"app/child/Taskfile.yml": `version: '3'
dotenv: [child.env]
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"app/child/child.env": "DOTENV_CHOICE=child\n",
		"common/Taskfile.yml": `version: '3'
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE | default ""}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"sibling/Taskfile.yml": `version: '3'
env:
  CONFIG_ENV: nonexistent
dotenv: [sibling.env]
includes:
  common: ../common/Taskfile.yml
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"sibling/sibling.env": "DOTENV_CHOICE=sibling\n",
		"shared/Taskfile.yml": `version: '3'
dotenv: ['{{.FLAVOR}}.env', '{{.FLAVOR}}.env']
tasks:
  inspect: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"shared/a.env": "DOTENV_CHOICE=a\n",
		"shared/b.env": "DOTENV_CHOICE=b\n",
		"flat/Taskfile.yml": `version: '3'
dotenv: ['{{.TASKFILE_DIR}}/flat.env']
tasks:
  flattened: 'echo "{{.DOTENV_CHOICE}}|$DOTENV_CHOICE|$DOTENV_APP"'
`,
		"flat/flat.env":     "DOTENV_CHOICE=flat\n",
		"work/override.env": "DOTENV_CHOICE=wrong-directory\n",
	}
	dir := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	}
	for _, test := range []struct {
		name, dir, entrypoint, call, want string
	}{
		{name: "root isolation", call: "inspect", want: "root|root|\n"},
		{name: "include directory and file order", call: "app:inspect", want: "app|app|app\n"},
		{name: "standalone", dir: "app", call: "inspect", want: "app|app|app\n"},
		{name: "standalone root remains global", dir: "app", call: "common:inspect", want: "app|app|app\n"},
		{name: "no dotenv inherited without root", entrypoint: "without-root.yml", call: "app:common:inspect", want: "||\n"},
		{name: "without root dotenv", entrypoint: "without-root.yml", call: "app:inspect", want: "app|app|app\n"},
		{name: "nested dotenv is independent", call: "app:child:inspect", want: "child|child|\n"},
		{name: "common only inherits root", call: "app:common:inspect", want: "root|root|\n"},
		{name: "sibling isolation", call: "sibling:inspect", want: "sibling|sibling|\n"},
		{name: "common via another parent", call: "sibling:common:inspect", want: "root|root|\n"},
		{name: "include vars a", call: "a:inspect", want: "a|a|\n"},
		{name: "include vars b", call: "b:inspect", want: "b|b|\n"},
		{name: "flatten", call: "flattened", want: "flat|flat|\n"},
		{name: "task env priority", call: "app:task-env", want: "task\n"},
		{name: "task dotenv priority", call: "app:task-dotenv", want: "task-dotenv\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			entrypoint := ""
			if test.entrypoint != "" {
				entrypoint = filepath.Join(dir, test.entrypoint)
			}
			e := task.NewExecutor(
				task.WithDir(filepath.Join(dir, test.dir)),
				task.WithEntrypoint(entrypoint),
				task.WithSilent(true),
				task.WithStdout(&stdout),
			)
			require.NoError(t, e.Setup())
			call := &task.Call{Task: test.call}
			require.NoError(t, e.Run(t.Context(), call))
			assert.Equal(t, test.want, stdout.String())
			compiled, err := e.CompiledTask(call)
			require.NoError(t, err)
			if test.call == "app:inspect" {
				value, ok := compiled.Env.Get("DOTENV_ENV_PATH")
				require.True(t, ok)
				assert.Equal(t, "env-path", value.Value)
				value, ok = compiled.Env.Get("DOTENV_BASE")
				require.True(t, ok)
				assert.Equal(t, "base", value.Value)
			}
			if test.dir == "" && test.entrypoint == "" {
				value, ok := compiled.Env.Get("DOTENV_EXPLICIT")
				require.True(t, ok)
				assert.Equal(t, "explicit", value.Value)
				value, ok = compiled.Env.Get("DOTENV_ROOT")
				require.True(t, ok)
				assert.Equal(t, "root", value.Value)
			}
		})
	}
}

func TestIncludedDotenvErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, path, contents, want string
	}{
		{name: "parse error", path: "broken.env", contents: "INVALID=\"unterminated", want: "broken.env"},
		{name: "template error", path: "{{fail \"invalid dotenv path\"}}", want: "invalid dotenv path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\nincludes:\n  app: app.yml\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "app.yml"), []byte("version: '3'\ndotenv: ['"+test.path+"']\ntasks:\n  test: echo test\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.env"), []byte(test.contents), 0o600))
			e := task.NewExecutor(task.WithDir(dir))
			require.ErrorContains(t, e.Setup(), test.want)
		})
	}
}

func TestIncludedDotenvDynamicCache(t *testing.T) {
	t.Parallel()
	for _, rootDotenv := range []bool{false, true} {
		t.Run(fmt.Sprintf("root_dotenv=%t", rootDotenv), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root := `version: '3'
vars:
  CONFIG:
    sh: 'echo run >> global-calls; echo config'
includes:
  a:
    taskfile: included.yml
    vars:
      FLAVOR: a
      LOCAL:
        sh: 'echo run >> {{.FLAVOR}}-calls; echo {{.FLAVOR}}'
  b:
    taskfile: included.yml
    vars:
      FLAVOR: b
      LOCAL:
        sh: 'echo run >> {{.FLAVOR}}-calls; echo {{.FLAVOR}}'
`
			if rootDotenv {
				root += "dotenv: [root.env]\n"
			}
			for name, contents := range map[string]string{
				"Taskfile.yml": root,
				"root.env":     "DOTENV_CACHE_ROOT=root\n",
				"included.yml": `version: '3'
dotenv: ['{{.CONFIG}}-{{.LOCAL}}.env']
tasks:
  inspect: 'echo {{.DOTENV_CACHE_VALUE}}'
`,
				"config-a.env": "DOTENV_CACHE_VALUE=a\n",
				"config-b.env": "DOTENV_CACHE_VALUE=b\n",
			} {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600))
			}
			e := task.NewExecutor(task.WithDir(dir))
			require.NoError(t, e.Setup())
			for _, flavor := range []string{"a", "b"} {
				compiled, err := e.CompiledTask(&task.Call{Task: flavor + ":inspect"})
				require.NoError(t, err)
				assert.Equal(t, "echo "+flavor, compiled.Cmds[0].Cmd)
			}
			for _, name := range []string{"global-calls", "a-calls", "b-calls"} {
				contents, err := os.ReadFile(filepath.Join(dir, name))
				require.NoError(t, err)
				assert.Equal(t, "run\n", string(contents), "%s should be evaluated once during setup and compilation", name)
			}
		})
	}
}
