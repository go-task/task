package task

import (
	"errors"
	"strings"
)

type Cmd struct {
	Cmd  string
	Task string
	Vars Vars
}

type Dep struct {
	Task string
	Vars Vars
}

var (
	ErrCantUnmarshalCmd = errors.New("task: can't unmarshal cmd value")
	ErrCantUnmarshalDep = errors.New("task: can't unmarshal dep value")
)

func (c *Cmd) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err == nil {
		if strings.HasPrefix(cmd, "^") {
			c.Task = strings.TrimPrefix(cmd, "^")
		} else {
			c.Cmd = cmd
		}
		return nil
	}
	var taskCall struct {
		Task string
		Vars Vars
	}
	if err := unmarshal(&taskCall); err == nil {
		c.Task = taskCall.Task
		c.Vars = taskCall.Vars
		return nil
	}
	return ErrCantUnmarshalCmd
}

func (d *Dep) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var task string
	if err := unmarshal(&task); err == nil {
		d.Task = task
		return nil
	}
	var taskCall struct {
		Task string
		Vars Vars
	}
	if err := unmarshal(&taskCall); err == nil {
		d.Task = taskCall.Task
		d.Vars = taskCall.Vars
		return nil
	}
	return ErrCantUnmarshalDep
}
