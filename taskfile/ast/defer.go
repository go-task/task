package ast

import (
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

type Defer struct {
	Cmd    string
	Task   string
	Vars   *Vars
	Silent bool
	When   string
}

func (d *Defer) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var cmd string
		if err := node.Decode(&cmd); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		d.Cmd = cmd
		return nil

	case yaml.MappingNode:
		var deferStruct struct {
			Defer  string
			Task   string
			Vars   *Vars
			Silent bool
			When   string
		}
		if err := node.Decode(&deferStruct); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		d.Cmd = deferStruct.Defer
		d.Task = deferStruct.Task
		d.Vars = deferStruct.Vars
		d.Silent = deferStruct.Silent
		d.When = deferStruct.When
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("defer")
}
