package task

import (
	"slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/input"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

func (e *Executor) canPrompt() bool {
	return e.Interactive && (e.AssumeTerm || term.IsTerminal())
}

func (e *Executor) newPrompter() *input.Prompter {
	return &input.Prompter{
		Stdin:  e.Stdin,
		Stdout: e.Stdout,
		Stderr: e.Stderr,
	}
}

// promptDepsVars traverses the dependency tree, collects all missing required
// variables, and prompts for them upfront. This is used for deps which execute
// in parallel, so all prompts must happen before execution to avoid interleaving.
// Prompted values are stored in e.promptedVars for injection into task calls.
func (e *Executor) promptDepsVars(calls []*Call) error {
	if !e.canPrompt() {
		return nil
	}

	// Collect all missing vars from the dependency tree
	visited := make(map[string]bool)
	varsMap := make(map[string]*ast.VarsWithValidation)

	var collect func(call *Call) error
	collect = func(call *Call) error {
		compiledTask, err := e.FastCompiledTask(call)
		if err != nil {
			return err
		}

		for _, v := range getMissingRequiredVars(compiledTask) {
			if _, exists := varsMap[v.Name]; !exists {
				varsMap[v.Name] = v
			}
		}

		// Check visited AFTER collecting vars to handle duplicate task calls with different vars
		if visited[call.Task] {
			return nil
		}
		visited[call.Task] = true

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

	for _, call := range calls {
		if err := collect(call); err != nil {
			return err
		}
	}

	if len(varsMap) == 0 {
		return nil
	}

	prompter := e.newPrompter()
	e.promptedVars = ast.NewVars()

	for _, v := range varsMap {
		value, err := prompter.Prompt(v.Name, v.Enum)
		if err != nil {
			if errors.Is(err, input.ErrCancelled) {
				return &errors.TaskCancelledByUserError{TaskName: "interactive prompt"}
			}
			return err
		}
		e.promptedVars.Set(v.Name, ast.Var{Value: value})
	}

	return nil
}

// promptTaskVars prompts for any missing required vars from a single task.
// Used for sequential task calls (cmds) where we can prompt just-in-time.
// Returns true if any vars were prompted (caller should recompile the task).
func (e *Executor) promptTaskVars(t *ast.Task, call *Call) (bool, error) {
	if !e.canPrompt() || t.Requires == nil || len(t.Requires.Vars) == 0 {
		return false, nil
	}

	// Find missing vars, excluding already prompted ones
	var missing []*ast.VarsWithValidation
	for _, v := range getMissingRequiredVars(t) {
		if e.promptedVars != nil {
			if _, ok := e.promptedVars.Get(v.Name); ok {
				continue
			}
		}
		missing = append(missing, v)
	}

	if len(missing) == 0 {
		return false, nil
	}

	prompter := e.newPrompter()

	for _, v := range missing {
		value, err := prompter.Prompt(v.Name, v.Enum)
		if err != nil {
			if errors.Is(err, input.ErrCancelled) {
				return false, &errors.TaskCancelledByUserError{TaskName: t.Name()}
			}
			return false, err
		}

		// Add to call.Vars for recompilation
		if call.Vars == nil {
			call.Vars = ast.NewVars()
		}
		call.Vars.Set(v.Name, ast.Var{Value: value})

		// Cache for reuse by other tasks
		if e.promptedVars == nil {
			e.promptedVars = ast.NewVars()
		}
		e.promptedVars.Set(v.Name, ast.Var{Value: value})
	}

	return true, nil
}

// getMissingRequiredVars returns required vars that are not set in the task's vars.
func getMissingRequiredVars(t *ast.Task) []*ast.VarsWithValidation {
	if t.Requires == nil {
		return nil
	}
	var missing []*ast.VarsWithValidation
	for _, v := range t.Requires.Vars {
		if _, ok := t.Vars.Get(v.Name); !ok {
			missing = append(missing, v)
		}
	}
	return missing
}

func (e *Executor) areTaskRequiredVarsSet(t *ast.Task) error {
	missing := getMissingRequiredVars(t)
	if len(missing) == 0 {
		return nil
	}

	missingVars := make([]errors.MissingVar, len(missing))
	for i, v := range missing {
		missingVars[i] = errors.MissingVar{
			Name:          v.Name,
			AllowedValues: v.Enum,
		}
	}

	return &errors.TaskMissingRequiredVarsError{
		TaskName:    t.Name(),
		MissingVars: missingVars,
	}
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
