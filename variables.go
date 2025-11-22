package task

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// CompiledTask returns a copy of a task, but replacing variables in almost all
// properties using the Go template package.
func (e *Executor) CompiledTask(call *Call) (*ast.Task, error) {
	return e.compiledTask(call, true)
}

// FastCompiledTask is like CompiledTask, but it skippes dynamic variables.
func (e *Executor) FastCompiledTask(call *Call) (*ast.Task, error) {
	return e.compiledTask(call, false)
}

func (e *Executor) CompiledTaskForTaskList(call *Call) (*ast.Task, error) {
	origTask, err := e.GetTask(call)
	if err != nil {
		return nil, err
	}

	vars, err := e.Compiler.FastGetVariables(origTask, call)
	if err != nil {
		return nil, err
	}

	cache := &templater.Cache{Vars: vars}

	return &ast.Task{
		Task:                 origTask.Task,
		Label:                templater.Replace(origTask.Label, cache),
		Desc:                 templater.Replace(origTask.Desc, cache),
		Prompt:               templater.Replace(origTask.Prompt, cache),
		Summary:              templater.Replace(origTask.Summary, cache),
		Aliases:              origTask.Aliases,
		Sources:              origTask.Sources,
		Generates:            origTask.Generates,
		Dir:                  origTask.Dir,
		Set:                  origTask.Set,
		Shopt:                origTask.Shopt,
		Vars:                 vars,
		Env:                  nil,
		Dotenv:               origTask.Dotenv,
		Silent:               origTask.Silent,
		Interactive:          origTask.Interactive,
		Internal:             origTask.Internal,
		Method:               origTask.Method,
		Prefix:               origTask.Prefix,
		IgnoreError:          origTask.IgnoreError,
		Run:                  origTask.Run,
		IncludeVars:          origTask.IncludeVars,
		IncludedTaskfileVars: origTask.IncludedTaskfileVars,
		Platforms:            origTask.Platforms,
		Location:             origTask.Location,
		Requires:             origTask.Requires,
		Watch:                origTask.Watch,
		Namespace:            origTask.Namespace,
		Failfast:             origTask.Failfast,
	}, nil
}

func (e *Executor) compiledTask(call *Call, evaluateShVars bool) (*ast.Task, error) {
	origTask, err := e.GetTask(call)
	if err != nil {
		return nil, err
	}

	var vars *ast.Vars
	if evaluateShVars {
		vars, err = e.Compiler.GetVariables(origTask, call)
	} else {
		vars, err = e.Compiler.FastGetVariables(origTask, call)
	}
	if err != nil {
		return nil, err
	}
	fullName := origTask.Task
	if matches, exists := vars.Get("MATCH"); exists {
		for _, match := range matches.Value.([]string) {
			fullName = strings.Replace(fullName, "*", match, 1)
		}
	}

	cache := &templater.Cache{Vars: vars}
	new := ast.Task{
		Task:                 origTask.Task,
		Label:                templater.Replace(origTask.Label, cache),
		Desc:                 templater.Replace(origTask.Desc, cache),
		Prompt:               templater.Replace(origTask.Prompt, cache),
		Summary:              templater.Replace(origTask.Summary, cache),
		Aliases:              origTask.Aliases,
		Sources:              templater.ReplaceGlobs(origTask.Sources, cache),
		Generates:            templater.ReplaceGlobs(origTask.Generates, cache),
		Dir:                  templater.Replace(origTask.Dir, cache),
		Set:                  origTask.Set,
		Shopt:                origTask.Shopt,
		Vars:                 vars,
		Env:                  nil,
		Dotenv:               templater.Replace(origTask.Dotenv, cache),
		Silent:               origTask.Silent,
		Interactive:          origTask.Interactive,
		Internal:             origTask.Internal,
		Method:               templater.Replace(origTask.Method, cache),
		Prefix:               templater.Replace(origTask.Prefix, cache),
		IgnoreError:          origTask.IgnoreError,
		Run:                  templater.Replace(origTask.Run, cache),
		IncludeVars:          origTask.IncludeVars,
		IncludedTaskfileVars: origTask.IncludedTaskfileVars,
		Platforms:            origTask.Platforms,
		Location:             origTask.Location,
		Requires:             origTask.Requires,
		Watch:                origTask.Watch,
		Failfast:             origTask.Failfast,
		Namespace:            origTask.Namespace,
		FullName:             fullName,
	}
	new.Dir, err = execext.ExpandLiteral(new.Dir)
	if err != nil {
		return nil, err
	}
	if e.Dir != "" {
		new.Dir = filepathext.SmartJoin(e.Dir, new.Dir)
	}
	if new.Prefix == "" {
		new.Prefix = new.Task
	}

	dotenvEnvs := ast.NewVars()
	if len(new.Dotenv) > 0 {
		for _, dotEnvPath := range new.Dotenv {
			dotEnvPath = filepathext.SmartJoin(new.Dir, dotEnvPath)
			if _, err := os.Stat(dotEnvPath); os.IsNotExist(err) {
				continue
			}
			envs, err := godotenv.Read(dotEnvPath)
			if err != nil {
				return nil, err
			}
			for key, value := range envs {
				if _, ok := dotenvEnvs.Get(key); !ok {
					dotenvEnvs.Set(key, ast.Var{Value: value})
				}
			}
		}
	}

	new.Env = ast.NewVars()
	new.Env.Merge(templater.ReplaceVars(e.Taskfile.Env, cache), nil)
	new.Env.Merge(templater.ReplaceVars(dotenvEnvs, cache), nil)
	new.Env.Merge(templater.ReplaceVars(origTask.Env, cache), nil)
	if evaluateShVars {
		for k, v := range new.Env.All() {
			// If the variable is not dynamic, we can set it and return
			if v.Value != nil || v.Sh == nil {
				new.Env.Set(k, ast.Var{Value: v.Value})
				continue
			}
			static, err := e.Compiler.HandleDynamicVar(v, new.Dir, env.GetFromVars(new.Env))
			if err != nil {
				return nil, err
			}
			new.Env.Set(k, ast.Var{Value: static})
		}
	}

	if len(origTask.Sources) > 0 && origTask.Method != "none" {
		var checker fingerprint.SourcesCheckable

		if origTask.Method == "timestamp" {
			checker = fingerprint.NewTimestampChecker(e.TempDir.Fingerprint, e.Dry)
		} else {
			checker = fingerprint.NewChecksumChecker(e.TempDir.Fingerprint, e.Dry)
		}

		value, err := checker.Value(&new)
		if err != nil {
			return nil, err
		}
		vars.Set(strings.ToUpper(checker.Kind()), ast.Var{Live: value})

		// Adding new variables, requires us to refresh the templaters
		// cache of the the values manually
		cache.ResetCache()
	}

	if len(origTask.Cmds) > 0 {
		new.Cmds = make([]*ast.Cmd, 0, len(origTask.Cmds))
		for _, cmd := range origTask.Cmds {
			if cmd == nil {
				continue
			}
			if cmd.For != nil {
				list, keys, err := itemsFromFor(cmd.For, new.Dir, new.Sources, new.Generates, vars, origTask.Location, cache)
				if err != nil {
					return nil, err
				}
				// Name the iterator variable
				var as string
				if cmd.For.As != "" {
					as = cmd.For.As
				} else {
					as = "ITEM"
				}
				// Create a new command for each item in the list
				for i, loopValue := range list {
					extra := map[string]any{
						as: loopValue,
					}
					if len(keys) > 0 {
						extra["KEY"] = keys[i]
					}
					newCmd := cmd.DeepCopy()
					newCmd.Cmd = templater.ReplaceWithExtra(cmd.Cmd, cache, extra)
					newCmd.Task = templater.ReplaceWithExtra(cmd.Task, cache, extra)
					newCmd.Vars = templater.ReplaceVarsWithExtra(cmd.Vars, cache, extra)
					new.Cmds = append(new.Cmds, newCmd)
				}
				continue
			}
			// Defer commands are replaced in a lazy manner because
			// we need to include EXIT_CODE.
			if cmd.Defer {
				new.Cmds = append(new.Cmds, cmd.DeepCopy())
				continue
			}
			newCmd := cmd.DeepCopy()
			newCmd.Cmd = templater.Replace(cmd.Cmd, cache)
			newCmd.Task = templater.Replace(cmd.Task, cache)
			newCmd.Vars = templater.ReplaceVars(cmd.Vars, cache)
			new.Cmds = append(new.Cmds, newCmd)
		}
	}
	if len(origTask.Deps) > 0 {
		new.Deps = make([]*ast.Dep, 0, len(origTask.Deps))
		for _, dep := range origTask.Deps {
			if dep == nil {
				continue
			}
			if dep.For != nil {
				list, keys, err := itemsFromFor(dep.For, new.Dir, new.Sources, new.Generates, vars, origTask.Location, cache)
				if err != nil {
					return nil, err
				}
				// Name the iterator variable
				var as string
				if dep.For.As != "" {
					as = dep.For.As
				} else {
					as = "ITEM"
				}
				// Create a new command for each item in the list
				for i, loopValue := range list {
					extra := map[string]any{
						as: loopValue,
					}
					if len(keys) > 0 {
						extra["KEY"] = keys[i]
					}
					newDep := dep.DeepCopy()
					newDep.Task = templater.ReplaceWithExtra(dep.Task, cache, extra)
					newDep.Vars = templater.ReplaceVarsWithExtra(dep.Vars, cache, extra)
					new.Deps = append(new.Deps, newDep)
				}
				continue
			}
			newDep := dep.DeepCopy()
			newDep.Task = templater.Replace(dep.Task, cache)
			newDep.Vars = templater.ReplaceVars(dep.Vars, cache)
			new.Deps = append(new.Deps, newDep)
		}
	}

	if len(origTask.Preconditions) > 0 {
		new.Preconditions = make([]*ast.Precondition, 0, len(origTask.Preconditions))
		for _, precondition := range origTask.Preconditions {
			if precondition == nil {
				continue
			}
			newPrecondition := precondition.DeepCopy()
			newPrecondition.Sh = templater.Replace(precondition.Sh, cache)
			newPrecondition.Msg = templater.Replace(precondition.Msg, cache)
			new.Preconditions = append(new.Preconditions, newPrecondition)
		}
	}

	if len(origTask.Status) > 0 {
		new.Status = templater.Replace(origTask.Status, cache)
	}

	// We only care about templater errors if we are evaluating shell variables
	if evaluateShVars && cache.Err() != nil {
		return &new, cache.Err()
	}

	return &new, nil
}

func asAnySlice[T any](slice []T) []any {
	ret := make([]any, len(slice))
	for i, v := range slice {
		ret[i] = v
	}
	return ret
}

func itemsFromFor(
	f *ast.For,
	dir string,
	sources []*ast.Glob,
	generates []*ast.Glob,
	vars *ast.Vars,
	location *ast.Location,
	cache *templater.Cache,
) ([]any, []string, error) {
	var keys []string // The list of keys to loop over (only if looping over a map)
	var values []any  // The list of values to loop over
	// Get the list from a matrix
	if f.Matrix.Len() != 0 {
		if err := resolveMatrixRefs(f.Matrix, cache); err != nil {
			return nil, nil, errors.TaskfileInvalidError{
				URI: location.Taskfile,
				Err: err,
			}
		}
		return asAnySlice(product(f.Matrix)), nil, nil
	}
	// Get the list from the explicit for list
	if len(f.List) > 0 {
		return f.List, nil, nil
	}
	// Get the list from the task sources
	if f.From == "sources" {
		glist, err := fingerprint.Globs(dir, sources)
		if err != nil {
			return nil, nil, err
		}
		// Make the paths relative to the task dir
		for i, v := range glist {
			if glist[i], err = filepath.Rel(dir, v); err != nil {
				return nil, nil, err
			}
		}
		values = asAnySlice(glist)
	}
	// Get the list from the task generates
	if f.From == "generates" {
		glist, err := fingerprint.Globs(dir, generates)
		if err != nil {
			return nil, nil, err
		}
		// Make the paths relative to the task dir
		for i, v := range glist {
			if glist[i], err = filepath.Rel(dir, v); err != nil {
				return nil, nil, err
			}
		}
		values = asAnySlice(glist)
	}
	// Get the list from a variable and split it up
	if f.Var != "" {
		if vars != nil {
			v, ok := vars.Get(f.Var)
			// If the variable is dynamic, then it hasn't been resolved yet
			// and we can't use it as a list. This happens when fast compiling a task
			// for use in --list or --list-all etc.
			if ok && v.Value != nil && v.Sh == nil {
				switch value := v.Value.(type) {
				case string:
					if f.Split != "" {
						values = asAnySlice(strings.Split(value, f.Split))
					} else {
						values = asAnySlice(strings.Fields(value))
					}
				case []string:
					values = asAnySlice(value)
				case []int:
					values = asAnySlice(value)
				case []any:
					values = value
				case map[string]any:
					for k, v := range value {
						keys = append(keys, k)
						values = append(values, v)
					}
				default:
					return nil, nil, errors.TaskfileInvalidError{
						URI: location.Taskfile,
						Err: errors.New("loop var must be a delimiter-separated string, list or a map"),
					}
				}
			}
		}
	}
	return values, keys, nil
}

func resolveMatrixRefs(matrix *ast.Matrix, cache *templater.Cache) error {
	if matrix.Len() == 0 {
		return nil
	}
	for _, row := range matrix.All() {
		if row.Ref != "" {
			v := templater.ResolveRef(row.Ref, cache)
			switch value := v.(type) {
			case []any:
				row.Value = value
			default:
				return fmt.Errorf("matrix reference %q must resolve to a list", row.Ref)
			}
		}
	}
	return nil
}

// product generates the cartesian product of the input map of slices.
func product(matrix *ast.Matrix) []map[string]any {
	if matrix.Len() == 0 {
		return nil
	}

	// Start with an empty product result
	result := []map[string]any{{}}

	// Iterate over each slice in the slices
	for key, row := range matrix.All() {
		var newResult []map[string]any

		// For each combination in the current result
		for _, combination := range result {
			// Append each element from the current slice to the combinations
			for _, item := range row.Value {
				newComb := make(map[string]any, len(combination))
				// Copy the existing combination
				maps.Copy(newComb, combination)
				// Add the current item with the corresponding key
				newComb[key] = item
				newResult = append(newResult, newComb)
			}
		}

		// Update result with the new combinations
		result = newResult
	}

	return result
}
