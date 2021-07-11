package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/status"
	"github.com/go-task/task/v3/taskfile"
)

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(ctx context.Context, calls ...taskfile.Call) error {
	for _, call := range calls {
		t, err := e.CompiledTask(call)
		if err != nil {
			return err
		}
		isUpToDate, err := e.isTaskUpToDate(ctx, t)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Name())
		}
	}
	return nil
}

func (e *Executor) isTaskUpToDate(ctx context.Context, t *taskfile.Task) (bool, error) {
	if len(t.Status) == 0 && len(t.Sources) == 0 {
		return false, nil
	}

	if len(t.Status) > 0 {
		isUpToDate, err := e.isTaskUpToDateStatus(ctx, t)
		if err != nil {
			return false, err
		}
		if !isUpToDate {
			return false, nil
		}
	}

	if len(t.Sources) > 0 {
		checker, err := e.getStatusChecker(t)
		if err != nil {
			return false, err
		}
		isUpToDate, err := checker.IsUpToDate()
		if err != nil {
			return false, err
		}
		if !isUpToDate {
			return false, nil
		}
	}

	return true, nil
}

func (e *Executor) statusOnError(t *taskfile.Task) error {
	checker, err := e.getStatusChecker(t)
	if err != nil {
		return err
	}
	return checker.OnError()
}

func (e *Executor) getStatusChecker(t *taskfile.Task) (status.Checker, error) {
	method := t.Method
	if method == "" {
		method = e.Taskfile.Method
	}
	switch method {
	case "timestamp":
		return e.timestampChecker(t), nil
	case "checksum":
		return e.checksumChecker(t), nil
	case "none":
		return status.None{}, nil
	default:
		return nil, fmt.Errorf(`task: invalid method "%s"`, method)
	}
}

func (e *Executor) timestampChecker(t *taskfile.Task) status.Checker {
	return &status.Timestamp{
		Dir:       t.Dir,
		Sources:   t.Sources,
		Generates: t.Generates,
	}
}

func (e *Executor) checksumChecker(t *taskfile.Task) status.Checker {
	return &status.Checksum{
		BaseDir:   e.Dir,
		TaskDir:   t.Dir,
		Task:      t.Name(),
		Sources:   t.Sources,
		Generates: t.Generates,
		Dry:       e.Dry,
	}
}

func (e *Executor) isTaskUpToDateStatus(ctx context.Context, t *taskfile.Task) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(ctx, &execext.RunCommandOptions{
			Command: s,
			Dir:     t.Dir,
			Env:     getEnviron(t),
		})
		if err != nil {
			e.Logger.VerboseOutf(logger.Yellow, "task: status command %s exited non-zero: %s", s, err)
			return false, nil
		}
		e.Logger.VerboseOutf(logger.Yellow, "task: status command %s exited zero", s)
	}
	return true, nil
}
