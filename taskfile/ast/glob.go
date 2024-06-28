package ast

import (
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

type Glob struct {
	Glob   string
	Negate bool
}

func (g *Glob) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		g.Glob = node.Value
		return nil

	case yaml.MappingNode:
		var glob struct {
			Exclude string
		}
		if err := node.Decode(&glob); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		g.Glob = glob.Exclude
		g.Negate = true
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("glob")
}
