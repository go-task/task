package summary_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/summary"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestPrintsDependenciesIfPresent(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Deps: []*ast.Dep{
			{Task: "dep1"},
			{Task: "dep2"},
			{Task: "dep3"},
		},
	}

	summary.PrintTask(&l, task)

	assert.Contains(t, buffer.String(), "\ndependencies:\n - dep1\n - dep2\n - dep3\n")
}

func createDummyLogger() (*bytes.Buffer, logger.Logger) {
	buffer := &bytes.Buffer{}
	l := logger.Logger{
		Stderr:  buffer,
		Stdout:  buffer,
		Verbose: false,
	}
	return buffer, l
}

func TestDoesNotPrintDependenciesIfMissing(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Deps: []*ast.Dep{},
	}

	summary.PrintTask(&l, task)

	assert.NotContains(t, buffer.String(), "dependencies:")
}

func TestPrintTaskName(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Task: "my-task-name",
	}

	summary.PrintTask(&l, task)

	assert.Contains(t, buffer.String(), "task: my-task-name\n")
}

func TestPrintTaskCommandsIfPresent(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Cmds: []*ast.Cmd{
			{Cmd: "command-1"},
			{Cmd: "command-2"},
			{Task: "task-1"},
		},
	}

	summary.PrintTask(&l, task)

	assert.Contains(t, buffer.String(), "\ncommands:\n")
	assert.Contains(t, buffer.String(), "\n - command-1\n")
	assert.Contains(t, buffer.String(), "\n - command-2\n")
	assert.Contains(t, buffer.String(), "\n - Task: task-1\n")
}

func TestDoesNotPrintCommandIfMissing(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Cmds: []*ast.Cmd{},
	}

	summary.PrintTask(&l, task)

	assert.NotContains(t, buffer.String(), "commands")
}

func TestLayout(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	task := &ast.Task{
		Task:    "sample-task",
		Summary: "line1\nline2\nline3\n",
		Deps: []*ast.Dep{
			{Task: "dependency"},
		},
		Cmds: []*ast.Cmd{
			{Cmd: "command"},
		},
	}

	summary.PrintTask(&l, task)

	assert.Equal(t, expectedOutput(), buffer.String())
}

func expectedOutput() string {
	expected := `task: sample-task

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

func TestPrintDescriptionAsFallback(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()
	taskWithoutSummary := &ast.Task{
		Desc: "description",
	}

	taskWithSummary := &ast.Task{
		Desc:    "description",
		Summary: "summary",
	}
	taskWithoutSummaryOrDescription := &ast.Task{}

	summary.PrintTask(&l, taskWithoutSummary)

	assert.Contains(t, buffer.String(), "description")

	buffer.Reset()
	summary.PrintTask(&l, taskWithSummary)

	assert.NotContains(t, buffer.String(), "description")

	buffer.Reset()
	summary.PrintTask(&l, taskWithoutSummaryOrDescription)

	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n")
}

func TestPrintAllWithSpaces(t *testing.T) {
	t.Parallel()

	buffer, l := createDummyLogger()

	t1 := &ast.Task{Task: "t1"}
	t2 := &ast.Task{Task: "t2"}
	t3 := &ast.Task{Task: "t3"}

	tasks := ast.NewTasks()
	tasks.Set("t1", t1)
	tasks.Set("t2", t2)
	tasks.Set("t3", t3)

	summary.PrintTasks(&l,
		&ast.Taskfile{Tasks: tasks},
		[]string{"t1", "t2", "t3"},
	)

	assert.True(t, strings.HasPrefix(buffer.String(), "task: t1"))
	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n\n\ntask: t2")
	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n\n\ntask: t3")
}
