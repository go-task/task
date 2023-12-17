package v3

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile"
)

var _ compiler.Compiler = &CompilerV3{}

type CompilerV3 struct {
	Dir            string
	UserWorkingDir string

	TaskfileEnv  *taskfile.Vars
	TaskfileVars *taskfile.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

func (c *CompilerV3) GetTaskfileVariables() (*taskfile.Vars, error) {
	return c.getVariables(nil, nil, true)
}

func (c *CompilerV3) GetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	return c.getVariables(t, &call, true)
}

func (c *CompilerV3) FastGetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	return c.getVariables(t, &call, false)
}

func (c *CompilerV3) getVariables(t *taskfile.Task, call *taskfile.Call, evaluateShVars bool) (*taskfile.Vars, error) {
	result := compiler.GetEnviron()
	if t != nil {
		specialVars, err := c.getSpecialVars(t)
		if err != nil {
			return nil, err
		}
		for k, v := range specialVars {
			result.Set(k, taskfile.Var{Value: v})
		}
	}

	getRangeFunc := func(dir string) func(k string, v taskfile.Var) error {
		return func(k string, v taskfile.Var) error {
			tr := templater.Templater{Vars: result, RemoveNoValue: true}
			// Replace values
			newVar := taskfile.Var{}
			switch value := v.Value.(type) {
			case string:
				newVar.Value = tr.Replace(value)
			default:
				newVar.Value = value
			}
			newVar.Sh = tr.Replace(v.Sh)
			newVar.Dir = v.Dir
			if err := tr.Err(); err != nil {
				return err
			}
			// If the variable should not be evaluated, but is nil, set it to an empty string
			// This stops empty interface errors when using the templater to replace values later
			if !evaluateShVars && newVar.Value == nil {
				result.Set(k, taskfile.Var{Value: ""})
				return nil
			}
			// If the variable is not dynamic, we can set it and return
			if !evaluateShVars || newVar.Value != nil || newVar.Sh == "" {
				result.Set(k, taskfile.Var{Value: newVar.Value})
				return nil
			}
			// If the variable is dynamic, we need to resolve it first
			static, err := c.HandleDynamicVar(newVar, dir)
			if err != nil {
				return err
			}
			result.Set(k, taskfile.Var{Value: static})
			return nil
		}
	}
	rangeFunc := getRangeFunc(c.Dir)

	var taskRangeFunc func(k string, v taskfile.Var) error
	if t != nil {
		// NOTE(@andreynering): We're manually joining these paths here because
		// this is the raw task, not the compiled one.
		tr := templater.Templater{Vars: result, RemoveNoValue: true}
		dir := tr.Replace(t.Dir)
		if err := tr.Err(); err != nil {
			return nil, err
		}
		dir = filepathext.SmartJoin(c.Dir, dir)
		taskRangeFunc = getRangeFunc(dir)
	}

	if err := c.TaskfileEnv.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := c.TaskfileVars.Range(rangeFunc); err != nil {
		return nil, err
	}
	if t != nil {
		if err := t.IncludedTaskfileVars.Range(taskRangeFunc); err != nil {
			return nil, err
		}
		if err := t.IncludeVars.Range(rangeFunc); err != nil {
			return nil, err
		}
	}

	if t == nil || call == nil {
		return result, nil
	}

	if err := call.Vars.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := t.Vars.Range(taskRangeFunc); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *CompilerV3) HandleDynamicVar(v taskfile.Var, dir string) (string, error) {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	if c.dynamicCache == nil {
		c.dynamicCache = make(map[string]string, 30)
	}
	if result, ok := c.dynamicCache[v.Sh]; ok {
		return result, nil
	}

	// NOTE(@andreynering): If a var have a specific dir, use this instead
	if v.Dir != "" {
		dir = v.Dir
	}

	var stdout bytes.Buffer
	opts := &execext.RunCommandOptions{
		Command: v.Sh,
		Dir:     dir,
		Stdout:  &stdout,
		Stderr:  c.Logger.Stderr,
	}
	if err := execext.RunCommand(context.Background(), opts); err != nil {
		return "", fmt.Errorf(`task: Command "%s" failed: %s`, opts.Command, err)
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\r\n")
	result = strings.TrimSuffix(result, "\n")

	c.dynamicCache[v.Sh] = result
	c.Logger.VerboseErrf(logger.Magenta, "task: dynamic variable: %q result: %q\n", v.Sh, result)

	return result, nil
}

// ResetCache clear the dymanic variables cache
func (c *CompilerV3) ResetCache() {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	c.dynamicCache = nil
}

func (c *CompilerV3) getSpecialVars(t *taskfile.Task) (map[string]string, error) {
	taskfileDir, err := c.getTaskfileDir(t)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"TASK":             t.Task,
		"ROOT_DIR":         c.Dir,
		"TASKFILE_DIR":     taskfileDir,
		"USER_WORKING_DIR": c.UserWorkingDir,
		"TASK_VERSION":     version.GetVersion(),
	}, nil
}

func (c *CompilerV3) getTaskfileDir(t *taskfile.Task) (string, error) {
	if t.IncludedTaskfile != nil {
		return t.IncludedTaskfile.FullDirPath()
	}
	return c.Dir, nil
}
