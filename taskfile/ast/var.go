package ast

import (
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

// Var represents either a static or dynamic variable.
type Var struct {
	Value       any
	Live        any
	Sh          *string
	Interactive Interactive
	Ref         string
	Dir         string
}

type Interactive struct {
	Enabled bool
	Default any
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		key := node.Content[0].Value
		switch key {
		case "sh", "ref", "map", "interactive":
			var m struct {
				Sh          *string
				Interactive Interactive
				Ref         string
				Map         any
			}
			if err := node.Decode(&m); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			v.Sh = m.Sh
			v.Interactive = m.Interactive
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
