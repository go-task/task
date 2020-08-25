package taskfile

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrCantUnmarshalVar is returned for invalid var YAML.
	ErrCantUnmarshalVar = errors.New("task: can't unmarshal var value")
)

// Vars is a string[string] variables map.
type Vars struct {
	Keys    []string
	Mapping map[string]Var
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (vs *Vars) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("task: vars is not a map")
	}

	// NOTE(@andreynering): on this style of custom unmarsheling,
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

// Merge merges the given Vars into the caller one
func (vs *Vars) Merge(other *Vars) {
	other.Range(func(key string, value Var) error {
		vs.Set(key, value)
		return nil
	})
}

// Set sets a value to a given key
func (vs *Vars) Set(key string, value Var) {
	if vs.Mapping == nil {
		vs.Mapping = make(map[string]Var, 1)
	}
	if !strSliceContains(vs.Keys, key) {
		vs.Keys = append(vs.Keys, key)
	}
	vs.Mapping[key] = value
}

func strSliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
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
func (vs *Vars) ToCacheMap() (m map[string]interface{}) {
	m = make(map[string]interface{}, len(vs.Keys))
	vs.Range(func(k string, v Var) error {
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

// Var represents either a static or dynamic variable.
type Var struct {
	Static string
	Live   interface{}
	Sh     string
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (v *Var) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		if strings.HasPrefix(str, "$") {
			v.Sh = strings.TrimPrefix(str, "$")
		} else {
			v.Static = str
		}
		return nil
	}

	var sh struct {
		Sh string
	}
	if err := unmarshal(&sh); err == nil {
		v.Sh = sh.Sh
		return nil
	}

	return ErrCantUnmarshalVar
}
