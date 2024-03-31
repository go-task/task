package ast

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
	// Prefix specific style
	Prefix OutputPrefix
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
			Group    *OutputGroup
			Prefixed *OutputPrefix
		}

		if err := node.Decode(&tmp); err != nil {
			return fmt.Errorf("task: output style must be a string or mapping with a \"group\" or \"prefixed\" key: %w", err)
		}

		if tmp.Group != nil {
			*s = Output{
				Name:  "group",
				Group: *tmp.Group,
			}

			return nil
		}

		if tmp.Prefixed != nil {
			*s = Output{
				Name:   "prefixed",
				Prefix: *tmp.Prefixed,
			}

			return nil
		}

		return fmt.Errorf("task: output style must have either \"group\" or \"prefixed\" key when in mapping form")
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

// OutputPrefix is the style options specific to the Prefix style.
type OutputPrefix struct {
	Color bool `yaml:"color"`
}
