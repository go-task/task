package ast

import (
	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
)

// Var represents either a static or dynamic variable.
type Var struct {
	Value  any
	Live   any
	Sh     *string
	Prompt *string
	Ref    string
	Dir    string
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		key := "<none>"
		if len(node.Content) > 0 {
			key = node.Content[0].Value
		}
		switch key {
		case "sh", "ref", "map", "prompt":
			var m struct {
				Sh     *string
				Prompt *string
				Ref    string
				Map    any
			}
			if err := node.Decode(&m); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			v.Sh = m.Sh
			v.Prompt = m.Prompt
			v.Ref = m.Ref
			v.Value = m.Map
			return nil
		default:
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`%q is not a valid variable type. Try "sh", "ref", "map", "interactive" or using a scalar value`, key)
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
