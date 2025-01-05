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

	var missingVars []string
	var notAllowedValuesVars []errors.NotAllowedVar
	for _, requiredVar := range t.Requires.Vars {
		value, ok := t.Vars.Get(requiredVar.Name)
		if !ok {
			missingVars = append(missingVars, requiredVar.Name)
		} else {
			value, isString := value.Value.(string)
			if isString && requiredVar.Enum != nil && !slices.Contains(requiredVar.Enum, value) {
				notAllowedValuesVars = append(notAllowedValuesVars, errors.NotAllowedVar{
					Value: value,
					Enum:  requiredVar.Enum,
					Name:  requiredVar.Name,
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
