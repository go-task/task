package taskfile

import (
	"strings"
)

// Cmd is a task command
type Cmd struct {
	Cmd         string
	Silent      bool
	Task        string
	Vars        *Vars
	IgnoreError bool
	Always      bool
}

// Dep is a task dependency
type Dep struct {
	Task string
	Vars *Vars
}

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
		Always      bool
	}
	if err := unmarshal(&cmdStruct); err == nil && cmdStruct.Cmd != "" {
		c.Cmd = cmdStruct.Cmd
		c.Silent = cmdStruct.Silent
		c.IgnoreError = cmdStruct.IgnoreError
		c.Always = cmdStruct.Always
		return nil
	}
	var taskCall struct {
		Task   string
		Vars   *Vars
		Always bool
	}
	if err := unmarshal(&taskCall); err != nil {
		return err
	}
	c.Task = taskCall.Task
	c.Vars = taskCall.Vars
	c.Always = taskCall.Always
	return nil
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
		Vars *Vars
	}
	if err := unmarshal(&taskCall); err != nil {
		return err
	}
	d.Task = taskCall.Task
	d.Vars = taskCall.Vars
	return nil
}
