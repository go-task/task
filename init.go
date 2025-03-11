package task

import (
	_ "embed"
	"os"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
)

const defaultTaskFilename = "Taskfile.yml"

//go:embed taskfile/templates/default.yml
var DefaultTaskfile string

// InitTaskfile creates a new Taskfile at path.
//
// path can be either a file path or a directory path.
// If path is a directory, path/Taskfile.yml will be created.
//
// The final file path is always returned and may be different from the input path.
func InitTaskfile(path string) (string, error) {
	fi, err := os.Stat(path)
	if err == nil && !fi.IsDir() {
		return path, errors.TaskfileAlreadyExistsError{}
	}

	if fi != nil && fi.IsDir() {
		path = filepathext.SmartJoin(path, defaultTaskFilename)
		// path was a directory, so check if Taskfile.yml exists in it
		if _, err := os.Stat(path); err == nil {
			return path, errors.TaskfileAlreadyExistsError{}
		}
	}

	if err := os.WriteFile(path, []byte(DefaultTaskfile), 0o644); err != nil {
		return path, err
	}

	return path, nil
}
