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

	r := templater.Templater{Vars: vars}

	new := ast.Task{
		Task:                 origTask.Task,
		Label:                r.Replace(origTask.Label),
		Desc:                 r.Replace(origTask.Desc),
		Prompt:               r.Replace(origTask.Prompt),
		Summary:              r.Replace(origTask.Summary),
		Aliases:              origTask.Aliases,
		Sources:              r.ReplaceGlobs(origTask.Sources),
		Generates:            r.ReplaceGlobs(origTask.Generates),
		Dir:                  r.Replace(origTask.Dir),
		Set:                  origTask.Set,
		Shopt:                origTask.Shopt,
		Vars:                 nil,
		Env:                  nil,
		Dotenv:               r.ReplaceSlice(origTask.Dotenv),
		Silent:               origTask.Silent,
		Interactive:          origTask.Interactive,
		Internal:             origTask.Internal,
		Method:               r.Replace(origTask.Method),
		Prefix:               r.Replace(origTask.Prefix),
		IgnoreError:          origTask.IgnoreError,
		Run:                  r.Replace(origTask.Run),
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
	new.Env.Merge(r.ReplaceVars(e.Taskfile.Env))
	new.Env.Merge(r.ReplaceVars(dotenvEnvs))
	new.Env.Merge(r.ReplaceVars(origTask.Env))
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
				var keys []string
				var list []any
				// Get the list from the explicit for list
				if cmd.For.List != nil && len(cmd.For.List) > 0 {
					list = cmd.For.List
				}
				// Get the list from the task sources
				if cmd.For.From == "sources" {
					glist, err := fingerprint.Globs(new.Dir, new.Sources)
					if err != nil {
						return nil, err
					}
					// Make the paths relative to the task dir
					for i, v := range glist {
						if glist[i], err = filepath.Rel(new.Dir, v); err != nil {
							return nil, err
						}
					}
					list = asAnySlice(glist)
				}
				// Get the list from a variable and split it up
				if cmd.For.Var != "" {
					if vars != nil {
						v := vars.Get(cmd.For.Var)
						// If the variable is dynamic, then it hasn't been resolved yet
						// and we can't use it as a list. This happens when fast compiling a task
						// for use in --list or --list-all etc.
						if v.Value != nil && v.Sh == "" {
							switch value := v.Value.(type) {
							case string:
								if cmd.For.Split != "" {
									list = asAnySlice(strings.Split(value, cmd.For.Split))
								} else {
									list = asAnySlice(strings.Fields(value))
								}
							case []any:
								list = value
							case map[string]any:
								for k, v := range value {
									keys = append(keys, k)
									list = append(list, v)
								}
							default:
								return nil, errors.TaskfileInvalidError{
									URI: origTask.Location.Taskfile,
									Err: errors.New("var must be a delimiter-separated string or a list"),
								}
							}
						}
					}
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
					newCmd.Cmd = r.ReplaceWithExtra(cmd.Cmd, extra)
					newCmd.Task = r.ReplaceWithExtra(cmd.Task, extra)
					newCmd.Vars = r.ReplaceVarsWithExtra(cmd.Vars, extra)
					new.Cmds = append(new.Cmds, newCmd)
				}
				continue
			}
			newCmd := cmd.DeepCopy()
			newCmd.Cmd = r.Replace(cmd.Cmd)
			newCmd.Task = r.Replace(cmd.Task)
			newCmd.Vars = r.ReplaceVars(cmd.Vars)
			// Loop over the command's variables and resolve any references to other variables
			err := cmd.Vars.Range(func(k string, v ast.Var) error {
				if v.Ref != "" {
					refVal := vars.Get(v.Ref)
					newCmd.Vars.Set(k, refVal)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			new.Cmds = append(new.Cmds, newCmd)
		}
	}
	if len(origTask.Deps) > 0 {
		new.Deps = make([]*ast.Dep, 0, len(origTask.Deps))
		for _, dep := range origTask.Deps {
			if dep == nil {
				continue
			}
			newDep := dep.DeepCopy()
			newDep.Task = r.Replace(dep.Task)
			newDep.Vars = r.ReplaceVars(dep.Vars)
			// Loop over the dep's variables and resolve any references to other variables
			err := dep.Vars.Range(func(k string, v ast.Var) error {
				if v.Ref != "" {
					refVal := vars.Get(v.Ref)
					newDep.Vars.Set(k, refVal)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
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
			newPrecondition.Sh = r.Replace(precondition.Sh)
			newPrecondition.Msg = r.Replace(precondition.Msg)
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
		r.ResetCache()

		new.Status = r.ReplaceSlice(origTask.Status)
	}

	// We only care about templater errors if we are evaluating shell variables
	if evaluateShVars && r.Err() != nil {
		return &new, r.Err()
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
