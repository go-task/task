package task

import (
	"golang.org/x/exp/slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile"
)

func (e *Executor) areTaskRequiredVarsSet(t *taskfile.Task, call taskfile.Call) error {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return nil
	}

	vars, err := e.Compiler.GetVariables(t, call)
	if err != nil {
		return err
	}

	var missingVars []string
	for _, requiredVar := range t.Requires.Vars {
		if !vars.Exists(requiredVar) {
			missingVars = append(missingVars, requiredVar)
		}
	}

	if len(missingVars) > 0 {
		return &errors.TaskMissingRequiredVars{
			TaskName:    t.Name(),
			MissingVars: missingVars,
		}
	}

	return nil
}

func (e *Executor) areTaskRequiredStrictVarsValid(t *taskfile.Task, call taskfile.Call) error {
	if t.RequiresStrict == nil || len(t.RequiresStrict.Vars) == 0 {
		return nil
	}

	vars, err := e.Compiler.GetVariables(t, call)
	if err != nil {
		return err
	}

	var missingVars []string
	var emptyVars []string
	var failedLimitVars []string
	for _, requiredVar := range t.RequiresStrict.Vars {
		if !vars.Exists(requiredVar) {
			missingVars = append(missingVars, requiredVar)
			continue
		}

		// Check for empty vars
		m := vars.Get(requiredVar)
		if len(m.Static) == 0 {
			emptyVars = append(emptyVars, requiredVar)
			continue
		}

		// Get the valid values from the filter
		if t.RequiresStrict.LimitValues != nil || len(t.RequiresStrict.LimitValues) > 0 {
			if values, found := t.RequiresStrict.LimitValues[requiredVar]; found {
				if !slices.Contains(values, m.Static) {
					failedLimitVars = append(failedLimitVars, requiredVar)
				}
			}
		}

	}

	if len(missingVars) > 0 {
		return &errors.TaskMissingRequiredVars{
			TaskName:    t.Name(),
			MissingVars: missingVars,
		}
	}

	if len(emptyVars) > 0 {
		return &errors.TaskRequiredStrictVarsEmpty{
			TaskName:    t.Name(),
			InvalidVars: emptyVars,
		}
	}

	if len(failedLimitVars) > 0 {
		return &errors.TaskRequiredStrictVarsLimitFail{
			TaskName:    t.Name(),
			InvalidVars: failedLimitVars,
		}
	}

	return nil
}
