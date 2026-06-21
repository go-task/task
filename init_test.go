package task_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestInitWithEnvDir(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "Taskfile.yml")

	// Create a template directory with a custom Taskfile
	templateDir := filepath.Join(tmpDir, "templates")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	customTemplate := `# Custom template
version: '3'
tasks:
  custom:
    cmds:
      - echo "custom"
`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "Taskfile.yml"), []byte(customTemplate), 0o644))

	// Set TASK_INIT_DIR to the template directory
	t.Setenv("TASK_INIT_DIR", templateDir)

	// Initialize the Taskfile
	_, err := task.InitTaskfile(tmpDir)
	require.NoError(t, err)

	// Read the created file and verify it matches the custom template
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, customTemplate, string(content))
}

func TestInitWithEnvDirFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "Taskfile.yml")

	// Create a template file directly
	templateFile := filepath.Join(tmpDir, "custom-template.yml")
	customTemplate := `# Direct file template
version: '3'
tasks:
  direct:
    cmds:
      - echo "direct"
`
	require.NoError(t, os.WriteFile(templateFile, []byte(customTemplate), 0o644))

	// Set TASK_INIT_DIR to the template file directly
	t.Setenv("TASK_INIT_DIR", templateFile)

	// Initialize the Taskfile
	_, err := task.InitTaskfile(tmpDir)
	require.NoError(t, err)

	// Read the created file and verify it matches the custom template
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, customTemplate, string(content))
}
