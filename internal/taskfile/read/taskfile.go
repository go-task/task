package read

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-task/task/v2/internal/taskfile"
)

// ErrIncludedTaskfilesCantHaveIncludes is returned when a included Taskfile contains includes
var ErrIncludedTaskfilesCantHaveIncludes = errors.New(
	`task: Included Taskfiles can't have includes. 
		Please, move the include to the main Taskfile`,
)

// Taskfile reads a Taskfile for a given directory
func Taskfile(dir string) (*taskfile.Taskfile, error) {
	path := filepath.Join(dir, "Taskfile.yml")
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf(`No Taskfile.yml found. Use "task --init" to create a new one`)
	}
	t, err := taskfile.LoadFromPath(path)
	if err != nil {
		return nil, err
	}

	if err := t.ProcessIncludes(dir); err != nil {
		return nil, err
	}

	path = filepath.Join(dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
	if _, err = os.Stat(path); err == nil {
		osTaskfile, err := taskfile.LoadFromPath(path)
		if err != nil {
			return nil, err
		}
		if err = taskfile.Merge(nil, t, osTaskfile); err != nil {
			return nil, err
		}
	}

	for name, task := range t.Tasks {
		task.Task = name
	}

	return t, nil
}
