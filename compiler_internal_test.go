package task

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestGetSpecialVarsRemote(t *testing.T) {
	t.Parallel()

	uwd := t.TempDir()
	uwdSlash := filepath.ToSlash(uwd)
	localProj := filepath.Join(uwd, "proj")
	localProjSlash := filepath.ToSlash(localProj)
	localTaskfile := filepath.Join(localProj, "Taskfile.yml")
	localTaskfileSlash := filepath.ToSlash(localTaskfile)
	absTaskDir := filepath.Join(uwd, "opt", "work")
	absTaskDirSlash := filepath.ToSlash(absTaskDir)

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
			entrypoint:       localTaskfile,
			compilerDir:      localProj,
			taskDir:          "",
			taskfileLocation: localTaskfile,
			wantRootTaskfile: localTaskfileSlash,
			wantRootDir:      localProjSlash,
			wantTaskfile:     localTaskfileSlash,
			wantTaskfileDir:  localProjSlash,
			wantTaskDir:      localProjSlash,
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
			wantTaskDir:      uwdSlash,
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
			wantTaskDir:      filepath.ToSlash(filepathext.SmartJoin(uwd, "subdir")),
		},
		{
			name:             "https entrypoint, absolute task.dir",
			entrypoint:       "https://taskfile.dev/Taskfile.yml",
			compilerDir:      "",
			taskDir:          absTaskDir,
			taskfileLocation: "https://taskfile.dev/Taskfile.yml",
			wantRootTaskfile: "https://taskfile.dev/Taskfile.yml",
			wantRootDir:      "",
			wantTaskfile:     "https://taskfile.dev/Taskfile.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      absTaskDirSlash,
		},
		{
			name:             "git entrypoint",
			entrypoint:       "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			compilerDir:      "",
			taskDir:          "",
			taskfileLocation: "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			wantRootTaskfile: "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			wantRootDir:      "",
			wantTaskfile:     "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			wantTaskfileDir:  "",
			wantTaskDir:      uwdSlash,
		},
		{
			name:             "local root, remote included task",
			entrypoint:       localTaskfile,
			compilerDir:      localProj,
			taskDir:          "",
			taskfileLocation: "https://taskfile.dev/included.yml",
			wantRootTaskfile: localTaskfileSlash,
			wantRootDir:      localProjSlash,
			wantTaskfile:     "https://taskfile.dev/included.yml",
			wantTaskfileDir:  "",
			wantTaskDir:      uwdSlash,
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
