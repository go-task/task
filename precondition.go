// Package task provides ...
package task

import (
	"context"
	"errors"

	"github.com/go-task/task/v2/internal/execext"
	"github.com/go-task/task/v2/internal/taskfile"
)

var (
	// ErrNecessaryPreconditionFailed is returned when a precondition fails
	ErrNecessaryPreconditionFailed = errors.New("task: precondition not met")
	// ErrOptionalPreconditionFailed is returned when a precondition fails
	// that has ignore_error set to true
	ErrOptionalPreconditionFailed = errors.New("task: optional precondition not met")
)

func (e *Executor) areTaskPreconditionsMet(ctx context.Context, t *taskfile.Task) (bool, error) {
	var optionalPreconditionFailed bool
	for _, p := range t.Precondition {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: p.Sh,
			Dir:     t.Dir,
			Env:     getEnviron(t),
		})

		if err != nil {
			e.Logger.Outf(p.Msg)
			if p.IgnoreError == true {
				optionalPreconditionFailed = true
			} else {
				return false, ErrNecessaryPreconditionFailed
			}
		}
	}

	if optionalPreconditionFailed == true {
		return true, ErrOptionalPreconditionFailed
	}

	return true, nil
}
