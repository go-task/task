package summary_test

import (
	"bytes"
	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/summary"
	"github.com/go-task/task/v2/internal/taskfile"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrintsDependenciesIfPresent(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Deps: []*taskfile.Dep{
			{Task: "dep1"},
			{Task: "dep2"},
			{Task: "dep3"},
		},
	}

	summary.Print(&l, task)

	assert.Contains(t, buffer.String(), "\ndependencies:\n")
	assert.Contains(t, buffer.String(), "\n - dep1\n")
	assert.Contains(t, buffer.String(), "\n - dep2\n")
	assert.Contains(t, buffer.String(), "\n - dep3\n")
}

func TestDoesNotPrintDependenciesIfMissing(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Deps: []*taskfile.Dep{},
	}

	summary.Print(&l, task)

	assert.NotContains(t, buffer.String(), "dependencies:")
}

func TestPrintTaskName(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Task: "my-task-name",
	}

	summary.Print(&l, task)

	assert.Contains(t, buffer.String(), "task: my-task-name\n")
}

func TestPrintTaskCommandsIfPresent(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Cmds: []*taskfile.Cmd{
			{Cmd: "command-1"},
			{Cmd: "command-2"},
			{Task: "task-1"},
		},
	}

	summary.Print(&l, task)

	assert.Contains(t, buffer.String(), "\ncommands:\n")
	assert.Contains(t, buffer.String(), "\n - command-1\n")
	assert.Contains(t, buffer.String(), "\n - command-2\n")
	assert.Contains(t, buffer.String(), "\n - Task: task-1\n")
}

func TestDoesNotPrintCommandIfMissing(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Cmds: []*taskfile.Cmd{},
	}

	summary.Print(&l, task)

	assert.NotContains(t, buffer.String(), "commands")
}

func TestFullSummary(t *testing.T) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stdout:  buffer,
		Stderr:  buffer,
		Verbose: false,
	}
	task := &taskfile.Task{
		Task:    "sample-task",
		Summary: "line1\nline2\nline3\n",
		Deps: []*taskfile.Dep{
			{Task: "dependency"},
		},
		Cmds: []*taskfile.Cmd{
			{Cmd: "command"},
		},
	}

	summary.Print(&l, task)

	assert.Equal(t, expectedOutput(), buffer.String())
}

func expectedOutput() string {
	expected :=
		`task: sample-task
	
	line1
	line2
	line3
	
	dependencies:
	 - dependency
	
	commands:
	 - command
	`
	return expected
}
