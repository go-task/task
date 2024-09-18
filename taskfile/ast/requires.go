package ast

import (
	"gopkg.in/yaml.v3"

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
	Name          string
	AllowedValues []string
}

func (v *VarsWithValidation) DeepCopy() *VarsWithValidation {
	if v == nil {
		return nil
	}
	return &VarsWithValidation{
		Name:          v.Name,
		AllowedValues: v.AllowedValues,
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
		v.AllowedValues = nil
		return nil

	case yaml.MappingNode:
		var vv struct {
			Name          string
			AllowedValues []string `yaml:"allowed_values"`
		}
		if err := node.Decode(&vv); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		v.Name = vv.Name
		v.AllowedValues = vv.AllowedValues
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("precondition")
}
