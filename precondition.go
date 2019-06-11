package task

import (
	"context"
	"errors"

	"github.com/go-task/task/v2/internal/execext"
	"github.com/go-task/task/v2/internal/taskfile"
)

var (
	// ErrPreconditionFailed is returned when a precondition fails
	ErrPreconditionFailed = errors.New("task: precondition not met")
)

func (e *Executor) areTaskPreconditionsMet(ctx context.Context, t *taskfile.Task) (bool, error) {
	for _, p := range t.Preconditions {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: p.Sh,
			Dir:     t.Dir,
			Env:     getEnviron(t),
		})

		if err != nil {
			e.Logger.Errf("task: %s", p.Msg)
			return false, ErrPreconditionFailed
		}
	}

	return true, nil
}
