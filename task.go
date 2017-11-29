package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/go-task/task/internal/execext"

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

	Context context.Context

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	taskvars Vars

	taskCallCount map[string]*int32

	dynamicCache   map[string]string
	muDynamicCache sync.Mutex

	RunTasksFunc func (e *Executor, calls ...Call) error
	RunTaskFunc func (e *Executor, ctx context.Context, call Call) error // TODO ctx as first param
	RunDepsFunc func (e *Executor, ctx context.Context, t *Task) error // TODO ctx as first param
	RunCommandFunc func (e *Executor, ctx context.Context, t *Task, call Call, i int) error // TODO ctx as first param
}

// RunDeps calls the default implementaion on how to run dependencies for a task
func RunDeps(e *Executor, ctx context.Context, t *Task) error {
	return e.runDeps(ctx, t)
}

// RunTasks calls the default implementation on how to execute multiple tasks
func RunTasks(e *Executor, calls ...Call) error {
	for _, c := range calls {
		if err := e.RunTaskFunc(e, e.Context, c); err != nil {
			return err
		}
	}
	return nil
}

// RunTask calls the default implementation on how to execute a single task
func RunTask(e *Executor, ctx context.Context, call Call) error {
	return e.RunTask(ctx, call)
}

// RunCommand calls the default function for running a command
func RunCommand(e *Executor, ctx context.Context, t *Task, call Call, i int) error {
	return e.runCommand(ctx, t, call, i)
}

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task      string
	Cmds      []*Cmd
	Deps      []*Dep
	Desc      string
	Sources   []string
	Generates []string
	Status    []string
	Dir       string
	Vars      Vars
	Env       Vars
	Silent    bool
	Method    string
}

// Run runs Task
func (e *Executor) Run(calls ...Call) error {
	if e.Context == nil {
		e.Context = context.Background()
	}
	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}

	if e.RunDepsFunc == nil {
		e.RunDepsFunc = RunDeps
	}

	if e.RunTasksFunc == nil {
		e.RunTasksFunc = RunTasks
	}

	if e.RunTaskFunc == nil {
		e.RunTaskFunc = RunTask
	}

	if e.RunCommandFunc == nil {
		e.RunCommandFunc = RunCommand
	}

	e.taskCallCount = make(map[string]*int32, len(e.Tasks))
	for k := range e.Tasks {
		e.taskCallCount[k] = new(int32)
	}

	if e.dynamicCache == nil {
		e.dynamicCache = make(map[string]string, 10)
	}

	// check if given tasks exist
	for _, c := range calls {
		if _, ok := e.Tasks[c.Task]; !ok {
			// FIXME: move to the main package
			e.PrintTasksHelp()
			return &taskNotFoundError{taskName: c.Task}
		}
	}

	if e.Watch {
		return e.watchTasks(calls...)
	}

	return e.RunTasksFunc(e, calls...)
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call Call) error {
	t, err := e.CompiledTask(call)
	if err != nil {
		return err
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[call.Task], 1) >= MaximumTaskCall {
		return &MaximumTaskCallExceededError{task: call.Task}
	}

	if err := e.RunDepsFunc(e, ctx, t); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := t.isUpToDate(ctx)
		if err != nil {
			return err
		}
		if upToDate {
			if !e.Silent {
				e.errf(`task: Task "%s" is up to date`, t.Task)
			}
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.RunCommandFunc(e, ctx, t, call, i); err != nil {
			if err2 := t.statusOnError(); err2 != nil {
				e.verboseErrf("task: error cleaning status on error: %v", err2)
			}
			return &taskRunError{t.Task, err}
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

	if e.Verbose || (!cmd.Silent && !t.Silent && !e.Silent) {
		e.errf(cmd.Cmd)
	}

	return execext.RunCommand(&execext.RunCommandOptions{
		Context: ctx,
		Command: cmd.Cmd,
		Dir:     t.Dir,
		Env:     t.getEnviron(),
		Stdin:   e.Stdin,
		Stdout:  e.Stdout,
		Stderr:  e.Stderr,
	})
}

func (t *Task) getEnviron() []string {
	if t.Env == nil {
		return nil
	}

	envs := os.Environ()
	for k, v := range t.Env.toStringMap() {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
