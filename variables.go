package task

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// CompiledTask returns a copy of a task, but replacing variables in almost all
// properties using the Go template package.
func (e *Executor) CompiledTask(call *ast.Call) (*ast.Task, error) {
	return e.compiledTask(call, true)
}

// FastCompiledTask is like CompiledTask, but it skippes dynamic variables.
func (e *Executor) FastCompiledTask(call *ast.Call) (*ast.Task, error) {
	return e.compiledTask(call, false)
}

func (e *Executor) compiledTask(call *ast.Call, evaluateShVars bool) (*ast.Task, error) {
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
		Vars:                 nil,
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
	}
	new.Dir, err = execext.Expand(new.Dir)
	if err != nil {
		return nil, err
	}
	if e.Dir != "" {
		new.Dir = filepathext.SmartJoin(e.Dir, new.Dir)
	}
	if new.Prefix == "" {
		new.Prefix = new.Task
	}

	dotenvEnvs := &ast.Vars{}
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
				if ok := dotenvEnvs.Exists(key); !ok {
					dotenvEnvs.Set(key, ast.Var{Value: value})
				}
			}
		}
	}

	new.Env = &ast.Vars{}
	new.Env.Merge(templater.ReplaceVars(e.Taskfile.Env, cache), nil)
	new.Env.Merge(templater.ReplaceVars(dotenvEnvs, cache), nil)
	new.Env.Merge(templater.ReplaceVars(origTask.Env, cache), nil)
	if evaluateShVars {
		err = new.Env.Range(func(k string, v ast.Var) error {
			// If the variable is not dynamic, we can set it and return
			if v.Value != nil || v.Sh == "" {
				new.Env.Set(k, ast.Var{Value: v.Value})
				return nil
			}
			static, err := e.Compiler.HandleDynamicVar(v, new.Dir)
			if err != nil {
				return err
			}
			new.Env.Set(k, ast.Var{Value: static})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(origTask.Cmds) > 0 {
		new.Cmds = make([]*ast.Cmd, 0, len(origTask.Cmds))
		for _, cmd := range origTask.Cmds {
			if cmd == nil {
				continue
			}
			if cmd.For != nil {
				list, keys, err := itemsFromFor(cmd.For, new.Dir, new.Sources, vars, origTask.Location)
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
				list, keys, err := itemsFromFor(dep.For, new.Dir, new.Sources, vars, origTask.Location)
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
		timestampChecker := fingerprint.NewTimestampChecker(e.TempDir, e.Dry)
		checksumChecker := fingerprint.NewChecksumChecker(e.TempDir, e.Dry)

		for _, checker := range []fingerprint.SourcesCheckable{timestampChecker, checksumChecker} {
			value, err := checker.Value(&new)
			if err != nil {
				return nil, err
			}
			vars.Set(strings.ToUpper(checker.Kind()), ast.Var{Live: value})
		}

		// Adding new variables, requires us to refresh the templaters
		// cache of the the values manually
		cache.ResetCache()

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
	vars *ast.Vars,
	location *ast.Location,
) ([]any, []string, error) {
	var keys []string // The list of keys to loop over (only if looping over a map)
	var values []any  // The list of values to loop over
	// Get the list from the explicit for list
	if f.List != nil && len(f.List) > 0 {
		values = f.List
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
	// Get the list from a variable and split it up
	if f.Var != "" {
		if vars != nil {
			v := vars.Get(f.Var)
			// If the variable is dynamic, then it hasn't been resolved yet
			// and we can't use it as a list. This happens when fast compiling a task
			// for use in --list or --list-all etc.
			if v.Value != nil && v.Sh == "" {
				switch value := v.Value.(type) {
				case string:
					if f.Split != "" {
						values = asAnySlice(strings.Split(value, f.Split))
					} else {
						values = asAnySlice(strings.Fields(value))
					}
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
