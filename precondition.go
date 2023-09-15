package task

import (
	"context"
	"errors"

	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

// ErrPreconditionFailed is returned when a precondition fails
var ErrPreconditionFailed = errors.New("task: precondition not met")

func (e *Executor) areTaskPreconditionsMet(ctx context.Context, t *taskfile.Task) (bool, error) {
	for _, p := range t.Preconditions {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: p.Sh,
			Dir:     t.Dir,
			Env:     env.Get(t),
		})
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				e.Logger.Errf(logger.Magenta, "task: %s\n", p.Msg)
			}
			return false, ErrPreconditionFailed
		}
	}

	return true, nil
}
