package task

import (
	"context"
	"errors"
	"strings"

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

	missingVars := []string{}
	for _, requiredVar := range t.Requires {
		if !vars.Exists(requiredVar) {
			missingVars = append(missingVars, requiredVar)
		}
	}

	if len(missingVars) > 0 {
		return false, errors.New("required variables not set: " + strings.Join(missingVars, ","))
	}

	return true, nil
}
