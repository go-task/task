package task

import (
	"slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/taskfile/ast"
)

func (e *Executor) areTaskRequiredVarsSet(t *ast.Task) error {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return nil
	}

	var missingVars []errors.MissingVar
	for _, requiredVar := range t.Requires.Vars {
		_, ok := t.Vars.Get(requiredVar.Name)
		if !ok {
			missingVars = append(missingVars, errors.MissingVar{
				Name:          requiredVar.Name,
				AllowedValues: requiredVar.Enum,
			})
		}
	}

	if len(missingVars) > 0 {
		return &errors.TaskMissingRequiredVarsError{
			TaskName:    t.Name(),
			MissingVars: missingVars,
		}
	}

	return nil
}

func (e *Executor) areTaskRequiredVarsAllowedValuesSet(t *ast.Task) error {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return nil
	}

	var notAllowedValuesVars []errors.NotAllowedVar
	for _, requiredVar := range t.Requires.Vars {
		varValue, _ := t.Vars.Get(requiredVar.Name)

		value, isString := varValue.Value.(string)
		if isString && requiredVar.Enum != nil && !slices.Contains(requiredVar.Enum, value) {
			notAllowedValuesVars = append(notAllowedValuesVars, errors.NotAllowedVar{
				Value: value,
				Enum:  requiredVar.Enum,
				Name:  requiredVar.Name,
			})
		}

	}

	if len(notAllowedValuesVars) > 0 {
		return &errors.TaskNotAllowedVarsError{
			TaskName:       t.Name(),
			NotAllowedVars: notAllowedValuesVars,
		}
	}

	return nil
}
