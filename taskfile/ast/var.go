package ast

import (
	"strings"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
	"github.com/go-task/task/v3/internal/experiments"
)

// Vars is a string[string] variables map.
type Vars struct {
	om *orderedmap.OrderedMap[string, Var]
}

type VarElement orderedmap.Element[string, Var]

func NewVars(els ...*VarElement) *Vars {
	vs := &Vars{
		om: orderedmap.NewOrderedMap[string, Var](),
	}
	for _, el := range els {
		vs.Set(el.Key, el.Value)
	}
	return vs
}

func (vs *Vars) Len() int {
	if vs == nil || vs.om == nil {
		return 0
	}
	return vs.om.Len()
}

func (vs *Vars) Get(key string) (Var, bool) {
	if vs == nil || vs.om == nil {
		return Var{}, false
	}
	return vs.om.Get(key)
}

func (vs *Vars) Set(key string, value Var) bool {
	if vs == nil {
		vs = NewVars()
	}
	if vs.om == nil {
		vs.om = orderedmap.NewOrderedMap[string, Var]()
	}
	return vs.om.Set(key, value)
}

func (vs *Vars) Range(f func(k string, v Var) error) error {
	if vs == nil || vs.om == nil {
		return nil
	}
	for pair := vs.om.Front(); pair != nil; pair = pair.Next() {
		if err := f(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return nil
}

// ToCacheMap converts Vars to a map containing only the static
// variables
func (vs *Vars) ToCacheMap() (m map[string]any) {
	m = make(map[string]any, vs.Len())
	for pair := vs.om.Front(); pair != nil; pair = pair.Next() {
		if pair.Value.Sh != nil && *pair.Value.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			return nil
		}
		if pair.Value.Live != nil {
			m[pair.Key] = pair.Value.Live
		} else {
			m[pair.Key] = pair.Value.Value
		}
	}
	return
}

// Wrapper around OrderedMap.Merge to ensure we don't get nil pointer errors
func (vs *Vars) Merge(other *Vars, include *Include) {
	if vs == nil || vs.om == nil || other == nil {
		return
	}
	for pair := other.om.Front(); pair != nil; pair = pair.Next() {
		if include != nil && include.AdvancedImport {
			pair.Value.Dir = include.Dir
		}
		vs.om.Set(pair.Key, pair.Value)
	}
}

// DeepCopy creates a new instance of Vars and copies
// data by value from the source struct.
func (vs *Vars) DeepCopy() *Vars {
	if vs == nil {
		return nil
	}
	return &Vars{
		om: deepcopy.OrderedMap(vs.om),
	}
}

func (vs *Vars) UnmarshalYAML(node *yaml.Node) error {
	vs.om = orderedmap.NewOrderedMap[string, Var]()
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE: orderedmap does not have an unmarshaler, so we have to decode
		// the map manually. We increment over 2 values at a time and assign
		// them as a key-value pair.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Decode the value node into a Task struct
			var v Var
			if err := valueNode.Decode(&v); err != nil {
				return errors.NewTaskfileDecodeError(err, node)
			}

			// Add the task to the ordered map
			vs.Set(keyNode.Value, v)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("vars")
}

// Var represents either a static or dynamic variable.
type Var struct {
	Value any
	Live  any
	Sh    *string
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
		if experiments.MapVariables.Value == "2" {
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
