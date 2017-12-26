package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/internal/execext"
	"github.com/go-task/task/internal/status"
)

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(calls ...Call) error {
	for _, call := range calls {
		t, ok := e.Tasks[call.Task]
		if !ok {
			return &taskNotFoundError{taskName: call.Task}
		}
		isUpToDate, err := t.isUpToDate(e.Context)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Task)
		}
	}
	return nil
}

func (t *Task) isUpToDate(ctx context.Context) (bool, error) {
	if len(t.Status) > 0 {
		return t.isUpToDateStatus(ctx)
	}

	checker, err := t.getStatusChecker()
	if err != nil {
		return false, err
	}

	return checker.IsUpToDate()
}

func (t *Task) statusOnError() error {
	checker, err := t.getStatusChecker()
	if err != nil {
		return err
	}
	return checker.OnError()
}

func (t *Task) getStatusChecker() (status.Checker, error) {
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

func (t *Task) isUpToDateStatus(ctx context.Context) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     t.Dir,
			Env:     t.getEnviron(),
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}
