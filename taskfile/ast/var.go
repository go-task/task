package ast

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/omap"
)

// Vars is a string[string] variables map.
type Vars struct {
	omap.OrderedMap[string, Var]
}

// ToCacheMap converts Vars to a map containing only the static
// variables
func (vs *Vars) ToCacheMap() (m map[string]any) {
	m = make(map[string]any, vs.Len())
	_ = vs.Range(func(k string, v Var) error {
		if v.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			return nil
		}

		if v.Live != nil {
			m[k] = v.Live
		} else {
			m[k] = v.Value
		}
		return nil
	})
	return
}

// Wrapper around OrderedMap.Set to ensure we don't get nil pointer errors
func (vs *Vars) Range(f func(k string, v Var) error) error {
	if vs == nil {
		return nil
	}
	return vs.OrderedMap.Range(f)
}

// Wrapper around OrderedMap.Merge to ensure we don't get nil pointer errors
func (vs *Vars) Merge(other *Vars, include *Include) {
	if vs == nil || other == nil {
		return
	}
	_ = other.Range(func(key string, value Var) error {
		if include != nil && include.AdvancedImport {
			value.Dir = include.Dir
		}
		vs.Set(key, value)
		return nil
	})
}

// Wrapper around OrderedMap.Len to ensure we don't get nil pointer errors
func (vs *Vars) Len() int {
	if vs == nil {
		return 0
	}
	return vs.OrderedMap.Len()
}

// DeepCopy creates a new instance of Vars and copies
// data by value from the source struct.
func (vs *Vars) DeepCopy() *Vars {
	if vs == nil {
		return nil
	}
	return &Vars{
		OrderedMap: vs.OrderedMap.DeepCopy(),
	}
}

// Var represents either a static or dynamic variable.
type Var struct {
	Value any
	Live  any
	Sh    string
	Ref   string
	Dir   string
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	if experiments.MapVariables.Enabled {

		// This implementation is not backwards-compatible and replaces the 'sh' key with map variables
		if experiments.MapVariables.Value == "1" {
			var value any
			if err := node.Decode(&value); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}
			// If the value is a string and it starts with $, then it's a shell command
			if str, ok := value.(string); ok {
				if str, ok = strings.CutPrefix(str, "$"); ok {
					v.Sh = str
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
		if experiments.MapVariables.Value == "2" {
			switch node.Kind {
			case yaml.MappingNode:
				key := node.Content[0].Value
				switch key {
				case "sh", "ref", "map":
					var m struct {
						Sh  string
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
				Sh  string
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
