package ast

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Dep is a task dependency
type Post struct {
	Task   string
	Vars   *Vars
	Silent bool
	Always bool
}

func (p *Post) DeepCopy() *Post {
	if p == nil {
		return nil
	}
	return &Post{
		Task:   p.Task,
		Vars:   p.Vars.DeepCopy(),
		Silent: p.Silent,
	}
}

func (p *Post) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var task string
		if err := node.Decode(&task); err != nil {
			return err
		}
		p.Task = task
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
		p.Task = taskCall.Task
		p.Vars = taskCall.Vars
		p.Silent = taskCall.Silent
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into dependency", node.ShortTag())
}
