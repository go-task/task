package task

import (
	"strings"

	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/status"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

// CompiledTask returns a copy of a task, but replacing variables in almost all
// properties using the Go template package.
func (e *Executor) CompiledTask(call taskfile.Call) (*taskfile.Task, error) {
	return e.compiledTask(call, true)
}

// FastCompiledTask is like CompiledTask, but it skippes dynamic variables.
func (e *Executor) FastCompiledTask(call taskfile.Call) (*taskfile.Task, error) {
	return e.compiledTask(call, false)
}

func (e *Executor) compiledTask(call taskfile.Call, evaluateShVars bool) (*taskfile.Task, error) {
	origTask, err := e.GetTask(call)
	if err != nil {
		return nil, err
	}

	var vars *taskfile.Vars
	if evaluateShVars {
		vars, err = e.Compiler.GetVariables(origTask, call)
	} else {
		vars, err = e.Compiler.FastGetVariables(origTask, call)
	}
	if err != nil {
		return nil, err
	}

	v, err := e.Taskfile.ParsedVersion()
	if err != nil {
		return nil, err
	}

	r := templater.Templater{Vars: vars, RemoveNoValue: v >= 3.0}

	new := taskfile.Task{
		Task:                 origTask.Task,
		Label:                r.Replace(origTask.Label),
		Desc:                 r.Replace(origTask.Desc),
		Summary:              r.Replace(origTask.Summary),
		Aliases:              origTask.Aliases,
		Sources:              r.ReplaceSlice(origTask.Sources),
		Generates:            r.ReplaceSlice(origTask.Generates),
		Dir:                  r.Replace(origTask.Dir),
		Vars:                 nil,
		Env:                  nil,
		Silent:               origTask.Silent,
		Interactive:          origTask.Interactive,
		Internal:             origTask.Internal,
		Method:               r.Replace(origTask.Method),
		Prefix:               r.Replace(origTask.Prefix),
		IgnoreError:          origTask.IgnoreError,
		Run:                  r.Replace(origTask.Run),
		IncludeVars:          origTask.IncludeVars,
		IncludedTaskfileVars: origTask.IncludedTaskfileVars,
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

	new.Env = &taskfile.Vars{}
	new.Env.Merge(r.ReplaceVars(e.Taskfile.Env))
	new.Env.Merge(r.ReplaceVars(origTask.Env))
	if evaluateShVars {
		err = new.Env.Range(func(k string, v taskfile.Var) error {
			static, err := e.Compiler.HandleDynamicVar(v, new.Dir)
			if err != nil {
				return err
			}
			new.Env.Set(k, taskfile.Var{Static: static})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(origTask.Cmds) > 0 {
		new.Cmds = make([]*taskfile.Cmd, 0, len(origTask.Cmds))
		for _, cmd := range origTask.Cmds {
			if cmd == nil {
				continue
			}
			new.Cmds = append(new.Cmds, &taskfile.Cmd{
				Task:        r.Replace(cmd.Task),
				Silent:      cmd.Silent,
				Cmd:         r.Replace(cmd.Cmd),
				Vars:        r.ReplaceVars(cmd.Vars),
				IgnoreError: cmd.IgnoreError,
				Defer:       cmd.Defer,
			})
		}
	}
	if len(origTask.Deps) > 0 {
		new.Deps = make([]*taskfile.Dep, 0, len(origTask.Deps))
		for _, dep := range origTask.Deps {
			if dep == nil {
				continue
			}
			new.Deps = append(new.Deps, &taskfile.Dep{
				Task: r.Replace(dep.Task),
				Vars: r.ReplaceVars(dep.Vars),
			})
		}
	}

	if len(origTask.Preconditions) > 0 {
		new.Preconditions = make([]*taskfile.Precondition, 0, len(origTask.Preconditions))
		for _, precond := range origTask.Preconditions {
			if precond == nil {
				continue
			}
			new.Preconditions = append(new.Preconditions, &taskfile.Precondition{
				Sh:  r.Replace(precond.Sh),
				Msg: r.Replace(precond.Msg),
			})
		}
	}

	if len(origTask.Status) > 0 {
		for _, checker := range []status.Checker{e.timestampChecker(&new), e.checksumChecker(&new)} {
			value, err := checker.Value()
			if err != nil {
				return nil, err
			}
			vars.Set(strings.ToUpper(checker.Kind()), taskfile.Var{Live: value})
		}

		// Adding new variables, requires us to refresh the templaters
		// cache of the the values manually
		r.ResetCache()

		new.Status = r.ReplaceSlice(origTask.Status)
	}

	return &new, r.Err()
}
