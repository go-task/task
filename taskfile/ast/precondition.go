package ast

import (
	"fmt"
	"github.com/go-task/task/v3/internal/deepcopy"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

// Precondition represents a precondition necessary for a task to run
type (
	Preconditions struct {
		preconditions []*Precondition
		mutex         sync.RWMutex
	}

	Precondition struct {
		Sh  string
		Msg string
	}
)

func (p *Preconditions) DeepCopy() *Preconditions {
	if p == nil {
		return nil
	}
	defer p.mutex.RUnlock()
	p.mutex.RLock()
	return &Preconditions{
		preconditions: deepcopy.Slice(p.preconditions),
	}
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

func NewPreconditions() *Preconditions {
	return &Preconditions{
		preconditions: make([]*Precondition, 0),
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
			Sh  string
			Msg string
		}
		if err := node.Decode(&sh); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		p.Sh = sh.Sh
		p.Msg = sh.Msg
		if p.Msg == "" {
			p.Msg = fmt.Sprintf("%s failed", sh.Sh)
		}
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("precondition")
}

func (p *Preconditions) UnmarshalYAML(node *yaml.Node) error {

	if p == nil || p.preconditions == nil {
		*p = *NewPreconditions()
	}

	if err := node.Decode(&p.preconditions); err != nil {
		return errors.NewTaskfileDecodeError(err, node).WithTypeMessage("precondition")
	}

	return nil
}
