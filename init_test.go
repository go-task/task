package task_test

import (
	"os"
	"strings"
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

func TestInitTaskfileWithFullTemplate(t *testing.T) {
	t.Parallel()

	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Taskfile.full.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.full.yml should not exist")
	}

	if _, err := task.InitTaskfileWithTemplate(file, "full"); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.full.yml should exist")
	}

	// Verify content contains expected full template markers
	content, err := os.ReadFile(file)
	if err != nil {
		t.Error(err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "vars:") || !strings.Contains(contentStr, "tasks:") {
		t.Error("Content should contain expected template sections")
	}

	_ = os.Remove(file)
}

func TestInitTaskfileWithTemplate(t *testing.T) {
	t.Parallel()

	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Taskfile.default.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.default.yml should not exist")
	}

	if _, err := task.InitTaskfileWithTemplate(file, "default"); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.default.yml should exist")
	}

	_ = os.Remove(file)
}

func TestListTemplates(t *testing.T) {
	t.Parallel()

	templates := task.ListTemplates()
	if len(templates) < 2 {
		t.Errorf("Expected at least 2 templates, got %d", len(templates))
	}

	// Check that expected templates are present
	found := make(map[string]bool)
	for _, name := range templates {
		found[name] = true
	}

	if !found["default"] {
		t.Error("Expected 'default' template to be available")
	}
	if !found["full"] {
		t.Error("Expected 'full' template to be available")
	}
}

func TestInitTaskfileWithUnknownTemplate(t *testing.T) {
	t.Parallel()

	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Taskfile.unknown.yml")

	_, err := task.InitTaskfileWithTemplate(file, "nonexistent")
	if err == nil {
		t.Error("Expected error for unknown template")
	}
}
