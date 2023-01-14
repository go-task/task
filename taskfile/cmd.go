package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Cmd is a task command
type Cmd struct {
	Cmd         string
	Silent      bool
	Task        string
	Set         []string
	Shopt       []string
	Vars        *Vars
	IgnoreError bool
	Defer       bool
	Platforms   []*Platform
}

// Dep is a task dependency
type Dep struct {
	Task string
	Vars *Vars
}

func (c *Cmd) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var cmd string
		if err := node.Decode(&cmd); err != nil {
			return err
		}
		c.Cmd = cmd
		return nil

	case yaml.MappingNode:

		// A command with additional options
		var cmdStruct struct {
			Cmd         string
			Silent      bool
			Set         []string
			Shopt       []string
			IgnoreError bool `yaml:"ignore_error"`
			Platforms   []*Platform
		}
		if err := node.Decode(&cmdStruct); err == nil && cmdStruct.Cmd != "" {
			c.Cmd = cmdStruct.Cmd
			c.Silent = cmdStruct.Silent
			c.Set = cmdStruct.Set
			c.Shopt = cmdStruct.Shopt
			c.IgnoreError = cmdStruct.IgnoreError
			c.Platforms = cmdStruct.Platforms
			return nil
		}

		// A deferred command
		var deferredCmd struct {
			Defer string
		}
		if err := node.Decode(&deferredCmd); err == nil && deferredCmd.Defer != "" {
			c.Defer = true
			c.Cmd = deferredCmd.Defer
			return nil
		}

		// A deferred task call
		var deferredCall struct {
			Defer Call
		}
		if err := node.Decode(&deferredCall); err == nil && deferredCall.Defer.Task != "" {
			c.Defer = true
			c.Task = deferredCall.Defer.Task
			c.Vars = deferredCall.Defer.Vars
			return nil
		}

		// A task call
		var taskCall struct {
			Task string
			Vars *Vars
		}
		if err := node.Decode(&taskCall); err == nil && taskCall.Task != "" {
			c.Task = taskCall.Task
			c.Vars = taskCall.Vars
			return nil
		}

		return fmt.Errorf("yaml: line %d: invalid keys in command", node.Line)
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into command", node.Line, node.ShortTag())
}

func (d *Dep) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var task string
		if err := node.Decode(&task); err != nil {
			return err
		}
		d.Task = task
		return nil

	case yaml.MappingNode:
		var taskCall struct {
			Task string
			Vars *Vars
		}
		if err := node.Decode(&taskCall); err != nil {
			return err
		}
		d.Task = taskCall.Task
		d.Vars = taskCall.Vars
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into dependency", node.ShortTag())
}
