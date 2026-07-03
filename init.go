package task

import (
	_ "embed"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
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
// The final file path is always returned and may be different from the input path.
func InitTaskfile(path string) (string, error) {
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		return path, errors.TaskfileAlreadyExistsError{}
	}

	if info != nil && info.IsDir() {
		// path was a directory, check if there is a Taskfile already
		if hasDefaultTaskfile(path) {
			return path, errors.TaskfileAlreadyExistsError{}
		}
		path = filepathext.SmartJoin(path, defaultFilename)
	}

	if err := os.WriteFile(path, []byte(DefaultTaskfile), 0o644); err != nil {
		return path, err
	}
	return path, nil
}

func hasDefaultTaskfile(dir string) bool {
	for _, name := range taskfile.DefaultTaskfiles {
		if _, err := os.Stat(filepathext.SmartJoin(dir, name)); err == nil {
			return true
		}
	}
	return false
}
