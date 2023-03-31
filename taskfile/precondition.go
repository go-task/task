package taskfile

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ErrCantUnmarshalPrecondition is returned for invalid precond YAML.
var ErrCantUnmarshalPrecondition = errors.New("task: Can't unmarshal precondition value")

// Precondition represents a precondition necessary for a task to run
type Precondition struct {
	Sh  string
	Msg string
}

func (p *Precondition) DeepCopy() *Precondition {
	if p == nil {
		return nil
	}
	return &Precondition{
		Sh:  p.Sh,
		Msg: p.Msg,
	}
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (p *Precondition) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var cmd string
		if err := node.Decode(&cmd); err != nil {
			return err
		}
		p.Sh = cmd
		p.Msg = fmt.Sprintf("`%s` failed", cmd)
		return nil

	case yaml.MappingNode:
		var sh struct {
			Sh  string
			Msg string
		}
		if err := node.Decode(&sh); err != nil {
			return err
		}
		p.Sh = sh.Sh
		p.Msg = sh.Msg
		if p.Msg == "" {
			p.Msg = fmt.Sprintf("%s failed", sh.Sh)
		}
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into precondition", node.Line, node.ShortTag())
}
