package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"
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
			return err
		}
		g.Glob = glob.Exclude
		g.Negate = true
		return nil
	default:
		return fmt.Errorf("yaml: line %d: cannot unmarshal %s into task", node.Line, node.ShortTag())
	}
}
