package ast

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/errors"
)

// Precondition represents a precondition necessary for a task to run
type Precondition struct {
	Sh      string
	Msg     string
	Inherit bool
}

func (p *Precondition) DeepCopy() *Precondition {
	if p == nil {
		return nil
	}
	return &Precondition{
		Sh:      p.Sh,
		Msg:     p.Msg,
		Inherit: p.Inherit,
	}
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (p *Precondition) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	case yaml.ScalarNode:
		var cmd string
		if err := node.Decode(&cmd); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		p.Sh = cmd
		p.Msg = fmt.Sprintf("`%s` failed", cmd)
		return nil

	case yaml.MappingNode:
		var sh struct {
			Sh      string
			Msg     string
			Inherit bool
		}
		if err := node.Decode(&sh); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		p.Sh = sh.Sh
		p.Msg = sh.Msg
		p.Inherit = sh.Inherit
		if p.Msg == "" {
			p.Msg = fmt.Sprintf("%s failed", sh.Sh)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("precondition")
}
