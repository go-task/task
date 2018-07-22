package read

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-task/task/internal/taskfile"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

// Taskfile reads a Taskfile for a given directory
func Taskfile(dir string) (*taskfile.Taskfile, error) {
	path := filepath.Join(dir, "Taskfile.yml")
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
		if err = mergo.MapWithOverwrite(&t.Tasks, osTaskfile.Tasks); err != nil {
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
