package taskfile

import "github.com/go-task/task/v3/internal/deepcopy"

// Requires represents a set of required variables necessary for a task to run
type Requires struct {
	Vars []string
}

func (r *Requires) DeepCopy() *Requires {
	if r == nil {
		return nil
	}

	return &Requires{
		Vars: deepcopy.Slice(r.Vars),
	}
}
