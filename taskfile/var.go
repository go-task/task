package taskfile

import (
	"fmt"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// Vars is a string[string] variables map.
type Vars struct {
	Keys    []string
	Mapping map[string]Var
}

func (vs *Vars) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		// NOTE(@andreynering): on this style of custom unmarshalling,
		// even number contains the keys, while odd numbers contains
		// the values.
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			var v Var
			if err := valueNode.Decode(&v); err != nil {
				return err
			}
			vs.Set(keyNode.Value, v)
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variables", node.Line, node.ShortTag())
}

// DeepCopy creates a new instance of Vars and copies
// data by value from the source struct.
func (vs *Vars) DeepCopy() *Vars {
	if vs == nil {
		return nil
	}
	return &Vars{
		Keys:    deepCopySlice(vs.Keys),
		Mapping: deepCopyMap(vs.Mapping),
	}
}

// Merge merges the given Vars into the caller one
func (vs *Vars) Merge(other *Vars) {
	_ = other.Range(func(key string, value Var) error {
		vs.Set(key, value)
		return nil
	})
}

// Set sets a value to a given key
func (vs *Vars) Set(key string, value Var) {
	if vs.Mapping == nil {
		vs.Mapping = make(map[string]Var, 1)
	}
	if !slices.Contains(vs.Keys, key) {
		vs.Keys = append(vs.Keys, key)
	}
	vs.Mapping[key] = value
}

// Range allows you to loop into the vars in its right order
func (vs *Vars) Range(yield func(key string, value Var) error) error {
	if vs == nil {
		return nil
	}
	for _, k := range vs.Keys {
		if err := yield(k, vs.Mapping[k]); err != nil {
			return err
		}
	}
	return nil
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
			m[k] = v.Static
		}
		return nil
	})
	return
}

// Len returns the size of the map
func (vs *Vars) Len() int {
	if vs == nil {
		return 0
	}
	return len(vs.Keys)
}

// Var represents either a static or dynamic variable.
type Var struct {
	Static string
	Live   any
	Sh     string
	Dir    string
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return err
		}
		v.Static = str
		return nil

	case yaml.MappingNode:
		var sh struct {
			Sh string
		}
		if err := node.Decode(&sh); err != nil {
			return err
		}
		v.Sh = sh.Sh
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into variable", node.Line, node.ShortTag())
}
