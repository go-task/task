package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Dep is a task dependency
type Dep struct {
	Task   string
	Vars   *Vars
	Silent bool
}

func (d *Dep) DeepCopy() *Dep {
	if d == nil {
		return nil
	}
	return &Dep{
		Task: d.Task,
		Vars: d.Vars.DeepCopy(),
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
			Vars   *Vars
			Silent bool
		}
		if err := node.Decode(&taskCall); err != nil {
			return err
		}
		d.Task = taskCall.Task
		d.Vars = taskCall.Vars
		d.Silent = taskCall.Silent
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into dependency", node.ShortTag())
}
