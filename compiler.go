package task

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-task/task/v3/experiments"
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
	CLIVars      *ast.Vars // CLI vars passed via command line (e.g., task foo VAR=value)
	Graph        *ast.TaskfileGraph

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

func (c *Compiler) GetTaskfileVariables() (*ast.Vars, error) {
	return c.getVariables(nil, nil, true)
}

func (c *Compiler) GetVariables(t *ast.Task, call *Call) (*ast.Vars, error) {
	return c.getVariables(t, call, true)
}

func (c *Compiler) FastGetVariables(t *ast.Task, call *Call) (*ast.Vars, error) {
	return c.getVariables(t, call, false)
}

// isScopedMode returns true if scoped variable resolution should be used.
// Scoped mode requires the experiment to be enabled, a task with location info, and a graph.
func (c *Compiler) isScopedMode(t *ast.Task) bool {
	return experiments.ScopedTaskfiles.Enabled() &&
		t != nil &&
		t.Location != nil &&
		c.Graph != nil
}

func (c *Compiler) getVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
	if c.isScopedMode(t) {
		return c.getScopedVariables(t, call, evaluateShVars)
	}
	return c.getLegacyVariables(t, call, evaluateShVars)
}

// getScopedVariables resolves variables in scoped mode.
// In scoped mode:
// - OS env vars are in {{.env.XXX}} namespace, not at root
// - Variables from sibling includes are isolated
//
// Variable resolution order (lowest to highest priority):
// 1. Root Taskfile vars
// 2. Include Taskfile vars
// 3. Include passthrough vars (includes: name: vars:)
// 4. Task vars
// 5. Call vars
// 6. CLI vars
func (c *Compiler) getScopedVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
	result := ast.NewVars()

	specialVars, err := c.getSpecialVars(t, call)
	if err != nil {
		return nil, err
	}
	for k, v := range specialVars {
		result.Set(k, ast.Var{Value: v})
	}

	getRangeFunc := func(dir string) func(k string, v ast.Var) error {
		return func(k string, v ast.Var) error {
			cache := &templater.Cache{Vars: result}
			newVar := templater.ReplaceVar(v, cache)
			if !evaluateShVars && newVar.Value == nil {
				result.Set(k, ast.Var{Value: "", Sh: newVar.Sh})
				return nil
			}
			if !evaluateShVars {
				result.Set(k, ast.Var{Value: newVar.Value, Sh: newVar.Sh})
				return nil
			}
			if err := cache.Err(); err != nil {
				return err
			}
			if newVar.Value != nil || newVar.Sh == nil {
				result.Set(k, ast.Var{Value: newVar.Value})
				return nil
			}
			static, err := c.HandleDynamicVar(newVar, dir, env.GetFromVars(result))
			if err != nil {
				return err
			}
			result.Set(k, ast.Var{Value: static})
			return nil
		}
	}
	rangeFunc := getRangeFunc(c.Dir)

	var taskRangeFunc func(k string, v ast.Var) error
	if t != nil {
		cache := &templater.Cache{Vars: result}
		dir := templater.Replace(t.Dir, cache)
		if err := cache.Err(); err != nil {
			return nil, err
		}
		dir = filepathext.SmartJoin(c.Dir, dir)
		taskRangeFunc = getRangeFunc(dir)
	}

	rootVertex, err := c.Graph.Root()
	if err != nil {
		return nil, err
	}

	envMap := make(map[string]any)
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		envMap[k] = v
	}

	resolveEnvToMap := func(k string, v ast.Var, dir string) error {
		cache := &templater.Cache{Vars: result}
		newVar := templater.ReplaceVar(v, cache)
		if err := cache.Err(); err != nil {
			return err
		}
		if newVar.Value != nil || newVar.Sh == nil {
			if newVar.Value != nil {
				envMap[k] = newVar.Value
			}
			return nil
		}
		if evaluateShVars {
			envSlice := os.Environ()
			for ek, ev := range envMap {
				if s, ok := ev.(string); ok {
					envSlice = append(envSlice, fmt.Sprintf("%s=%s", ek, s))
				}
			}
			static, err := c.HandleDynamicVar(newVar, dir, envSlice)
			if err != nil {
				return err
			}
			envMap[k] = static
		}
		return nil
	}

	for k, v := range rootVertex.Taskfile.Env.All() {
		if err := resolveEnvToMap(k, v, c.Dir); err != nil {
			return nil, err
		}
	}

	for k, v := range rootVertex.Taskfile.Vars.All() {
		if err := rangeFunc(k, v); err != nil {
			return nil, err
		}
	}

	if t.Location.Taskfile != rootVertex.URI {
		predecessorMap, err := c.Graph.PredecessorMap()
		if err != nil {
			return nil, err
		}

		var parentChain []*ast.TaskfileVertex
		currentURI := t.Location.Taskfile
		for {
			edges := predecessorMap[currentURI]
			if len(edges) == 0 {
				break
			}
			var parentURI string
			for _, edge := range edges {
				parentURI = edge.Source
				break
			}
			if parentURI == rootVertex.URI {
				break
			}
			parentVertex, err := c.Graph.Vertex(parentURI)
			if err != nil {
				return nil, err
			}
			parentChain = append([]*ast.TaskfileVertex{parentVertex}, parentChain...)
			currentURI = parentURI
		}

		for _, parent := range parentChain {
			parentDir := filepath.Dir(parent.URI)
			for k, v := range parent.Taskfile.Env.All() {
				if err := resolveEnvToMap(k, v, parentDir); err != nil {
					return nil, err
				}
			}
			// Vars use the parent's directory too
			parentRangeFunc := getRangeFunc(parentDir)
			for k, v := range parent.Taskfile.Vars.All() {
				if err := parentRangeFunc(k, v); err != nil {
					return nil, err
				}
			}
		}

		includeVertex, err := c.Graph.Vertex(t.Location.Taskfile)
		if err != nil {
			return nil, err
		}
		includeDir := filepath.Dir(includeVertex.URI)
		for k, v := range includeVertex.Taskfile.Env.All() {
			if err := resolveEnvToMap(k, v, includeDir); err != nil {
				return nil, err
			}
		}
		includeRangeFunc := getRangeFunc(includeDir)
		for k, v := range includeVertex.Taskfile.Vars.All() {
			if err := includeRangeFunc(k, v); err != nil {
				return nil, err
			}
		}
	}

	if t.IncludeVars != nil {
		for k, v := range t.IncludeVars.All() {
			if err := rangeFunc(k, v); err != nil {
				return nil, err
			}
		}
	}

	if call != nil {
		for k, v := range t.Vars.All() {
			if err := taskRangeFunc(k, v); err != nil {
				return nil, err
			}
		}
		for k, v := range call.Vars.All() {
			if err := taskRangeFunc(k, v); err != nil {
				return nil, err
			}
		}
	}

	for k, v := range c.CLIVars.All() {
		if err := rangeFunc(k, v); err != nil {
			return nil, err
		}
	}

	result.Set("env", ast.Var{Value: envMap})

	return result, nil
}

// getLegacyVariables resolves variables in legacy mode.
// In legacy mode, all variables (including OS env) are merged at root level.
func (c *Compiler) getLegacyVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
	result := env.GetEnviron()

	specialVars, err := c.getSpecialVars(t, call)
	if err != nil {
		return nil, err
	}
	for k, v := range specialVars {
		result.Set(k, ast.Var{Value: v})
	}

	getRangeFunc := func(dir string) func(k string, v ast.Var) error {
		return func(k string, v ast.Var) error {
			cache := &templater.Cache{Vars: result}
			newVar := templater.ReplaceVar(v, cache)
			if !evaluateShVars && newVar.Value == nil {
				result.Set(k, ast.Var{Value: "", Sh: newVar.Sh})
				return nil
			}
			if !evaluateShVars {
				result.Set(k, ast.Var{Value: newVar.Value, Sh: newVar.Sh})
				return nil
			}
			if err := cache.Err(); err != nil {
				return err
			}
			if newVar.Value != nil || newVar.Sh == nil {
				result.Set(k, ast.Var{Value: newVar.Value})
				return nil
			}
			static, err := c.HandleDynamicVar(newVar, dir, env.GetFromVars(result))
			if err != nil {
				return err
			}
			result.Set(k, ast.Var{Value: static})
			return nil
		}
	}
	rangeFunc := getRangeFunc(c.Dir)

	var taskRangeFunc func(k string, v ast.Var) error
	if t != nil {
		cache := &templater.Cache{Vars: result}
		dir := templater.Replace(t.Dir, cache)
		if err := cache.Err(); err != nil {
			return nil, err
		}
		dir = filepathext.SmartJoin(c.Dir, dir)
		taskRangeFunc = getRangeFunc(dir)
	}

	for k, v := range c.TaskfileEnv.All() {
		if err := rangeFunc(k, v); err != nil {
			return nil, err
		}
	}
	for k, v := range c.TaskfileVars.All() {
		if err := rangeFunc(k, v); err != nil {
			return nil, err
		}
	}

	if t != nil {
		for k, v := range t.IncludeVars.All() {
			if err := rangeFunc(k, v); err != nil {
				return nil, err
			}
		}
		for k, v := range t.IncludedTaskfileVars.All() {
			if err := taskRangeFunc(k, v); err != nil {
				return nil, err
			}
		}
	}

	if t == nil || call == nil {
		return result, nil
	}

	for k, v := range call.Vars.All() {
		if err := rangeFunc(k, v); err != nil {
			return nil, err
		}
	}
	for k, v := range t.Vars.All() {
		if err := taskRangeFunc(k, v); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *Compiler) HandleDynamicVar(v ast.Var, dir string, e []string) (string, error) {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

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

func (c *Compiler) getSpecialVars(t *ast.Task, call *Call) (map[string]string, error) {
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

	return allVars, nil
}
