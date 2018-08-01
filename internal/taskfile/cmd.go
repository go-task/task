package taskfile

import (
	"errors"
	"strings"
)

// Cmd is a task command
type Cmd struct {
	Cmd         string
	Silent      bool
	Task        string
	Vars        Vars
	IgnoreError bool
}

// Dep is a task dependency
type Dep struct {
	Task string
	Vars Vars
}

var (
	// ErrCantUnmarshalCmd is returned for invalid command YAML
	ErrCantUnmarshalCmd = errors.New("task: can't unmarshal cmd value")
	// ErrCantUnmarshalDep is returned for invalid dependency YAML
	ErrCantUnmarshalDep = errors.New("task: can't unmarshal dep value")
)

// UnmarshalYAML implements yaml.Unmarshaler interface
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
	var cmdStruct struct {
		Cmd         string
		Silent      bool
		IgnoreError bool `yaml:"ignore_error"`
	}
	if err := unmarshal(&cmdStruct); err == nil && cmdStruct.Cmd != "" {
		c.Cmd = cmdStruct.Cmd
		c.Silent = cmdStruct.Silent
		c.IgnoreError = cmdStruct.IgnoreError
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

// UnmarshalYAML implements yaml.Unmarshaler interface
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
