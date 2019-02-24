package summary_test

import (
	"bytes"
	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/summary"
	"github.com/go-task/task/v2/internal/taskfile"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrintsDependencies(t *testing.T) {
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

func TestDoesNotPrintDependencies(t *testing.T) {
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
