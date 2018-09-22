package read

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-task/task/internal/taskfile"

	"gopkg.in/yaml.v2"
)

// Taskfile reads a Taskfile for a given directory
func Taskfile(dir string) (*taskfile.Taskfile, error) {
	path := filepath.Join(dir, "Taskfile.yml")
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf(`No Taskfile.yml found. Use "task --init" to create a new one`)
	}
	t, err := readTaskfile(path)
	if err != nil {
		return nil, err
	}

	path = filepath.Join(dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
	if _, err = os.Stat(path); err == nil {
		osTaskfile, err := readTaskfile(path)
		if err != nil {
			return nil, err
		}
		if err = taskfile.Merge(t, osTaskfile); err != nil {
			return nil, err
		}
	}

	for name, task := range t.Tasks {
		task.Task = name
	}

	return t, nil
}

func readTaskfile(file string) (*taskfile.Taskfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var t taskfile.Taskfile
	return &t, yaml.NewDecoder(f).Decode(&t)
}
