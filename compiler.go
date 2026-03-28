package task

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile/ast"
)

type Compiler struct {
	Dir            string
	Entrypoint     string
	UserWorkingDir string

	TaskfileEnv  *ast.Vars
	TaskfileVars *ast.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

type mergeProc func() error

type mergeVars func() *ast.Vars

type mergeItem struct {
	name string    // Name of the merge item, for logging.
	cond bool      // Indicates if this mergeItem should be processed.
	vars mergeVars // Variables to be merged (overwrite existing).
	dir  *string   // Directory used when evaluating variables.
	proc mergeProc // Called to modify state between merge items.
}

var (
	enableDebug = os.Getenv("TASK_DEBUG_COMPILER")
	entryOsEnv  = env.GetEnviron()
)

func (c *Compiler) logf(s string, args ...any) {
	if enableDebug != "" {
		c.Logger.VerboseErrf(logger.Grey, s, args...)
	}
}

func (c *Compiler) GetTaskfileVariables() (*ast.Vars, error) {
	c.logf("GetTaskfileVariables:\n")
	return c.getVariables(nil, nil, true)
}

func (c *Compiler) GetVariables(t *ast.Task, call *Call) (*ast.Vars, error) {
	c.logf("GetVariables: task=%s, call=%s\n",
		func() string {
			if t == nil {
				return "<nil>"
			}
			return t.Name()
		}(),
		func() string {
			if call == nil {
				return "<nil>"
			}
			return call.Task
		}(),
	)
	return c.getVariables(t, call, true)
}

func (c *Compiler) FastGetVariables(t *ast.Task, call *Call) (*ast.Vars, error) {
	c.logf("FastGetVariables: task=%s, call=%s\n",
		func() string {
			if t == nil {
				return "<nil>"
			}
			return t.Name()
		}(),
		func() string {
			if call == nil {
				return "<nil>"
			}
			return call.Task
		}(),
	)
	return c.getVariables(t, call, false)
}

func (c *Compiler) resolveAndSetVar(result *ast.Vars, k string, v ast.Var, dir string, evaluateSh bool) error {
	cache := &templater.Cache{Vars: result}
	newVar := templater.ReplaceVar(v, cache)

	set := func(key string, value ast.Var) {
		result.Set(key, value)
		if enableDebug != "" {
			_v, found := entryOsEnv.Get(key)
			if !found || !reflect.DeepEqual(_v, value) {
				valStr := fmt.Sprintf("%v", value.Value)
				if strings.Contains(valStr, "\n") {
					indent := strings.Repeat(" ", 6)
					valStr = strings.ReplaceAll("\n"+valStr, "\n", "\n"+indent)
				}
				c.logf("    %s <-- %v\n", k, valStr)
			}
		}
	}

	// Templating only (no shell evaluation).
	if !evaluateSh {
		if newVar.Value == nil {
			// If the variable should not be evaluated, but is nil, set it to an empty string.
			newVar.Value = ""
		}
		set(k, ast.Var{Value: newVar.Value, Sh: newVar.Sh})
		return nil
	}
	// Check cache error condition before continuing.
	if err := cache.Err(); err != nil {
		return err
	}
	// Variable already set, use use its value.
	if newVar.Value != nil || newVar.Sh == nil {
		set(k, ast.Var{Value: newVar.Value})
		return nil
	}
	// Resolve the variable.
	c.logf("    --> %s  (v.Dir=%s, dir=%s)\n", *newVar.Sh, newVar.Dir, dir)
	if static, err := c.HandleDynamicVar(newVar, dir, env.GetFromVars(result)); err == nil {
		set(k, ast.Var{Value: static})
	} else {
		return err
	}
	return nil
}

func (c *Compiler) mergeVars(dest *ast.Vars, source *ast.Vars, dir string, evaluateShVars bool) error {
	if source == nil || dest == nil {
		return nil
	}
	for k, v := range source.All() {
		if err := c.resolveAndSetVar(dest, k, v, dir, evaluateShVars); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) getVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
	result := ast.NewVars()
	taskdir := ""
	taskOnly := (t != nil)
	taskCall := (t != nil && call != nil)

	processMergeItem := func(items []mergeItem) error {
		for _, m := range items {
			if m.proc != nil {
				if err := m.proc(); err != nil {
					return err
				}
			}
			if !m.cond {
				continue
			}
			c.logf("  compiler: variable merge: %s\n", m.name)
			if m.vars == nil {
				continue
			}
			dir := c.Dir
			if m.dir != nil {
				dir = *m.dir
			}
			evalSh := evaluateShVars
			if m.name == "SpecialVars" || m.name == "OS.Env" {
				evalSh = false
			}
			if err := c.mergeVars(result, m.vars(), dir, evalSh); err != nil {
				return err
			}
		}
		return nil
	}
	updateTaskdir := func() error {
		if t != nil {
			cache := &templater.Cache{Vars: result}
			dir := templater.Replace(t.Dir, cache)
			if err := cache.Err(); err != nil {
				return err
			}
			taskdir = filepathext.SmartJoin(c.Dir, dir)
		}
		return nil
	}
	resolveGlobalVarRefs := func() error {
		return nil
	}

	if err := processMergeItem([]mergeItem{
		{"OS.Env", true, func() *ast.Vars { return env.GetEnviron() }, nil, nil},
		{"SpecialVars", true, func() *ast.Vars { return c.getSpecialVars(t, call) }, nil, nil},
		{proc: updateTaskdir},
		{"Taskfile.Env", true, func() *ast.Vars { return c.TaskfileEnv }, nil, nil},
		{"Taskfile.Vars", true, func() *ast.Vars { return c.TaskfileVars }, nil, nil},
		{proc: resolveGlobalVarRefs},
		{"Inc.Vars", taskOnly, func() *ast.Vars { return t.IncludeVars }, nil, nil},
		{"IncTaskfile.Vars", taskOnly, func() *ast.Vars { return t.IncludedTaskfileVars }, &taskdir, nil},
		{"Call.Vars", taskCall, func() *ast.Vars { return call.Vars }, nil, nil},
		{"Task.Vars", taskCall, func() *ast.Vars { return t.Vars }, &taskdir, nil},
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Compiler) HandleDynamicVar(v ast.Var, dir string, e []string) (string, error) {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	// If the variable is not dynamic or it is empty, return an empty string
	if v.Sh == nil || *v.Sh == "" {
		return "", nil
	}

	if c.dynamicCache == nil {
		c.dynamicCache = make(map[string]string, 30)
	}
	if result, ok := c.dynamicCache[*v.Sh]; ok {
		return result, nil
	}

	// NOTE(@andreynering): If a var have a specific dir, use this instead
	if v.Dir != "" {
		dir = v.Dir
	}

	var stdout bytes.Buffer
	opts := &execext.RunCommandOptions{
		Command: *v.Sh,
		Dir:     dir,
		Stdout:  &stdout,
		Stderr:  c.Logger.Stderr,
		Env:     e,
	}
	if err := execext.RunCommand(context.Background(), opts); err != nil {
		return "", fmt.Errorf(`task: Command "%s" failed: %s`, opts.Command, err)
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\r\n")
	result = strings.TrimSuffix(result, "\n")

	c.dynamicCache[*v.Sh] = result
	c.Logger.VerboseErrf(logger.Magenta, "task: dynamic variable: %q result: %q\n", *v.Sh, result)

	return result, nil
}

// ResetCache clear the dynamic variables cache
func (c *Compiler) ResetCache() {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	c.dynamicCache = nil
}

func (c *Compiler) getSpecialVars(t *ast.Task, call *Call) *ast.Vars {
	allVars := map[string]string{
		"TASK_EXE":         filepath.ToSlash(os.Args[0]),
		"ROOT_TASKFILE":    filepathext.SmartJoin(c.Dir, c.Entrypoint),
		"ROOT_DIR":         c.Dir,
		"USER_WORKING_DIR": c.UserWorkingDir,
		"TASK_VERSION":     version.GetVersion(),
	}
	if t != nil {
		allVars["TASK"] = t.Task
		allVars["TASK_DIR"] = filepathext.SmartJoin(c.Dir, t.Dir)
		allVars["TASKFILE"] = t.Location.Taskfile
		allVars["TASKFILE_DIR"] = filepath.Dir(t.Location.Taskfile)
	} else {
		allVars["TASK"] = ""
		allVars["TASK_DIR"] = ""
		allVars["TASKFILE"] = ""
		allVars["TASKFILE_DIR"] = ""
	}
	if call != nil {
		allVars["ALIAS"] = call.Task
	} else {
		allVars["ALIAS"] = ""
	}

	vars := ast.NewVars()
	for k, v := range allVars {
		vars.Set(k, ast.Var{Value: v})
	}
	return vars
}
