package taskfile

import (
	"github.com/go-task/task/v3/internal/deepcopy"
)

// RequiresStrict represents a set of required variables necessary for a task to run
type RequiresStrict struct {
	Vars        []string
	LimitValues map[string][]string `yaml:"limit_values"`
}

func (r *RequiresStrict) DeepCopy() *RequiresStrict {
	if r == nil {
		return nil
	}

	return &RequiresStrict{
		Vars:        deepcopy.Slice(r.Vars),
		LimitValues: deepcopy.Map(r.LimitValues),
	}
}
