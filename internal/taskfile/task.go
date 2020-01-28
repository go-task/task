package taskfile

import (
	"errors"
)

// Tasks represents a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task          string
	Image         string
	Cmds          []*Cmd
	Deps          []*Dep
	Desc          string
	Summary       string
	Sources       []string
	Generates     []string
	Status        []string
	Preconditions []*Precondition
	Dir           string
	Vars          Vars
	Env           Vars
	Silent        bool
	Method        string
	Prefix        string
	IgnoreError   bool
}

var (
	// ErrCantUnmarshalTask is returned for invalid task YAML
	ErrCantUnmarshalTask = errors.New("task: can't unmarshal task value")
)

func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd Cmd
	if err := unmarshal(&cmd); err == nil && cmd.Cmd != "" {
		t.Cmds = append(t.Cmds, &cmd)
		return nil
	}

	var cmds []*Cmd
	if err := unmarshal(&cmds); err == nil && len(cmds) > 0 {
		t.Cmds = cmds
		return nil
	}

	var task struct {
		Cmds          []*Cmd
		Image         string
		Deps          []*Dep
		Desc          string
		Summary       string
		Sources       []string
		Generates     []string
		Status        []string
		Preconditions []*Precondition
		Dir           string
		Vars          Vars
		Env           Vars
		Silent        bool
		Method        string
		Prefix        string
		IgnoreError   bool `yaml:"ignore_error"`
	}
	if err := unmarshal(&task); err == nil {
		t.Cmds = task.Cmds
		t.Image = task.Image
		t.Deps = task.Deps
		t.Desc = task.Desc
		t.Summary = task.Summary
		t.Sources = task.Sources
		t.Generates = task.Generates
		t.Status = task.Status
		t.Preconditions = task.Preconditions
		t.Dir = task.Dir
		t.Vars = task.Vars
		t.Env = task.Env
		t.Silent = task.Silent
		t.Method = task.Method
		t.Prefix = task.Prefix
		t.IgnoreError = task.IgnoreError

		return nil
	}

	return ErrCantUnmarshalTask
}
