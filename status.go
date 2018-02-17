package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/internal/execext"
	"github.com/go-task/task/internal/status"
	"github.com/go-task/task/internal/taskfile"
)

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(calls ...taskfile.Call) error {
	for _, call := range calls {
		t, ok := e.Taskfile.Tasks[call.Task]
		if !ok {
			return &taskNotFoundError{taskName: call.Task}
		}
		isUpToDate, err := isTaskUpToDate(e.Context, t)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Task)
		}
	}
	return nil
}

func isTaskUpToDate(ctx context.Context, t *taskfile.Task) (bool, error) {
	if len(t.Status) > 0 {
		return isTaskUpToDateStatus(ctx, t)
	}

	checker, err := getStatusChecker(t)
	if err != nil {
		return false, err
	}

	return checker.IsUpToDate()
}

func statusOnError(t *taskfile.Task) error {
	checker, err := getStatusChecker(t)
	if err != nil {
		return err
	}
	return checker.OnError()
}

func getStatusChecker(t *taskfile.Task) (status.Checker, error) {
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
		}, nil
	case "none":
		return status.None{}, nil
	default:
		return nil, fmt.Errorf(`task: invalid method "%s"`, t.Method)
	}
}

func isTaskUpToDateStatus(ctx context.Context, t *taskfile.Task) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     t.Dir,
			Env:     getEnviron(t),
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}
