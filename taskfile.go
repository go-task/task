package task

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/go-task/task/internal/taskfile"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

// readTaskfile parses Taskfile from the disk
func (e *Executor) readTaskfile() error {
	path := filepath.Join(e.Dir, TaskFilePath)

	var err error
	e.Taskfile, err = e.readTaskfileData(path)
	if err != nil {
		return err
	}

	osTasks, err := e.readTaskfileData(fmt.Sprintf("%s_%s", path, runtime.GOOS))
	if err != nil {
		switch err.(type) {
		case taskFileNotFound:
		default:
			return err
		}
	} else {
		if err := mergo.MapWithOverwrite(&e.Taskfile.Tasks, osTasks.Tasks); err != nil {
			return err
		}
	}
	for name, task := range e.Taskfile.Tasks {
		task.Task = name
	}

	return e.readTaskvars()
}

func (e *Executor) readTaskfileData(path string) (*taskfile.Taskfile, error) {
	if b, err := ioutil.ReadFile(path + ".yml"); err == nil {
		var taskfile taskfile.Taskfile
		return &taskfile, yaml.Unmarshal(b, &taskfile)
	}
	return nil, taskFileNotFound{path}
}

func (e *Executor) readTaskvars() error {
	var (
		file           = filepath.Join(e.Dir, TaskvarsFilePath)
		osSpecificFile = fmt.Sprintf("%s_%s", file, runtime.GOOS)
	)

	if b, err := ioutil.ReadFile(file + ".yml"); err == nil {
		if err := yaml.Unmarshal(b, &e.taskvars); err != nil {
			return err
		}
	}

	if b, err := ioutil.ReadFile(osSpecificFile + ".yml"); err == nil {
		osTaskvars := make(taskfile.Vars, 10)
		if err := yaml.Unmarshal(b, &osTaskvars); err != nil {
			return err
		}
		for k, v := range osTaskvars {
			e.taskvars[k] = v
		}
	}
	return nil
}
