package task

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v2/internal/execext"
	"github.com/go-task/task/v2/internal/taskfile"
	"github.com/go-task/task/v2/internal/templater"
)

// CompiledTask returns a copy of a task, but replacing variables in almost all
// properties using the Go template package.
func (e *Executor) CompiledTask(call taskfile.Call) (*taskfile.Task, error) {
	origTask, ok := e.Taskfile.Tasks[call.Task]
	if !ok {
		return nil, &taskNotFoundError{call.Task}
	}

	vars, err := e.Compiler.GetVariables(origTask, call)
	if err != nil {
		return nil, err
	}

	r := templater.Templater{Vars: vars}

	new := taskfile.Task{
		Task:        origTask.Task,
		Desc:        r.Replace(origTask.Desc),
		Sources:     r.ReplaceSlice(origTask.Sources),
		Generates:   r.ReplaceSlice(origTask.Generates),
		Dir:         r.Replace(origTask.Dir),
		Vars:        nil,
		Env:         nil,
		Silent:      origTask.Silent,
		Method:      r.Replace(origTask.Method),
		Prefix:      r.Replace(origTask.Prefix),
		IgnoreError: origTask.IgnoreError,
	}
	new.Dir, err = execext.Expand(new.Dir)
	if err != nil {
		return nil, err
	}
	if e.Dir != "" && !filepath.IsAbs(new.Dir) {
		new.Dir = filepath.Join(e.Dir, new.Dir)
	}
	if new.Prefix == "" {
		new.Prefix = new.Task
	}

	new.Env = make(taskfile.Vars, len(e.Taskfile.Env)+len(origTask.Env))
	for k, v := range r.ReplaceVars(e.Taskfile.Env) {
		new.Env[k] = v
	}
	for k, v := range r.ReplaceVars(origTask.Env) {
		new.Env[k] = v
	}
	for k, v := range new.Env {
		static, err := e.Compiler.HandleDynamicVar(v)
		if err != nil {
			return nil, err
		}
		new.Env[k] = taskfile.Var{Static: static}
	}

	if len(origTask.Status) > 0 {

		e := &Executor{
			Dir:    new.Dir,
			Stdout: ioutil.Discard,
			Stderr: ioutil.Discard,
			Dry:    true,
		}

		checker, err := e.GetStatusChecker(&new)
		if err != nil {
			return nil, err
		}

		value, err := checker.Value()
		if err != nil {
			return nil, err
		}

		vars[strings.ToUpper(checker.Kind())] = taskfile.Var{Static: value}

		statusTemplater := templater.Templater{Vars: vars}
		new.Status = statusTemplater.ReplaceSlice(origTask.Status)
	}

	if len(origTask.Cmds) > 0 {
		new.Cmds = make([]*taskfile.Cmd, len(origTask.Cmds))
		for i, cmd := range origTask.Cmds {
			new.Cmds[i] = &taskfile.Cmd{
				Task:        r.Replace(cmd.Task),
				Silent:      cmd.Silent,
				Cmd:         r.Replace(cmd.Cmd),
				Vars:        r.ReplaceVars(cmd.Vars),
				IgnoreError: cmd.IgnoreError,
			}
		}
	}
	if len(origTask.Deps) > 0 {
		new.Deps = make([]*taskfile.Dep, len(origTask.Deps))
		for i, dep := range origTask.Deps {
			new.Deps[i] = &taskfile.Dep{
				Task: r.Replace(dep.Task),
				Vars: r.ReplaceVars(dep.Vars),
			}
		}
	}

	if len(origTask.Preconditions) > 0 {
		new.Preconditions = make([]*taskfile.Precondition, len(origTask.Preconditions))
		for i, precond := range origTask.Preconditions {
			new.Preconditions[i] = &taskfile.Precondition{
				Sh:  r.Replace(precond.Sh),
				Msg: r.Replace(precond.Msg),
			}
		}
	}

	return &new, r.Err()
}
