package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/taskfile/ast"
)

type StatusChecker struct {
	logger    *logger.Logger
	posixOpts []string
	bashOpts  []string
}

func NewStatusChecker(logger *logger.Logger, posixOpts []string, bashOpts []string) StatusCheckable {
	return &StatusChecker{
		logger:    logger,
		posixOpts: posixOpts,
		bashOpts:  bashOpts,
	}
}

func (checker *StatusChecker) IsUpToDate(ctx context.Context, t *ast.Task) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command:   s,
			Dir:       t.Dir,
			Env:       env.Get(t),
			PosixOpts: slicesext.UniqueJoin(checker.posixOpts, t.Set),
			BashOpts:  slicesext.UniqueJoin(checker.bashOpts, t.Shopt),
		})
		if err != nil {
			checker.logger.VerboseOutf(logger.Yellow, "task: status command %s exited non-zero: %s\n", s, err)
			return false, nil
		}
		checker.logger.VerboseOutf(logger.Yellow, "task: status command %s exited zero\n", s)
	}
	return true, nil
}
