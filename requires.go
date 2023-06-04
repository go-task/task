package task

import (
	"context"
	"errors"

	"github.com/go-task/task/v3/taskfile"
)

func (e *Executor) areTaskRequiredVarsSet(ctx context.Context, t *taskfile.Task, call taskfile.Call) (bool, error) {
	vars, err := e.Compiler.GetVariables(t, call)
	if err != nil {
		return false, err
	}

	for _, req := range t.Requires {
		if !vars.Exists(req) {
			return false, errors.New("required variable not set: " + req)
		}
	}

	return true, nil
}
