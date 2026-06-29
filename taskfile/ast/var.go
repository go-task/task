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
		// Validate the keys regardless of their order: every key must be known
		// and at least one type-defining key must be present. "secret" is a
		// modifier, not a type, so it can appear in any position.
		hasType := false
		for i := 0; i+1 < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "sh", "ref", "map", "value":
				hasType = true
			case "secret":
				// modifier, not a type
			default:
				return errors.NewTaskfileDecodeError(nil, node).WithMessage(`%q is not a valid variable type. Try "sh", "ref", "map", "value" or using a scalar value`, node.Content[i].Value)
			}
		}
		if !hasType {
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`a variable must define one of "sh", "ref", "map", "value" or be a scalar value`)
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
		var value any
		if err := node.Decode(&value); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		v.Value = value
		return nil
	}
}
