package ast

import (
	"go.yaml.in/yaml/v3"

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

// Enum represents an enum constraint for a required variable.
// It can either be a static list of values or a reference to another variable.
type Enum struct {
	Ref   string
	Value []string
}

func (e *Enum) DeepCopy() *Enum {
	if e == nil {
		return nil
	}
	return &Enum{
		Ref:   e.Ref,
		Value: deepcopy.Slice(e.Value),
	}
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (e *Enum) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		// Static list of values: enum: ["a", "b"]
		var values []string
		if err := node.Decode(&values); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		e.Value = values
		return nil

	case yaml.MappingNode:
		// Reference to another variable: enum: { ref: .VAR }
		var refStruct struct {
			Ref string
		}
		if err := node.Decode(&refStruct); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		e.Ref = refStruct.Ref
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("enum")
}

type VarsWithValidation struct {
	Name string
	Enum *Enum
}

func (v *VarsWithValidation) DeepCopy() *VarsWithValidation {
	if v == nil {
		return nil
	}
	return &VarsWithValidation{
		Name: v.Name,
		Enum: v.Enum.DeepCopy(),
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
			Enum *Enum
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
