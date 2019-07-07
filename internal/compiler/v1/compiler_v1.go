package v1

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-task/task/v2/internal/compiler"
	"github.com/go-task/task/v2/internal/execext"
	"github.com/go-task/task/v2/internal/logger"
	"github.com/go-task/task/v2/internal/taskfile"
	"github.com/go-task/task/v2/internal/templater"
)

var _ compiler.Compiler = &CompilerV1{}

type CompilerV1 struct {
	Dir  string
	Vars taskfile.Vars

	Logger *logger.Logger

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

// GetVariables returns fully resolved variables following the priority order:
// 1. Call variables (should already have been resolved)
// 2. Environment (should not need to be resolved)
// 3. Task variables, resolved with access to:
//    - call, taskvars and environment variables
// 4. Taskvars variables, resolved with access to:
//    - environment variables
func (c *CompilerV1) GetVariables(t *taskfile.Task, call taskfile.Call) (taskfile.Vars, error) {
	merge := func(dest taskfile.Vars, srcs ...taskfile.Vars) {
		for _, src := range srcs {
			for k, v := range src {
				dest[k] = v
			}
		}
	}
	varsKeys := func(srcs ...taskfile.Vars) []string {
		m := make(map[string]struct{})
		for _, src := range srcs {
			for k := range src {
				m[k] = struct{}{}
			}
		}
		lst := make([]string, 0, len(m))
		for k := range m {
			lst = append(lst, k)
		}
		return lst
	}
	replaceVars := func(dest taskfile.Vars, keys []string) error {
		r := templater.Templater{Vars: dest}
		for _, k := range keys {
			v := dest[k]
			dest[k] = taskfile.Var{
				Static: r.Replace(v.Static),
				Sh:     r.Replace(v.Sh),
			}
		}
		return r.Err()
	}
	resolveShell := func(dest taskfile.Vars, keys []string) error {
		for _, k := range keys {
			v := dest[k]
			static, err := c.HandleDynamicVar(v)
			if err != nil {
				return err
			}
			dest[k] = taskfile.Var{Static: static}
		}
		return nil
	}
	update := func(dest taskfile.Vars, srcs ...taskfile.Vars) error {
		merge(dest, srcs...)
		// updatedKeys ensures template evaluation is run only once.
		updatedKeys := varsKeys(srcs...)
		if err := replaceVars(dest, updatedKeys); err != nil {
			return err
		}
		return resolveShell(dest, updatedKeys)
	}

	// Resolve taskvars variables to "result" with environment override variables.
	override := compiler.GetEnviron()
	result := make(taskfile.Vars, len(c.Vars)+len(t.Vars)+len(override))
	if err := update(result, c.Vars, override); err != nil {
		return nil, err
	}
	// Resolve task variables to "result" with environment and call override variables.
	merge(override, call.Vars)
	if err := update(result, t.Vars, override); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CompilerV1) HandleDynamicVar(v taskfile.Var) (string, error) {
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
