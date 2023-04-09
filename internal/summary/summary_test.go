package summary_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/log"
	"github.com/go-task/task/v3/internal/summary"
	"github.com/go-task/task/v3/taskfile"
)

func TestPrintsDependenciesIfPresent(t *testing.T) {
	buffer := createDummyLogger()
	task := &taskfile.Task{
		Deps: []*taskfile.Dep{
			{Task: "dep1"},
			{Task: "dep2"},
			{Task: "dep3"},
		},
	}

	summary.PrintTask(task)

	assert.Contains(t, buffer.String(), "\ndependencies:\n - dep1\n - dep2\n - dep3\n")
}

func createDummyLogger() *bytes.Buffer {
	buffer := &bytes.Buffer{}
	log.SetStderr(buffer)
	log.SetStdout(buffer)
	log.SetVerbose(false)
	return buffer
}

func TestDoesNotPrintDependenciesIfMissing(t *testing.T) {
	buffer := createDummyLogger()
	task := &taskfile.Task{
		Deps: []*taskfile.Dep{},
	}

	summary.PrintTask(task)

	assert.NotContains(t, buffer.String(), "dependencies:")
}

func TestPrintTaskName(t *testing.T) {
	buffer := createDummyLogger()
	task := &taskfile.Task{
		Task: "my-task-name",
	}

	summary.PrintTask(task)

	assert.Contains(t, buffer.String(), "task: my-task-name\n")
}

func TestPrintTaskCommandsIfPresent(t *testing.T) {
	buffer := createDummyLogger()
	task := &taskfile.Task{
		Cmds: []*taskfile.Cmd{
			{Cmd: "command-1"},
			{Cmd: "command-2"},
			{Task: "task-1"},
		},
	}

	summary.PrintTask(task)

	assert.Contains(t, buffer.String(), "\ncommands:\n")
	assert.Contains(t, buffer.String(), "\n - command-1\n")
	assert.Contains(t, buffer.String(), "\n - command-2\n")
	assert.Contains(t, buffer.String(), "\n - Task: task-1\n")
}

func TestDoesNotPrintCommandIfMissing(t *testing.T) {
	buffer := createDummyLogger()
	task := &taskfile.Task{
		Cmds: []*taskfile.Cmd{},
	}

	summary.PrintTask(task)

	assert.NotContains(t, buffer.String(), "commands")
}

func TestLayout(t *testing.T) {
	buffer := createDummyLogger()
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

	summary.PrintTask(task)

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
	buffer := createDummyLogger()
	taskWithoutSummary := &taskfile.Task{
		Desc: "description",
	}

	taskWithSummary := &taskfile.Task{
		Desc:    "description",
		Summary: "summary",
	}
	taskWithoutSummaryOrDescription := &taskfile.Task{}

	summary.PrintTask(taskWithoutSummary)

	assert.Contains(t, buffer.String(), "description")

	buffer.Reset()
	summary.PrintTask(taskWithSummary)

	assert.NotContains(t, buffer.String(), "description")

	buffer.Reset()
	summary.PrintTask(taskWithoutSummaryOrDescription)

	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n")
}

func TestPrintAllWithSpaces(t *testing.T) {
	buffer := createDummyLogger()

	t1 := &taskfile.Task{Task: "t1"}
	t2 := &taskfile.Task{Task: "t2"}
	t3 := &taskfile.Task{Task: "t3"}

	tasks := taskfile.Tasks{}
	tasks.Set("t1", t1)
	tasks.Set("t2", t2)
	tasks.Set("t3", t3)

	summary.PrintTasks(
		&taskfile.Taskfile{Tasks: tasks},
		[]taskfile.Call{{Task: "t1"}, {Task: "t2"}, {Task: "t3"}},
	)

	assert.True(t, strings.HasPrefix(buffer.String(), "task: t1"))
	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n\n\ntask: t2")
	assert.Contains(t, buffer.String(), "\n(task does not have description or summary)\n\n\ntask: t3")
}
