package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/log"
	"github.com/go-task/task/v3/taskfile"
)

type StatusChecker struct{}

func NewStatusChecker() StatusCheckable {
	return &StatusChecker{}
}

func (checker *StatusChecker) IsUpToDate(ctx context.Context, t *taskfile.Task) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: s,
			Dir:     t.Dir,
			Env:     env.Get(t),
		})
		if err != nil {
			log.VerboseOutf(log.Yellow, "task: status command %s exited non-zero: %s", s, err)
			return false, nil
		}
		log.VerboseOutf(log.Yellow, "task: status command %s exited zero", s)
	}
	return true, nil
}
