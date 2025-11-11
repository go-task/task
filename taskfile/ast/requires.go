package ast

import (
	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

// Requires represents a set of required variables necessary for a task to run
type Requires struct {
	Vars []*VarsWithValidation
}

func (r *Requires) DeepCopy() *Requires {
	if r == nil {
		return nil
	}

	return &Requires{
		Vars: deepcopy.Slice(r.Vars),
	}
}

type VarsWithValidation struct {
	Name string
	Enum []string
}

func (v *VarsWithValidation) DeepCopy() *VarsWithValidation {
	if v == nil {
		return nil
	}
	return &VarsWithValidation{
		Name: v.Name,
		Enum: v.Enum,
	}
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (v *VarsWithValidation) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var cmd string
		if err := node.Decode(&cmd); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		v.Name = cmd
		v.Enum = nil
		return nil

	case yaml.MappingNode:
		var vv struct {
			Name string
			Enum []string
		}
		if err := node.Decode(&vv); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		v.Name = vv.Name
		v.Enum = vv.Enum
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("requires")
}
