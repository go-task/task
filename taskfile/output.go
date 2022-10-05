package taskfile

import (
	"fmt"
)

// Output of the Task output
type Output struct {
	// Name of the Output.
	Name string `yaml:"-" json:"name"`
	// Group specific style
	Group OutputGroup `json:"group"`
}

// IsSet returns true if and only if a custom output style is set.
func (s *Output) IsSet() bool {
	return s.Name != ""
}

// UnmarshalYAML implements yaml.Unmarshaler
// It accepts a scalar node representing the Output.Name or a mapping node representing the OutputGroup.
func (s *Output) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err == nil {
		s.Name = name
		return nil
	}
	var tmp struct {
		Group *OutputGroup
	}
	if err := unmarshal(&tmp); err != nil {
		return fmt.Errorf("task: output style must be a string or mapping with a \"group\" key: %w", err)
	}
	if tmp.Group == nil {
		return fmt.Errorf("task: output style must have the \"group\" key when in mapping form")
	}
	*s = Output{
		Name:  "group",
		Group: *tmp.Group,
	}
	return nil
}

// OutputGroup is the style options specific to the Group style.
type OutputGroup struct {
	Begin, End string
}

// IsSet returns true if and only if a custom output style is set.
func (g *OutputGroup) IsSet() bool {
	if g == nil {
		return false
	}
	return g.Begin != "" || g.End != ""
}
