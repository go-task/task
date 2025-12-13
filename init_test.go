package task_test

import (
	"os"
	"testing"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/filepathext"
)

func TestInitDir(t *testing.T) {
	t.Parallel()

	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Taskfile.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yml should not exist")
	}

	if _, err := task.InitTaskfile(dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.yml should exist")
	}

	_ = os.Remove(file)
}

func TestInitDirWithCustomDefaultName(t *testing.T) {
	const dir = "testdata/init"

	// Set environment variable before running the test
	t.Setenv("TASKFILE_DEFAULT_NAME", "Taskfile.yaml")

	file := filepathext.SmartJoin(dir, "Taskfile.yaml")
	defaultFile := filepathext.SmartJoin(dir, "Taskfile.yml")

	// Clean up any existing files
	_ = os.Remove(file)
	_ = os.Remove(defaultFile)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yaml should not exist")
	}

	// Manually call init logic
	task.SetDefaultFilename("Taskfile.yaml")
	defer task.SetDefaultFilename("Taskfile.yml")

	if _, err := task.InitTaskfile(dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.yaml should exist")
	}

	_ = os.Remove(file)
}

func TestInitFile(t *testing.T) {
	t.Parallel()

	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Tasks.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Tasks.yml should not exist")
	}

	if _, err := task.InitTaskfile(file); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Tasks.yml should exist")
	}
	_ = os.Remove(file)
}
