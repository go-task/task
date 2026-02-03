package task

import (
	"context"
	"fmt"

	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/taskfile/ast"
)

// checkTaskStatus checks if a single task is up-to-date
func (e *Executor) checkTaskStatus(ctx context.Context, call *Call) (bool, *ast.Task, error) {
	// Compile the task
	t, err := e.CompiledTask(call)
	if err != nil {
		return false, nil, err
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
		return false, t, err
	}

	return isUpToDate, t, nil
}

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(ctx context.Context, calls ...*Call) error {
	if e.IncludeDeps {
		return e.statusWithDeps(ctx, calls...)
	}

	for _, call := range calls {
		isUpToDate, t, err := e.checkTaskStatus(ctx, call)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Name())
		}
	}
	return nil
}

// statusWithDeps checks if tasks and their dependencies are up-to-date
func (e *Executor) statusWithDeps(ctx context.Context, calls ...*Call) error {
	// Track visited tasks to avoid checking the same task multiple times
	visited := make(map[string]bool)
	var notUpToDate []string

	// Use traverse to recursively check all dependencies
	err := e.traverse(calls, func(t *ast.Task) error {
		// Skip if we've already checked this task
		if visited[t.Task] {
			return nil
		}
		visited[t.Task] = true

		// Check if the task is up-to-date
		isUpToDate, _, err := e.checkTaskStatus(ctx, &Call{Task: t.Task})
		if err != nil {
			return err
		}
		if !isUpToDate {
			notUpToDate = append(notUpToDate, t.Task)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// If any task is not up-to-date, return an error
	if len(notUpToDate) > 0 {
		if len(notUpToDate) == 1 {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, notUpToDate[0])
		}
		return fmt.Errorf(`task: Tasks are not up-to-date: %v`, notUpToDate)
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
