package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Output of the Task output
type Output struct {
	// Name of the Output.
	Name string `yaml:"-"`
	// Group specific style
	Group OutputGroup
}

// IsSet returns true if and only if a custom output style is set.
func (s *Output) IsSet() bool {
	return s.Name != ""
}

func (s *Output) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var name string
		if err := node.Decode(&name); err != nil {
			return err
		}
		s.Name = name
		return nil

	case yaml.MappingNode:
		var tmp struct {
			Group *OutputGroup
		}
		if err := node.Decode(&tmp); err != nil {
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

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into output", node.Line, node.ShortTag())
}

// OutputGroup is the style options specific to the Group style.
type OutputGroup struct {
	Begin, End string
	ErrorOnly  bool `yaml:"error_only"`
}

// IsSet returns true if and only if a custom output style is set.
func (g *OutputGroup) IsSet() bool {
	if g == nil {
		return false
	}
	return g.Begin != "" || g.End != ""
}
