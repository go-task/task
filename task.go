package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-task/task/execext"

	"golang.org/x/sync/errgroup"
)

const (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"
	// MaximumTaskCall is the max number of times a task can be called.
	// This exists to prevent infinite loops on cyclic dependencies
	MaximumTaskCall = 100
)

// Executor executes a Taskfile
type Executor struct {
	Tasks   Tasks
	Dir     string
	Force   bool
	Watch   bool
	Verbose bool
	Silent  bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	taskvars Vars

	taskCallCount map[string]*int32

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex
}

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Cmds      []*Cmd
	Deps      []*Dep
	Desc      string
	Sources   []string
	Generates []string
	Status    []string
	Dir       string
	Vars      Vars
	Set       string
	Env       Vars
	Silent    bool
}

// Run runs Task
func (e *Executor) Run(args ...string) error {
	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}

	e.taskCallCount = make(map[string]*int32, len(e.Tasks))
	for k := range e.Tasks {
		e.taskCallCount[k] = new(int32)
	}

	if e.dynamicCache == nil {
		e.dynamicCache = make(map[string]string, 10)
	}

	// check if given tasks exist
	for _, a := range args {
		if _, ok := e.Tasks[a]; !ok {
			// FIXME: move to the main package
			e.PrintTasksHelp()
			return &taskNotFoundError{taskName: a}
		}
	}

	if e.Watch {
		if err := e.watchTasks(args...); err != nil {
			return err
		}
		return nil
	}

	for _, a := range args {
		if err := e.RunTask(context.Background(), Call{Task: a, Vars: nil}); err != nil {
			return err
		}
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call Call) error {
	origTask, ok := e.Tasks[call.Task]
	if !ok {
		return &taskNotFoundError{call.Task}
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[call.Task], 1) >= MaximumTaskCall {
		return &MaximumTaskCallExceededError{task: call.Task}
	}

	vars, err := e.getVariables(call)
	if err != nil {
		return err
	}

	t, err := origTask.CompiledTask(e.Dir, vars)
	if err != nil {
		return err
	}

	if err := e.runDeps(ctx, t); err != nil {
		return err
	}

	// FIXME: doing again, since a var may have been overriden
	// using the `set:` attribute of a dependecy.
	// Remove this when `set` (that is deprecated) be removed
	vars, err = e.getVariables(call)
	if err != nil {
		return err
	}
	t, err = origTask.CompiledTask(e.Dir, vars)
	if err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := t.isUpToDate(ctx)
		if err != nil {
			return err
		}
		if upToDate {
			e.printfln(`task: Task "%s" is up to date`, call.Task)
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, t, call, i); err != nil {
			return &taskRunError{call.Task, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, t *Task) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, d := range t.Deps {
		d := d

		g.Go(func() error {
			return e.RunTask(ctx, Call{Task: d.Task, Vars: d.Vars})
		})
	}

	return g.Wait()
}

func (e *Executor) runCommand(ctx context.Context, t *Task, call Call, i int) error {
	cmd := t.Cmds[i]

	if cmd.Cmd == "" {
		return e.RunTask(ctx, Call{Task: cmd.Task, Vars: cmd.Vars})
	}

	opts := &execext.RunCommandOptions{
		Context: ctx,
		Command: cmd.Cmd,
		Dir:     t.Dir,
		Env:     t.getEnviron(),
		Stdin:   e.Stdin,
		Stderr:  e.Stderr,
	}

	if e.Verbose || (!cmd.Silent && !t.Silent && !e.Silent) {
		e.println(cmd.Cmd)
	}
	if t.Set != "" {
		var stdout bytes.Buffer
		opts.Stdout = &stdout
		if err := execext.RunCommand(opts); err != nil {
			return err
		}
		return os.Setenv(t.Set, strings.TrimSpace(stdout.String()))
	}

	opts.Stdout = e.Stdout
	return execext.RunCommand(opts)
}

func (t *Task) getEnviron() []string {
	if t.Env == nil {
		return nil
	}

	envs := os.Environ()
	for k, v := range t.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
