package task

import (
	"fmt"
	"slices"

	"github.com/elliotchance/orderedmap/v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/input"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/internal/term"
	"github.com/go-task/task/v3/taskfile/ast"
)

func (e *Executor) canPrompt() bool {
	if e.Prompter != nil {
		// The client asks, so no terminal is needed.
		return e.Interactive
	}
	return e.Interactive && (e.AssumeTerm || term.IsTerminal())
}

// askVar obtains a required variable that was not supplied: from the client
// when one can answer, and otherwise on the terminal as Task always has.
func (e *Executor) askVar(taskName string, v *ast.VarsWithValidation) (any, error) {
	if e.Prompter != nil {
		return e.Prompter.Ask(VarRequest{
			Task: taskName,
			Name: v.Name,
			Type: varTypeOf(v),
		})
	}
	if e.ownsScreen() {
		return nil, fmt.Errorf(
			"task: task %q needs a value for %q, and the client cannot ask for one",
			taskName, v.Name)
	}
	return e.newPrompter().Prompt(v.Name, getEnumValues(v.Enum))
}

func varTypeOf(v *ast.VarsWithValidation) VarType {
	if options := getEnumValues(v.Enum); len(options) > 0 {
		return EnumVar{Options: options}
	}
	return StringVar{}
}

// confirm asks whether to run a task that declares "prompt".
func (e *Executor) confirm(taskName, message string) (bool, error) {
	if e.Prompter != nil {
		return e.Prompter.Confirm(taskName, message)
	}
	if e.ownsScreen() {
		return false, fmt.Errorf(
			"task: task %q needs confirmation, and the client cannot ask for it", taskName)
	}
	err := e.Logger.Prompt(logger.Yellow, message, "n", "y", "yes")
	switch {
	case errors.Is(err, logger.ErrNoTerminal):
		return false, &errors.TaskCancelledNoTerminalError{TaskName: taskName}
	case errors.Is(err, logger.ErrPromptCancelled):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
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
	varsMap := orderedmap.NewOrderedMap[string, *ast.VarsWithValidation]()
	askedFor := make(map[string]string)

	var collect func(call *Call) error
	collect = func(call *Call) error {
		compiledTask, err := e.FastCompiledTask(call)
		if err != nil {
			return err
		}

		for _, v := range getMissingRequiredVars(compiledTask) {
			if !varsMap.Has(v.Name) {
				varsMap.Set(v.Name, resolveEnumRefForPrompt(v, compiledTask.Vars))
				// Remember who needed it first, so a client can say which task
				// it is asking on behalf of.
				askedFor[v.Name] = call.Task
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

	if varsMap.Len() == 0 {
		return nil
	}

	e.promptedVars = ast.NewVars()

	for v := range varsMap.Values() {
		value, err := e.askVar(askedFor[v.Name], v)
		if err != nil {
			if errors.Is(err, input.ErrCancelled) || errors.Is(err, ErrPromptCancelled) {
				return &errors.TaskCancelledByUserError{TaskName: askedFor[v.Name]}
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

	for _, v := range missing {
		value, err := e.askVar(t.Task, v)
		if err != nil {
			if errors.Is(err, input.ErrCancelled) || errors.Is(err, ErrPromptCancelled) {
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
			AllowedValues: getEnumValues(v.Enum),
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

		enumValues := getEnumValues(requiredVar.Enum)
		value, isString := varValue.Value.(string)
		if isString && len(enumValues) > 0 && !slices.Contains(enumValues, value) {
			notAllowedValuesVars = append(notAllowedValuesVars, errors.NotAllowedVar{
				Value: value,
				Enum:  enumValues,
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

func getEnumValues(e *ast.Enum) []string {
	if e == nil {
		return nil
	}
	return e.Value
}

// resolveEnumRefForPrompt returns a copy of v with its enum ref resolved into
// concrete values, so the interactive prompter can show a Select. Refs that
// depend on dynamic vars may not resolve here and fall back to free-form input.
func resolveEnumRefForPrompt(v *ast.VarsWithValidation, vars *ast.Vars) *ast.VarsWithValidation {
	if v.Enum == nil || v.Enum.Ref == "" || len(v.Enum.Value) > 0 {
		return v
	}
	vCopy := v.DeepCopy()
	cache := &templater.Cache{Vars: vars}
	_ = resolveEnumRefs(&ast.Requires{Vars: []*ast.VarsWithValidation{vCopy}}, cache)
	return vCopy
}
