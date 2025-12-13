package task

import (
	"slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/prompt"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

// promptForInteractiveVars prompts the user for any missing interactive variables
// and injects them into the call's Vars. It returns true if any variables were
// prompted for (meaning the task needs to be recompiled).
func (e *Executor) promptForInteractiveVars(t *ast.Task, call *Call) (bool, error) {
	if t.Requires == nil || len(t.Requires.Vars) == 0 {
		return false, nil
	}

	// Don't prompt if NoTTY is set or we're not in a terminal
	if e.NoTTY || (!e.AssumeTerm && !term.IsTerminal()) {
		return false, nil
	}

	prompter := prompt.New()
	var prompted bool

	for _, requiredVar := range t.Requires.Vars {
		// Skip non-interactive vars
		if !requiredVar.Interactive {
			continue
		}

		// Skip if already set
		if _, ok := t.Vars.Get(requiredVar.Name); ok {
			continue
		}

		var value string
		var err error

		if len(requiredVar.Enum) > 0 {
			value, err = prompter.Select(requiredVar.Name, requiredVar.Enum)
		} else {
			value, err = prompter.Text(requiredVar.Name)
		}

		if err != nil {
			if errors.Is(err, prompt.ErrCancelled) {
				return false, &errors.TaskCancelledByUserError{TaskName: call.Task}
			}
			return false, err
		}

		// Inject into call.Vars
		if call.Vars == nil {
			call.Vars = ast.NewVars()
		}
		call.Vars.Set(requiredVar.Name, ast.Var{Value: value})
		prompted = true
	}

	return prompted, nil
}

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
