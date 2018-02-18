package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/go-task/task/internal/compiler"
	compilerv1 "github.com/go-task/task/internal/compiler/v1"
	compilerv2 "github.com/go-task/task/internal/compiler/v2"
	"github.com/go-task/task/internal/execext"
	"github.com/go-task/task/internal/logger"
	"github.com/go-task/task/internal/taskfile"

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
	Taskfile *taskfile.Taskfile
	Dir      string
	Force    bool
	Watch    bool
	Verbose  bool
	Silent   bool

	Context context.Context

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Logger   *logger.Logger
	Compiler compiler.Compiler

	taskvars taskfile.Vars

	taskCallCount map[string]*int32
}

// Run runs Task
func (e *Executor) Run(calls ...taskfile.Call) error {
	if err := e.setup(); err != nil {
		return err
	}

	// check if given tasks exist
	for _, c := range calls {
		if _, ok := e.Taskfile.Tasks[c.Task]; !ok {
			// FIXME: move to the main package
			e.PrintTasksHelp()
			return &taskNotFoundError{taskName: c.Task}
		}
	}

	if e.Watch {
		return e.watchTasks(calls...)
	}

	for _, c := range calls {
		if err := e.RunTask(e.Context, c); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) setup() error {
	if e.Taskfile.Version == 0 {
		e.Taskfile.Version = 1
	}
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
	e.Logger = &logger.Logger{
		Stdout:  e.Stdout,
		Stderr:  e.Stderr,
		Verbose: e.Verbose,
	}
	switch e.Taskfile.Version {
	case 1:
		e.Compiler = &compilerv1.CompilerV1{
			Dir:    e.Dir,
			Vars:   e.taskvars,
			Logger: e.Logger,
		}
	case 2:
		e.Compiler = &compilerv2.CompilerV2{
			Dir:    e.Dir,
			Vars:   e.taskvars,
			Logger: e.Logger,
		}

		if !e.Silent {
			e.Logger.Errf(`task: warning: Taskfile "version: 2" is experimental and implementation can change before v2.0.0 release`)
		}
	default:
		return fmt.Errorf(`task: Unrecognized Taskfile version "%d"`, e.Taskfile.Version)
	}

	e.taskCallCount = make(map[string]*int32, len(e.Taskfile.Tasks))
	for k := range e.Taskfile.Tasks {
		e.taskCallCount[k] = new(int32)
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, call taskfile.Call) error {
	t, err := e.CompiledTask(call)
	if err != nil {
		return err
	}
	if !e.Watch && atomic.AddInt32(e.taskCallCount[call.Task], 1) >= MaximumTaskCall {
		return &MaximumTaskCallExceededError{task: call.Task}
	}

	if err := e.runDeps(ctx, t); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := isTaskUpToDate(ctx, t)
		if err != nil {
			return err
		}
		if upToDate {
			if !e.Silent {
				e.Logger.Errf(`task: Task "%s" is up to date`, t.Task)
			}
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, t, call, i); err != nil {
			if err2 := statusOnError(t); err2 != nil {
				e.Logger.VerboseErrf("task: error cleaning status on error: %v", err2)
			}
			return &taskRunError{t.Task, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, t *taskfile.Task) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, d := range t.Deps {
		d := d

		g.Go(func() error {
			return e.RunTask(ctx, taskfile.Call{Task: d.Task, Vars: d.Vars})
		})
	}

	return g.Wait()
}

func (e *Executor) runCommand(ctx context.Context, t *taskfile.Task, call taskfile.Call, i int) error {
	cmd := t.Cmds[i]

	if cmd.Cmd == "" {
		return e.RunTask(ctx, taskfile.Call{Task: cmd.Task, Vars: cmd.Vars})
	}

	if e.Verbose || (!cmd.Silent && !t.Silent && !e.Silent) {
		e.Logger.Errf(cmd.Cmd)
	}

	return execext.RunCommand(&execext.RunCommandOptions{
		Context: ctx,
		Command: cmd.Cmd,
		Dir:     t.Dir,
		Env:     getEnviron(t),
		Stdin:   e.Stdin,
		Stdout:  e.Stdout,
		Stderr:  e.Stderr,
	})
}

func getEnviron(t *taskfile.Task) []string {
	if t.Env == nil {
		return nil
	}

	envs := os.Environ()
	for k, v := range t.Env.ToStringMap() {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
