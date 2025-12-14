package task

import (
	"slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/prompt"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

// collectAllRequiredVars traverses the dependency tree of all calls and collects
// all required variables that are missing. Returns a deduplicated list.
func (e *Executor) collectAllRequiredVars(calls []*Call) ([]*ast.VarsWithValidation, error) {
	visited := make(map[string]bool)
	varsMap := make(map[string]*ast.VarsWithValidation)

	var collect func(call *Call) error
	collect = func(call *Call) error {
		// Always compile to resolve variables (also fetches the task)
		compiledTask, err := e.FastCompiledTask(call)
		if err != nil {
			return err
		}

		// Always collect required vars from this task
		if compiledTask.Requires != nil {
			for _, v := range compiledTask.Requires.Vars {
				// Check if var is already set
				if _, ok := compiledTask.Vars.Get(v.Name); !ok {
					// Add to map if not already there
					if _, exists := varsMap[v.Name]; !exists {
						varsMap[v.Name] = v
					}
				}
			}
		}

		// Only skip recursion if already visited (to avoid infinite loops)
		// We already collected the vars above, so we're good
		if visited[call.Task] {
			return nil
		}
		visited[call.Task] = true

		// Recurse into deps
		for _, dep := range compiledTask.Deps {
			depCall := &Call{
				Task:   dep.Task,
				Vars:   dep.Vars,
				Silent: dep.Silent,
			}
			if err := collect(depCall); err != nil {
				return err
			}
		}

		return nil
	}

	// Collect from all initial calls
	for _, call := range calls {
		if err := collect(call); err != nil {
			return nil, err
		}
	}

	// Convert map to slice
	result := make([]*ast.VarsWithValidation, 0, len(varsMap))
	for _, v := range varsMap {
		result = append(result, v)
	}

	return result, nil
}

// promptForAllVars prompts for all the given variables at once and returns
// a Vars object with all the values.
func (e *Executor) promptForAllVars(vars []*ast.VarsWithValidation) (*ast.Vars, error) {
	if len(vars) == 0 {
		return nil, nil
	}

	// Don't prompt if interactive mode is disabled
	if !e.Interactive {
		return nil, nil
	}

	// Don't prompt if NoTTY is set or we're not in a terminal
	if e.NoTTY || (!e.AssumeTerm && !term.IsTerminal()) {
		return nil, nil
	}

	prompter := &prompt.Prompter{
		Stdin:  e.Stdin,
		Stdout: e.Stdout,
		Stderr: e.Stderr,
	}

	result := ast.NewVars()

	for _, v := range vars {
		var value string
		var err error

		if len(v.Enum) > 0 {
			value, err = prompter.Select(v.Name, v.Enum)
		} else {
			value, err = prompter.Text(v.Name)
		}

		if err != nil {
			if errors.Is(err, prompt.ErrCancelled) {
				return nil, &errors.TaskCancelledByUserError{TaskName: "interactive prompt"}
			}
			return nil, err
		}

		result.Set(v.Name, ast.Var{Value: value})
	}

	return result, nil
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
