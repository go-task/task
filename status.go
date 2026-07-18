package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/v3/taskfile/ast"
)

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(ctx context.Context, calls ...*Call) error {
	for _, call := range calls {
		t, err := e.CompiledTask(call)
		if err != nil {
			return err
		}

		isUpToDate, err := e.fingerprinter().UpToDate(ctx, t)
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
	return e.fingerprinter().OnError(t)
}
