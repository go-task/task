package task

import (
	"context"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile"
)

func (e *Executor) areTaskRequiredVarsSet(ctx context.Context, t *taskfile.Task, call taskfile.Call) (bool, error) {
	if len(t.Requires) == 0 {
		return true, nil
	}

	vars, err := e.Compiler.GetVariables(t, call)
	if err != nil {
		return false, err
	}

	var missingVars []string
	for _, requiredVar := range t.Requires {
		if !vars.Exists(requiredVar) {
			missingVars = append(missingVars, requiredVar)
		}
	}

	if len(missingVars) > 0 {
		return false, &errors.TaskMissingRequiredVars{
			TaskName:    t.Name(),
			MissingVars: missingVars,
		}
	}

	return true, nil
}
