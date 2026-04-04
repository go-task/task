package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/taskfile/ast"
)

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(ctx context.Context, calls ...*Call) error {
	for _, call := range calls {

		// Compile the task
		t, err := e.CompiledTask(call)
		if err != nil {
			return err
		}

		// Get the fingerprinting method to use
		method := e.Taskfile.Method
		if t.Method != "" {
			method = t.Method
		}

		// Check if the task is up-to-date
		isUpToDate, err := fingerprint.IsTaskUpToDate(ctx, t,
			fingerprint.WithMethod(method),
			fingerprint.WithTempDir(e.TempDir.Fingerprint),
			fingerprint.WithDry(e.Dry),
			fingerprint.WithLogger(e.Logger),
		)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Name())
		}
	}
	return nil
}

func (e *Executor) statusOnError(t *ast.Task) error {
	method := t.Method
	if method == "" {
		method = e.Taskfile.Method
	}
	checker, err := fingerprint.NewSourcesChecker(method, e.TempDir.Fingerprint, e.Dry)
	if err != nil {
		return err
	}
	return checker.OnError(t)
}
