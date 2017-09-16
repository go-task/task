package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/execext"
	"github.com/go-task/task/status"
)

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
