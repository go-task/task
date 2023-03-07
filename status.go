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
	var statusUpToDate bool
	var sourcesUpToDate bool
	var err error

	statusIsSet := len(t.Status) != 0
	sourcesIsSet := len(t.Sources) != 0

	// If status is set, check if it is up-to-date
	if statusIsSet {
		statusUpToDate, err = e.isTaskUpToDateStatus(ctx, t)
		if err != nil {
			return false, err
		}
	}

	// If sources is set, check if they are up-to-date
	if sourcesIsSet {
		checker, err := e.getStatusChecker(t)
		if err != nil {
			return false, err
		}
		sourcesUpToDate, err = checker.IsUpToDate()
		if err != nil {
			return false, err
		}
	}

	// If both status and sources are set, the task is up-to-date if both are up-to-date
	if statusIsSet && sourcesIsSet {
		return statusUpToDate && sourcesUpToDate, nil
	}

	// If only status is set, the task is up-to-date if the status is up-to-date
	if statusIsSet {
		return statusUpToDate, nil
	}

	// If only sources is set, the task is up-to-date if the sources are up-to-date
	if sourcesIsSet {
		return sourcesUpToDate, nil
	}

	// If no status or sources are set, the task should always run
	// i.e. it is never considered "up-to-date"
	return false, nil
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
		TempDir:   e.TempDir,
		Task:      t.Name(),
		Dir:       t.Dir,
		Sources:   t.Sources,
		Generates: t.Generates,
		Dry:       e.Dry,
	}
}

func (e *Executor) checksumChecker(t *taskfile.Task) status.Checker {
	return &status.Checksum{
		TempDir:   e.TempDir,
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
