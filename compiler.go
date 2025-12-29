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

func (c *Compiler) getVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
	// In scoped mode, OS env vars are in {{.env.XXX}} namespace, not at root
	// In legacy mode, they are at root level
	scopedMode := experiments.ScopedTaskfiles.Enabled() && t != nil && t.Location != nil && c.Graph != nil
	var result *ast.Vars
	if scopedMode {
		result = ast.NewVars()
	} else {
		result = env.GetEnviron()
	}

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
			// Replace values
			newVar := templater.ReplaceVar(v, cache)
			// If the variable should not be evaluated, but is nil, set it to an empty string
			// This stops empty interface errors when using the templater to replace values later
			// Preserve the Sh field so it can be displayed in summary
			if !evaluateShVars && newVar.Value == nil {
				result.Set(k, ast.Var{Value: "", Sh: newVar.Sh})
				return nil
			}
			// If the variable should not be evaluated and it is set, we can set it and return
			if !evaluateShVars {
				result.Set(k, ast.Var{Value: newVar.Value, Sh: newVar.Sh})
				return nil
			}
			// Now we can check for errors since we've handled all the cases when we don't want to evaluate
			if err := cache.Err(); err != nil {
				return err
			}
			// If the variable is already set, we can set it and return
			if newVar.Value != nil || newVar.Sh == nil {
				result.Set(k, ast.Var{Value: newVar.Value})
				return nil
			}
			// If the variable is dynamic, we need to resolve it first
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
		// NOTE(@andreynering): We're manually joining these paths here because
		// this is the raw task, not the compiled one.
		cache := &templater.Cache{Vars: result}
		dir := templater.Replace(t.Dir, cache)
		if err := cache.Err(); err != nil {
			return nil, err
		}
		dir = filepathext.SmartJoin(c.Dir, dir)
		taskRangeFunc = getRangeFunc(dir)
	}

	// When scoped includes is enabled, resolve vars from DAG instead of merged vars
	if scopedMode {
		// Get root Taskfile for inheritance (parent vars are always accessible)
		rootVertex, err := c.Graph.Root()
		if err != nil {
			return nil, err
		}

		// === ENV NAMESPACE ===
		// Create a separate map for environment variables
		// Accessible via {{.env.VAR}} in templates
		envMap := make(map[string]any)

		// 1. OS environment variables
		for _, e := range os.Environ() {
			k, v, _ := strings.Cut(e, "=")
			envMap[k] = v
		}

		// Helper to resolve env vars and add to envMap
		resolveEnvToMap := func(k string, v ast.Var, dir string) error {
			cache := &templater.Cache{Vars: result}
			newVar := templater.ReplaceVar(v, cache)
			if err := cache.Err(); err != nil {
				return err
			}
			// Static value
			if newVar.Value != nil || newVar.Sh == nil {
				if newVar.Value != nil {
					envMap[k] = newVar.Value
				}
				return nil
			}
			// Dynamic value (sh:)
			if evaluateShVars {
				// Build env slice for sh execution (includes envMap values)
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

		// 2. Root taskfile env
		for k, v := range rootVertex.Taskfile.Env.All() {
			if err := resolveEnvToMap(k, v, c.Dir); err != nil {
				return nil, err
			}
		}

		// === VARS (at root level) ===
		// Apply root vars
		for k, v := range rootVertex.Taskfile.Vars.All() {
			if err := rangeFunc(k, v); err != nil {
				return nil, err
			}
		}

		// If task is from an included Taskfile (not the root), get its vars from the DAG
		if t.Location.Taskfile != rootVertex.URI {
			includeVertex, err := c.Graph.Vertex(t.Location.Taskfile)
			if err != nil {
				return nil, err
			}
			// Apply include's env to envMap (overrides root's env)
			for k, v := range includeVertex.Taskfile.Env.All() {
				if err := resolveEnvToMap(k, v, filepathext.SmartJoin(c.Dir, t.Dir)); err != nil {
					return nil, err
				}
			}
			// Apply include's vars (overrides root's vars)
			for k, v := range includeVertex.Taskfile.Vars.All() {
				if err := taskRangeFunc(k, v); err != nil {
					return nil, err
				}
			}
		}

		// Apply IncludeVars (vars passed via includes: section)
		if t.IncludeVars != nil {
			for k, v := range t.IncludeVars.All() {
				if err := rangeFunc(k, v); err != nil {
					return nil, err
				}
			}
		}

		// Inject env namespace into result
		result.Set("env", ast.Var{Value: envMap})
	} else {
		// Legacy behavior: use merged vars
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
