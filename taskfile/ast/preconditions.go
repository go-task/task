package ast

import (
	"sync"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"

	"gopkg.in/yaml.v3"
)

// Precondition represents a precondition necessary for a task to run
type (
	Preconditions struct {
		Values []*Precondition
		mutex  sync.RWMutex
	}
)

func NewPreconditions() *Preconditions {
	return &Preconditions{
		Values: make([]*Precondition, 0),
	}
}

func (p *Preconditions) DeepCopy() *Preconditions {
	if p == nil {
		return nil
	}
	defer p.mutex.RUnlock()
	p.mutex.RLock()
	return &Preconditions{
		Values: deepcopy.Slice(p.Values),
	}
}

func (p *Preconditions) Merge(other *Preconditions) {
	if p == nil || p.Values == nil || other == nil {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	other.mutex.RLock()
	defer other.mutex.RUnlock()

	p.Values = append(p.Values, deepcopy.Slice(other.Values)...)
}

func (p *Preconditions) UnmarshalYAML(node *yaml.Node) error {
	if p == nil || p.Values == nil {
		*p = *NewPreconditions()
	}

	if err := node.Decode(&p.Values); err != nil {
		return errors.NewTaskfileDecodeError(err, node).WithTypeMessage("preconditions")
	}

	return nil
}
