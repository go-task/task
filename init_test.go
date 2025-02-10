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
