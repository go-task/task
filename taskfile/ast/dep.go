package ast

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Dep is a task dependency
type Dep struct {
	Task   string
	For    *For
	Vars   *Vars
	Silent bool
}

func (d *Dep) DeepCopy() *Dep {
	if d == nil {
		return nil
	}
	return &Dep{
		Task:   d.Task,
		For:    d.For.DeepCopy(),
		Vars:   d.Vars.DeepCopy(),
		Silent: d.Silent,
	}
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
			Task   string
			For    *For
			Vars   *Vars
			Silent bool
		}
		if err := node.Decode(&taskCall); err != nil {
			return err
		}
		d.Task = taskCall.Task
		d.For = taskCall.For
		d.Vars = taskCall.Vars
		d.Silent = taskCall.Silent
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into dependency", node.ShortTag())
}
