package v3

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/taskfile"
	"github.com/go-task/task/v3/internal/templater"
)

var _ compiler.Compiler = &CompilerV3{}

type CompilerV3 struct {
	Dir string

	TaskfileVars *taskfile.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

func (c *CompilerV3) GetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	result := compiler.GetEnviron()
	result.Set("TASK", taskfile.Var{Static: t.Task})

	rangeFunc := func(k string, v taskfile.Var) error {
		tr := templater.Templater{Vars: result, RemoveNoValue: true}
		v = taskfile.Var{
			Static: tr.Replace(v.Static),
			Sh:     tr.Replace(v.Sh),
		}
		if err := tr.Err(); err != nil {
			return err
		}
		static, err := c.HandleDynamicVar(v)
		if err != nil {
			return err
		}
		result.Set(k, taskfile.Var{Static: static})
		return nil
	}

	if err := c.TaskfileVars.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := call.Vars.Range(rangeFunc); err != nil {
		return nil, err
	}
	if err := t.Vars.Range(rangeFunc); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *CompilerV3) HandleDynamicVar(v taskfile.Var) (string, error) {
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

	var stdout bytes.Buffer
	opts := &execext.RunCommandOptions{
		Command: v.Sh,
		Dir:     c.Dir,
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
