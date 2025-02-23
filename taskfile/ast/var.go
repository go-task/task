package ast

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
)

// Var represents either a static or dynamic variable.
type Var struct {
	Value any
	Live  any
	Sh    *string
	Ref   string
	Dir   string
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	if experiments.MapVariables.Enabled() {

		// This implementation is not backwards-compatible and replaces the 'sh' key with map variables
		if experiments.MapVariables.Value == 1 {
			var value any
			if err := node.Decode(&value); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			// If the value is a string and it starts with $, then it's a shell command
			if str, ok := value.(string); ok {
				if str, ok = strings.CutPrefix(str, "$"); ok {
					v.Sh = &str
					return nil
				}
				if str, ok = strings.CutPrefix(str, "#"); ok {
					v.Ref = str
					return nil
				}
			}
			v.Value = value
			return nil
		}

		// This implementation IS backwards-compatible and keeps the 'sh' key and allows map variables to be added under the `map` key
		if experiments.MapVariables.Value == 2 {
			switch node.Kind {
			case yaml.MappingNode:
				key := node.Content[0].Value
				switch key {
				case "sh", "ref", "map":
					var m struct {
						Sh  *string
						Ref string
						Map any
					}
					if err := node.Decode(&m); err != nil {
						return errors.NewTaskfileDecodeError(err, node)
					}
					v.Sh = m.Sh
					v.Ref = m.Ref
					v.Value = m.Map
					return nil
				default:
					return errors.NewTaskfileDecodeError(nil, node).WithMessage(`%q is not a valid variable type. Try "sh", "ref", "map" or using a scalar value`, key)
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
	}

	switch node.Kind {

	case yaml.MappingNode:
		key := node.Content[0].Value
		switch key {
		case "sh", "ref":
			var m struct {
				Sh  *string
				Ref string
			}
			if err := node.Decode(&m); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			v.Sh = m.Sh
			v.Ref = m.Ref
			return nil
		default:
			return errors.NewTaskfileDecodeError(nil, node).WithMessage("maps cannot be assigned to variables")
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
