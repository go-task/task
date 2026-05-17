package task

import (
	_ "embed"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fsext"
	"github.com/go-task/task/v3/taskfile"
)

const defaultFilename = "Taskfile.yml"

//go:embed taskfile/templates/default.yml
var DefaultTaskfile string

// InitTaskfile creates a new Taskfile at path.
//
// path can be either a file path or a directory path.
// If path is a directory, path/Taskfile.yml will be created.
//
// If the TASK_INIT_DIR environment variable is set, the template will be
// read from that location instead of using the default embedded template.
// TASK_INIT_DIR can be a file path or a directory (searched using the same
// logic as calling task).
//
// The final file path is always returned and may be different from the input path.
func InitTaskfile(path string) (string, error) {
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		return path, errors.TaskfileAlreadyExistsError{}
	}

	if info != nil && info.IsDir() {
		// path was a directory, check if there is a Taskfile already
		if hasExistingTaskfile(path) {
			return path, errors.TaskfileAlreadyExistsError{}
		}
		path = filepathext.SmartJoin(path, defaultFilename)
	}

	// Check for TASK_INIT_DIR environment variable
	initDir := env.GetTaskEnv("INIT_DIR")
	if initDir == "" {
		// No override specified, use the default embedded template
		if err := os.WriteFile(path, []byte(DefaultTaskfile), 0o644); err != nil { //nolint:gosec
			return path, err
		}
		return path, nil
	}

	// Expand shell symbols like ~ and environment variables
	initDir, err = execext.ExpandLiteral(initDir)
	if err != nil {
		return path, err
	}

	// Use the same search logic as calling task:
	// - If initDir is a file, use it directly
	// - If initDir is a directory, search for a Taskfile in it
	templatePath, err := fsext.SearchPath(initDir, taskfile.DefaultTaskfiles)
	if err != nil {
		return path, err
	}

	// Read the template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return path, err
	}

	// Write the template to the destination
	return path, os.WriteFile(path, templateContent, 0o644) //nolint:gosec
}

func hasExistingTaskfile(dir string) bool {
	for _, name := range taskfile.DefaultTaskfiles {
		if _, err := os.Stat(filepathext.SmartJoin(dir, name)); err == nil {
			return true
		}
	}
	return false
}
