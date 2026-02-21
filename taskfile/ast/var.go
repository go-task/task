package ast

import (
	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/errors"
)

// Var represents either a static or dynamic variable.
type Var struct {
	Value  any
	Live   any
	Sh     *string
	Ref    string
	Dir    string
	Secret bool
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		key := "<none>"
		if len(node.Content) > 0 {
			key = node.Content[0].Value
		}
		switch key {
		case "sh", "ref", "map", "value":
			var m struct {
				Sh     *string
				Ref    string
				Map    any
				Value  any
				Secret bool
			}
			if err := node.Decode(&m); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			v.Sh = m.Sh
			v.Ref = m.Ref
			v.Secret = m.Secret
			// Handle both "map" and "value" keys
			if m.Map != nil {
				v.Value = m.Map
			} else if m.Value != nil {
				v.Value = m.Value
			}
			return nil
		default:
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`%q is not a valid variable type. Try "sh", "ref", "map", "value" or using a scalar value`, key)
		}
	default:
		var value any
		if err := node.Decode(&value); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		v.Value = value
		return nil
	}
}
