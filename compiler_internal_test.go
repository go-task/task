package task

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestGetSpecialVarsRemote(t *testing.T) {
	t.Parallel()

	const uwd = "/home/user/project"

	tests := []struct {
		name             string
		entrypoint       string
		compilerDir      string
		taskDir          string
		taskfileLocation string
		wantRootTaskfile string
		wantRootDir      string
		wantTaskfile     string
		wantTaskfileDir  string
		wantTaskDir      string
	}{
		{
			name:             "local entrypoint, local task",
			entrypoint:       "/abs/proj/Taskfile.yml",
			compilerDir:      "/abs/proj",
			taskDir:          "",
			taskfileLocation: "/abs/proj/Taskfile.yml",
			wantRootTaskfile: "/abs/proj/Taskfile.yml",
			wantRootDir:      "/abs/proj",
			wantTaskfile:     "/abs/proj/Taskfile.yml",
			wantTaskfileDir:  "/abs/proj",
			wantTaskDir:      "/abs/proj",
		},
		{
			name:             "https entrypoint, empty task.dir",
			entrypoint:       "https://taskfile.dev/Taskfile.yml",
			compilerDir:      "",
			taskDir:          "",
			taskfileLocation: "https://taskfile.dev/Taskfile.yml",
			wantRootTaskfile: "https://taskfile.dev/Taskfile.yml",
			wantRootDir:      "",
			wantTaskfile:     "https://taskfile.dev/Taskfile.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      uwd,
		},
		{
			name:             "https entrypoint, relative task.dir",
			entrypoint:       "https://taskfile.dev/Taskfile.yml",
			compilerDir:      "",
			taskDir:          "subdir",
			taskfileLocation: "https://taskfile.dev/Taskfile.yml",
			wantRootTaskfile: "https://taskfile.dev/Taskfile.yml",
			wantRootDir:      "",
			wantTaskfile:     "https://taskfile.dev/Taskfile.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      filepath.ToSlash(filepath.Join(uwd, "subdir")),
		},
		{
			name:             "https entrypoint, absolute task.dir",
			entrypoint:       "https://taskfile.dev/Taskfile.yml",
			compilerDir:      "",
			taskDir:          "/opt/work",
			taskfileLocation: "https://taskfile.dev/Taskfile.yml",
			wantRootTaskfile: "https://taskfile.dev/Taskfile.yml",
			wantRootDir:      "",
			wantTaskfile:     "https://taskfile.dev/Taskfile.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      "/opt/work",
		},
		{
			name:             "git entrypoint",
			entrypoint:       "git://github.com/foo/bar.git//Taskfile.yml",
			compilerDir:      "",
			taskDir:          "",
			taskfileLocation: "git://github.com/foo/bar.git//Taskfile.yml",
			wantRootTaskfile: "git://github.com/foo/bar.git//Taskfile.yml",
			wantRootDir:      "",
			wantTaskfile:     "git://github.com/foo/bar.git//Taskfile.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      uwd,
		},
		{
			name:             "local root, remote included task",
			entrypoint:       "/abs/proj/Taskfile.yml",
			compilerDir:      "/abs/proj",
			taskDir:          "",
			taskfileLocation: "https://taskfile.dev/included.yml",
			wantRootTaskfile: "/abs/proj/Taskfile.yml",
			wantRootDir:      "/abs/proj",
			wantTaskfile:     "https://taskfile.dev/included.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      uwd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Compiler{
				Dir:            tt.compilerDir,
				Entrypoint:     tt.entrypoint,
				UserWorkingDir: uwd,
			}
			task := &ast.Task{
				Task:     "mytask",
				Dir:      tt.taskDir,
				Location: &ast.Location{Taskfile: tt.taskfileLocation},
			}

			vars, err := c.getSpecialVars(task, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantRootTaskfile, vars["ROOT_TASKFILE"], "ROOT_TASKFILE")
			assert.Equal(t, tt.wantRootDir, vars["ROOT_DIR"], "ROOT_DIR")
			assert.Equal(t, tt.wantTaskfile, vars["TASKFILE"], "TASKFILE")
			assert.Equal(t, tt.wantTaskfileDir, vars["TASKFILE_DIR"], "TASKFILE_DIR")
			assert.Equal(t, tt.wantTaskDir, vars["TASK_DIR"], "TASK_DIR")
		})
	}
}
