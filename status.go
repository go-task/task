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

// traverseInheritOutdated traverses dependencies marked with inherit_outdated
// and calls yield for each one. Similar to traverse() but only follows inherit_outdated deps.
func (e *Executor) traverseInheritOutdated(ctx context.Context, task *ast.Task, visited map[string]bool, yield func(*ast.Task) error) error {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Avoid infinite loops
	if visited[task.Task] {
		return nil
	}
	visited[task.Task] = true

	// Only traverse dependencies marked with inherit_outdated
	for _, dep := range task.Deps {
		if !dep.InheritOutdated || dep.Task == "" {
			continue
		}

		// Compile the dependency task
		depTask, err := e.CompiledTask(&Call{Task: dep.Task, Vars: dep.Vars})
		if err != nil {
			return err
		}

		// Recursively traverse the dependency's inherit_outdated deps
		if err := e.traverseInheritOutdated(ctx, depTask, visited, yield); err != nil {
			return err
		}

		// Yield this dependency
		if err := yield(depTask); err != nil {
			return err
		}
	}

	return nil
}

// Status returns an error if any the of given tasks is not up-to-date
func (e *Executor) Status(ctx context.Context, calls ...*Call) error {
	for _, call := range calls {
		// Check if the task itself is up-to-date
		isUpToDate, t, err := e.checkTaskStatus(ctx, call)
		if err != nil {
			return err
		}
		if !isUpToDate {
			return fmt.Errorf(`task: Task "%s" is not up-to-date`, t.Name())
		}

		// Check dependencies with inherit_outdated attribute
		var outdatedDep string
		err = e.traverseInheritOutdated(ctx, t, nil, func(depTask *ast.Task) error {
			isUpToDate, _, err := e.checkTaskStatus(ctx, &Call{Task: depTask.Task})
			if err != nil {
				return err
			}
			if !isUpToDate && outdatedDep == "" {
				outdatedDep = depTask.Task
			}
			return nil
		})
		if err != nil {
			return err
		}
		if outdatedDep != "" {
			return fmt.Errorf(`task: Task "%s" is not up-to-date because dependency "%s" is not up-to-date`, t.Name(), outdatedDep)
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
