package v2

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/go-task/task/internal/compiler"
	"github.com/go-task/task/internal/execext"
	"github.com/go-task/task/internal/logger"
	"github.com/go-task/task/internal/taskfile"
	"github.com/go-task/task/internal/templater"
)

var _ compiler.Compiler = &CompilerV2{}

type CompilerV2 struct {
	Dir  string
	Vars taskfile.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

// GetVariables returns fully resolved variables following the priority order:
// 1. Task variables
// 2. Call variables
// 3. Taskvars file variables
// 4. Environment variables
func (c *CompilerV2) GetVariables(t *taskfile.Task, call taskfile.Call) (taskfile.Vars, error) {
	vr := varResolver{c: c, vars: compiler.GetEnviron()}
	vr.merge(c.Vars)
	vr.merge(c.Vars)
	vr.merge(call.Vars)
	vr.merge(call.Vars)
	vr.merge(t.Vars)
	vr.merge(t.Vars)
	return vr.vars, vr.err
}

type varResolver struct {
	c    *CompilerV2
	vars taskfile.Vars
	err  error
}

func (vr *varResolver) merge(vars taskfile.Vars) {
	if vr.err != nil {
		return
	}
	tr := templater.Templater{Vars: vr.vars}
	for k, v := range vars {
		v = taskfile.Var{
			Static: tr.Replace(v.Static),
			Sh:     tr.Replace(v.Sh),
		}
		static, err := vr.c.HandleDynamicVar(v)
		if err != nil {
			vr.err = err
			return
		}
		vr.vars[k] = taskfile.Var{Static: static}
	}
	vr.err = tr.Err()
}

func (c *CompilerV2) HandleDynamicVar(v taskfile.Var) (string, error) {
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
	if err := execext.RunCommand(opts); err != nil {
		return "", fmt.Errorf(`task: Command "%s" in taskvars file failed: %s`, opts.Command, err)
	}

	// Trim a single trailing newline from the result to make most command
	// output easier to use in shell commands.
	result := strings.TrimSuffix(stdout.String(), "\n")

	c.dynamicCache[v.Sh] = result
	c.Logger.VerboseErrf(`task: dynamic variable: '%s' result: '%s'`, v.Sh, result)

	return result, nil
}
