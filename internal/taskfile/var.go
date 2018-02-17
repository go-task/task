package taskfile

import (
	"errors"
	"strings"
)

var (
	// ErrCantUnmarshalVar is returned for invalid var YAML.
	ErrCantUnmarshalVar = errors.New("task: can't unmarshal var value")
)

// Vars is a string[string] variables map.
type Vars map[string]Var

// ToStringMap converts Vars to a string map containing only the static
// variables
func (vs Vars) ToStringMap() (m map[string]string) {
	m = make(map[string]string, len(vs))
	for k, v := range vs {
		if v.Sh != "" {
			// Dynamic variable is not yet resolved; trigger
			// <no value> to be used in templates.
			continue
		}
		m[k] = v.Static
	}
	return
}

// Var represents either a static or dynamic variable.
type Var struct {
	Static string
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
