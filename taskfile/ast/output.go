package ast

import (
	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/errors"
)

// Output of the Task output
type Output struct {
	// Name of the Output.
	Name string `yaml:"-"`
	// Group specific style
	Group OutputGroup
	// TUI specific options
	TUI OutputTUI
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
			return errors.NewTaskfileDecodeError(err, node)
		}
		s.Name = name
		return nil

	case yaml.MappingNode:
		var tmp struct {
			Group *OutputGroup
			TUI   *OutputTUI
		}
		if err := node.Decode(&tmp); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		if tmp.Group != nil && tmp.TUI != nil {
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`output style must have only one of the "group" or "tui" keys`)
		}
		if tmp.Group != nil {
			*s = Output{
				Name:  "group",
				Group: *tmp.Group,
			}
			return nil
		}
		if tmp.TUI != nil {
			*s = Output{
				Name: "tui",
				TUI:  *tmp.TUI,
			}
			return nil
		}
		return errors.NewTaskfileDecodeError(nil, node).WithMessage(`output style must have the "group" or "tui" key when in mapping form`)
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("output")
}

// OutputGroup is the style options specific to the Group style.
type OutputGroup struct {
	Begin, End string
	ErrorOnly  bool `yaml:"error_only"`
}

// OutputTUI contains options specific to the TUI output style.
type OutputTUI struct {
	HideInternal bool   `yaml:"hide_internal"`
	Status       string `yaml:"status"`
}

// IsSet returns true if and only if a custom output style is set.
func (g *OutputGroup) IsSet() bool {
	if g == nil {
		return false
	}
	return g.Begin != "" || g.End != ""
}
