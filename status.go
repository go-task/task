package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/v2/internal/execext"
	"github.com/go-task/task/v2/internal/status"
	"github.com/go-task/task/v2/internal/taskfile"
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
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Task)
		}
	}
	return nil
}

func (e *Executor) isTaskUpToDate(ctx context.Context, t *taskfile.Task) (bool, error) {
	if len(t.Status) > 0 {
		return e.isTaskUpToDateStatus(ctx, t)
	}

	checker, err := e.GetStatusChecker(t)
	if err != nil {
		return false, err
	}

	return checker.IsUpToDate()
}

func (e *Executor) statusOnError(t *taskfile.Task) error {
	checker, err := e.GetStatusChecker(t)
	if err != nil {
		return err
	}
	return checker.OnError()
}

func (e *Executor) GetStatusChecker(t *taskfile.Task) (status.Checker, error) {
	switch t.Method {
	case "", "timestamp":
		return &status.Timestamp{
			Dir:       t.Dir,
			Sources:   t.Sources,
			Generates: t.Generates,
		}, nil
	case "checksum":
		return &status.Checksum{
			Dir:     t.Dir,
			Task:    t.Task,
			Sources: t.Sources,
			Dry:     e.Dry,
		}, nil
	case "none":
		return status.None{}, nil
	default:
		return nil, fmt.Errorf(`task: invalid method "%s"`, t.Method)
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
			e.Logger.VerboseOutf("task: status command %s exited non-zero: %s", s, err)
			return false, nil
		}
		e.Logger.VerboseOutf("task: status command %s exited zero", s)
	}
	return true, nil
}
