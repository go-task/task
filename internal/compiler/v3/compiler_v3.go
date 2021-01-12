package v3

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

var _ compiler.Compiler = &CompilerV3{}

type CompilerV3 struct {
	Dir string

	TaskfileEnv  *taskfile.Vars
	TaskfileVars *taskfile.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

func (c *CompilerV3) GetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	result := compiler.GetEnviron()
	result.Set("TASK", taskfile.Var{Static: t.Task})

	getRangeFunc := func(dir string) func(k string, v taskfile.Var) error {
		return func(k string, v taskfile.Var) error {
			tr := templater.Templater{Vars: result, RemoveNoValue: true}

			v = taskfile.Var{
				Static: tr.Replace(v.Static),
				Sh:     tr.Replace(v.Sh),
				Dir:    v.Dir,
			}
			if err := tr.Err(); err != nil {
				return err
			}
			static, err := c.HandleDynamicVar(v, dir)
			if err != nil {
				return err
			}
			result.Set(k, taskfile.Var{Static: static})
			return nil
		}
	}
	rangeFunc := getRangeFunc(c.Dir)

	if err := c.TaskfileEnv.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := c.TaskfileVars.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := call.Vars.Range(rangeFunc); err != nil {
		return nil, err
	}

	// NOTE(@andreynering): We're manually joining these paths here because
	// this is the raw task, not the compiled one.
	tr := templater.Templater{Vars: result, RemoveNoValue: true}
	dir := tr.Replace(t.Dir)
	if err := tr.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(c.Dir, dir)
	}
	taskRangeFunc := getRangeFunc(dir)

	if err := t.Vars.Range(taskRangeFunc); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *CompilerV3) HandleDynamicVar(v taskfile.Var, dir string) (string, error) {
	if v.Static != "" || v.Sh == "" {
		return v.Static, nil
	}

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
		return "", fmt.Errorf(`task: Command "%s" in taskvars file failed: %s`, opts.Command, err)
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\n")

	c.dynamicCache[v.Sh] = result
	c.Logger.VerboseErrf(logger.Magenta, `task: dynamic variable: '%s' result: '%s'`, v.Sh, result)

	return result, nil
}

// ResetCache clear the dymanic variables cache
func (c *CompilerV3) ResetCache() {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	c.dynamicCache = nil
}
