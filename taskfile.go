package task

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	// TODO: version is still not used
	Version int
	Tasks   Tasks
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&tf.Tasks); err == nil {
		return nil
	}

	var taskfile struct {
		Version int
		Tasks   Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Tasks = taskfile.Tasks
	return nil
}

// ReadTaskfile parses Taskfile from the disk
func (e *Executor) ReadTaskfile() error {
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

func (e *Executor) readTaskfileData(path string) (*Taskfile, error) {
	if b, err := ioutil.ReadFile(path + ".yml"); err == nil {
		var taskfile Taskfile
		return &taskfile, yaml.UnmarshalStrict(b, &taskfile)
	}
	return nil, taskFileNotFound{path}
}

func (e *Executor) readTaskvars() error {
	var (
		file           = filepath.Join(e.Dir, TaskvarsFilePath)
		osSpecificFile = fmt.Sprintf("%s_%s", file, runtime.GOOS)
	)

	if b, err := ioutil.ReadFile(file + ".yml"); err == nil {
		if err := yaml.UnmarshalStrict(b, &e.taskvars); err != nil {
			return err
		}
	}

	if b, err := ioutil.ReadFile(osSpecificFile + ".yml"); err == nil {
		osTaskvars := make(Vars, 10)
		if err := yaml.UnmarshalStrict(b, &osTaskvars); err != nil {
			return err
		}
		for k, v := range osTaskvars {
			e.taskvars[k] = v
		}
	}
	return nil
}
