package v2

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

var _ compiler.Compiler = &CompilerV2{}

type CompilerV2 struct {
	Dir string

	Taskvars     *taskfile.Vars
	TaskfileVars *taskfile.Vars

	Expansions int

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

func (c *CompilerV2) GetTaskfileVariables() (*taskfile.Vars, error) {
	return &taskfile.Vars{}, nil
}

// FastGetVariables is a no-op on v2
func (c *CompilerV2) FastGetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	return c.GetVariables(t, call)
}

// GetVariables returns fully resolved variables following the priority order:
// 1. Task variables
// 2. Call variables
// 3. Taskfile variables
// 4. Taskvars file variables
// 5. Environment variables
func (c *CompilerV2) GetVariables(t *taskfile.Task, call taskfile.Call) (*taskfile.Vars, error) {
	vr := varResolver{
		c:    c,
		vars: compiler.GetEnviron(),
	}
	vr.vars.Set("TASK", taskfile.Var{Static: t.Task})

	for _, vars := range []*taskfile.Vars{c.Taskvars, c.TaskfileVars, call.Vars, t.Vars} {
		for i := 0; i < c.Expansions; i++ {
			vr.merge(vars)
		}
	}
	return vr.vars, vr.err
}

type varResolver struct {
	c    *CompilerV2
	vars *taskfile.Vars
	err  error
}

func (vr *varResolver) merge(vars *taskfile.Vars) {
	if vr.err != nil {
		return
	}
	tr := templater.Templater{Vars: vr.vars}
	_ = vars.Range(func(k string, v taskfile.Var) error {
		v = taskfile.Var{
			Static: tr.Replace(v.Static),
			Sh:     tr.Replace(v.Sh),
		}
		static, err := vr.c.HandleDynamicVar(v, "")
		if err != nil {
			vr.err = err
			return err
		}
		vr.vars.Set(k, taskfile.Var{Static: static})
		return nil
	})
	vr.err = tr.Err()
}

func (c *CompilerV2) HandleDynamicVar(v taskfile.Var, _ string) (string, error) {
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
		Stdout:  &stdout,
		Stderr:  c.Logger.Stderr,
	}
	if err := execext.RunCommand(context.Background(), opts); err != nil {
		return "", fmt.Errorf(`task: Command "%s" failed: %s`, opts.Command, err)
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\n")

	c.dynamicCache[v.Sh] = result
	c.Logger.VerboseErrf(logger.Magenta, "task: dynamic variable: %q result: %q\n", v.Sh, result)

	return result, nil
}

// ResetCache clear the dymanic variables cache
func (c *CompilerV2) ResetCache() {
	c.muDynamicCache.Lock()
	defer c.muDynamicCache.Unlock()

	c.dynamicCache = nil
}
