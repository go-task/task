package task

import (
	"slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile/ast"
)

func (e *Executor) areTaskRequiredVarsSet(t *ast.Task, call *ast.Call) error {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return nil
	}

	vars, err := e.Compiler.GetVariables(t, call)
	if err != nil {
		return err
	}

	var missingVars []string
	var notAllowedValuesVars []errors.NotAllowedVar
	for _, requiredVar := range t.Requires.Vars {
		value, isString := vars.Get(requiredVar.Name).Value.(string)
		if !vars.Exists(requiredVar.Name) {
			missingVars = append(missingVars, requiredVar.Name)
		} else {
			if isString && requiredVar.AllowedValues != nil && !slices.Contains(requiredVar.AllowedValues, value) {
				notAllowedValuesVars = append(notAllowedValuesVars, errors.NotAllowedVar{
					Value:         value,
					AllowedValues: requiredVar.AllowedValues,
					Name:          requiredVar.Name,
				})
			}
		}
	}

	if len(missingVars) > 0 {
		return &errors.TaskMissingRequiredVars{
			TaskName:    t.Name(),
			MissingVars: missingVars,
		}
	}

	if len(notAllowedValuesVars) > 0 {
		return &errors.TaskNotAllowedVars{
			TaskName:       t.Name(),
			NotAllowedVars: notAllowedValuesVars,
		}
	}

	return nil
}
